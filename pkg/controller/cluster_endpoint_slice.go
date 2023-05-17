// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/coalescing"
	"reflect"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
)

type endpointClusterReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r *endpointClusterReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(
		zap.String("name", req.NamespacedName.Name),
		zap.String("kind", "EgressClusterEndpointSlice"),
	)

	log.Info("reconcile")
	deleted := false
	policy := new(egressv1.EgressClusterGatewayPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	if deleted {
		return reconcile.Result{}, nil
	}

	pods, err := listPodsByClusterPolicy(ctx, r.client, policy)
	if err != nil {
		return reconcile.Result{}, err
	}

	podMap := make(map[types.NamespacedName]corev1.Pod)
	for _, pod := range pods {
		podMap[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = pod
	}

	endpointSlices, err := listClusterEndpointSlices(ctx, r.client, policy.Name)
	if err != nil {
		return reconcile.Result{}, err
	}

	existingKeyMap := make(map[types.NamespacedName]bool)
	slicesToUpdate := make([]egressv1.EgressClusterEndpointSlice, 0)
	slicesToCreate := make([]egressv1.EgressClusterEndpointSlice, 0)
	slicesToDelete := make([]egressv1.EgressClusterEndpointSlice, 0)
	slicesNotChange := make([]egressv1.EgressClusterEndpointSlice, 0)

	for _, epSlice := range endpointSlices.Items {
		needUpdate := false
		index := 0
		for i := 0; i < len(epSlice.Endpoints); i++ {
			ep := epSlice.Endpoints[i]
			key := types.NamespacedName{Namespace: ep.Namespace, Name: ep.Pod}
			if _, ok := podMap[key]; ok {
				existingKeyMap[key] = true
				epSlice.Endpoints[index] = epSlice.Endpoints[i]
				index = index + 1
			} else {
				needUpdate = true
			}
		}
		epSlice.Endpoints = epSlice.Endpoints[:index]
		if needUpdate {
			slicesToUpdate = append(slicesToUpdate, epSlice)
		} else {
			slicesNotChange = append(slicesNotChange, epSlice)
		}
	}

	needToCreateEp := make([]egressv1.EgressEndpoint, 0)

	for _, pod := range pods {
		key := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
		if _, ok := existingKeyMap[key]; !ok {
			if ep := newEndpoint(pod); ep != nil {
				needToCreateEp = append(needToCreateEp, *ep)
			}
		}
	}

	if len(needToCreateEp) > 0 {
		for i, slice := range slicesToUpdate {
			if len(slice.Endpoints) < r.config.FileConfig.MaxNumberEndpointPerSlice {
				count := r.config.FileConfig.MaxNumberEndpointPerSlice - len(slice.Endpoints)
				if count < len(needToCreateEp) {
					slicesToUpdate[i].Endpoints = append(slicesToUpdate[i].Endpoints, needToCreateEp[:count]...)
					needToCreateEp = needToCreateEp[count:]
				} else {
					slicesToUpdate[i].Endpoints = append(slicesToUpdate[i].Endpoints, needToCreateEp...)
					needToCreateEp = make([]egressv1.EgressEndpoint, 0)
					break
				}
			}
		}

		for _, slice := range slicesNotChange {
			if len(slice.Endpoints) < r.config.FileConfig.MaxNumberEndpointPerSlice {
				count := r.config.FileConfig.MaxNumberEndpointPerSlice - len(slice.Endpoints)
				if count < len(needToCreateEp) {
					slice.Endpoints = append(slice.Endpoints, needToCreateEp[:count]...)
					needToCreateEp = needToCreateEp[count:]
				} else {
					slice.Endpoints = append(slice.Endpoints, needToCreateEp...)
					needToCreateEp = make([]egressv1.EgressEndpoint, 0)
					break
				}
				slicesToUpdate = append(slicesToUpdate, slice)
			}
		}
	}

	for len(needToCreateEp) > 0 {
		epSlice := newClusterEndpointSlice(policy)
		if len(needToCreateEp) > r.config.FileConfig.MaxNumberEndpointPerSlice {
			// > 100
			tmp := needToCreateEp[:r.config.FileConfig.MaxNumberEndpointPerSlice]
			needToCreateEp = needToCreateEp[r.config.FileConfig.MaxNumberEndpointPerSlice:]
			epSlice.Endpoints = append(epSlice.Endpoints, tmp...)
			slicesToCreate = append(slicesToCreate, *epSlice)
		} else {
			// < 100
			epSlice.Endpoints = append(epSlice.Endpoints, needToCreateEp...)
			needToCreateEp = make([]egressv1.EgressEndpoint, 0)
			slicesToCreate = append(slicesToCreate, *epSlice)
		}
	}

	errs := make([]error, 0) // all errors generated in the process of reconciling

	for _, slice := range slicesToUpdate {
		if len(slice.Endpoints) == 0 {
			slicesToDelete = append(slicesToDelete, slice)
			continue
		}
		err := r.client.Update(ctx, &slice)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to update endpoint slice %v/%v: %v",
				slice.Namespace, slice.Name, err))
		}
	}

	for _, slice := range slicesToCreate {
		err := r.client.Create(ctx, &slice)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create endpoint slice %v/%v: %v",
				slice.Namespace, slice.Name, err))
		}
	}

	for _, slice := range slicesToDelete {
		err := r.client.Delete(ctx, &slice)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete endpoint slice %v/%v: %v",
				slice.Namespace, slice.Name, err))
		}
	}

	return reconcile.Result{}, utilerrors.NewAggregate(errs)
}

func newClusterEndpointSlice(policy *egressv1.EgressClusterGatewayPolicy) *egressv1.EgressClusterEndpointSlice {
	// TODO: change it on release v1
	gvk := schema.GroupVersionKind{
		Group:   "egressgateway.spidernet.io",
		Version: "v1beta1",
		Kind:    "EgressClusterGatewayPolicy",
	}
	ownerRef := metav1.NewControllerRef(policy, gvk)

	return &egressv1.EgressClusterEndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    getEndpointSlicePrefix(policy.Name),
			Namespace:       policy.Namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
			Labels: map[string]string{
				egressv1.LabelPolicyName: policy.Name,
			},
		},
	}
}

func listPodsByClusterPolicy(ctx context.Context, cli client.Client, policy *egressv1.EgressClusterGatewayPolicy) ([]corev1.Pod, error) {
	if policy.Spec.AppliedTo.NamespaceSelector == nil {
		pods := new(corev1.PodList)
		selector, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.PodSelector)
		if err != nil {
			return nil, err
		}
		opt := &client.ListOptions{
			LabelSelector: selector,
		}
		err = cli.List(ctx, pods, opt)
		if err != nil {
			return nil, err
		}
		return pods.Items, nil
	}

	nsList := new(corev1.NamespaceList)
	nsSelector, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.NamespaceSelector)
	if err != nil {
		return nil, err
	}
	opt := &client.ListOptions{
		LabelSelector: nsSelector,
		Namespace:     policy.Namespace,
	}
	err = cli.List(ctx, nsList, opt)
	if err != nil {
		return nil, err
	}

	res := make([]corev1.Pod, 0)

	for _, ns := range nsList.Items {
		pods := new(corev1.PodList)
		selector, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.PodSelector)
		if err != nil {
			return nil, err
		}
		opt := &client.ListOptions{
			LabelSelector: selector,
			Namespace:     ns.Name,
		}
		err = cli.List(ctx, pods, opt)
		if err != nil {
			return nil, err
		}
		res = append(res, pods.Items...)
	}

	return res, nil
}

func listClusterEndpointSlices(ctx context.Context, cli client.Client, policyName string) (*egressv1.EgressClusterEndpointSliceList, error) {
	slices := new(egressv1.EgressClusterEndpointSliceList)
	labelSelector := &metav1.LabelSelector{MatchLabels: map[string]string{
		egressv1.LabelPolicyName: policyName,
	}}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	opt := &client.ListOptions{
		LabelSelector: selector,
	}
	err = cli.List(ctx, slices, opt)
	return slices, err
}

func newEgressClusterEpSliceController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	r := &endpointClusterReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}

	name := "cluster-endpoint"
	log.Sugar().Infof("new %v controller", name)

	cache, err := coalescing.NewRequestCache(time.Second)
	if err != nil {
		return err
	}
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New(name, mgr, controller.Options{Reconciler: reduce})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(enqueueEGCP(r.client)),
		podPredicate{},
	)
	if err != nil {
		return fmt.Errorf("failed to watch pod: %v", err)
	}

	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}},
		handler.EnqueueRequestsFromMapFunc(enqueueNS(r.client)),
		nsPredicate{},
	)
	if err != nil {
		return fmt.Errorf("failed to watch namespace: %v", err)
	}

	err = c.Watch(&source.Kind{Type: &egressv1.EgressClusterGatewayPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("failed to watch EgressClusterGatewayPolicy: %v", err)
	}

	err = c.Watch(&source.Kind{Type: &egressv1.EgressClusterEndpointSlice{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &egressv1.EgressClusterGatewayPolicy{},
		IsController: true,
	})
	if err != nil {
		return fmt.Errorf("failed to watch EgressClusterEndpointSlice: %v", err)
	}

	return nil
}

type nsPredicate struct {
}

func (p nsPredicate) Create(_ event.CreateEvent) bool {
	return true
}

func (p nsPredicate) Delete(_ event.DeleteEvent) bool {
	return true
}

func (p nsPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldNS, ok := updateEvent.ObjectOld.(*corev1.Namespace)
	if !ok {
		return false
	}
	newNS, ok := updateEvent.ObjectNew.(*corev1.Namespace)
	if !ok {
		return false
	}

	// case by pods labels are changed
	if reflect.DeepEqual(oldNS.Labels, newNS.Labels) {
		return false
	}

	return true
}

func (p nsPredicate) Generic(_ event.GenericEvent) bool {
	return true
}

func enqueueNS(cli client.Client) handler.MapFunc {
	return func(obj client.Object) []reconcile.Request {
		ns, ok := obj.(*corev1.Namespace)
		if !ok {
			return nil
		}

		policyList := new(egressv1.EgressClusterGatewayPolicyList)
		err := cli.List(context.Background(), policyList)
		if err != nil {
			return nil
		}

		res := make([]reconcile.Request, 0)

		for _, policy := range policyList.Items {
			selPods, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.NamespaceSelector)
			if err != nil {
				return nil
			}
			match := selPods.Matches(labels.Set(ns.Labels))
			if match {
				res = append(res, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: policy.Namespace,
						Name:      policy.Name,
					},
				})
			}
		}
		return res
	}
}

func enqueueEGCP(cli client.Client) handler.MapFunc {
	return func(obj client.Object) []reconcile.Request {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}

		policyList := new(egressv1.EgressClusterGatewayPolicyList)
		err := cli.List(context.Background(), policyList)
		if err != nil {
			return nil
		}

		res := make([]reconcile.Request, 0)

		nsList := new(corev1.NamespaceList)
		err = cli.List(context.Background(), nsList)
		if err != nil {
			return nil
		}

		for _, policy := range policyList.Items {
			selPods, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.PodSelector)
			if err != nil {
				return nil
			}
			match := selPods.Matches(labels.Set(pod.Labels))
			if match {
				if policy.Spec.AppliedTo.NamespaceSelector != nil {
					ns := new(corev1.Namespace)
					err := cli.Get(context.Background(), types.NamespacedName{Name: pod.Namespace}, ns)
					if err != nil {
						return nil
					}
					selNS, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.NamespaceSelector)
					if err != nil {
						return nil
					}
					match := selNS.Matches(labels.Set(ns.Labels))
					if !match {
						continue
					}
				}

				res = append(res, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: policy.Namespace,
						Name:      policy.Name,
					},
				})
			}
		}
		return res
	}
}
