// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"
	"net"
	"reflect"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
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

	"github.com/spidernet-io/egressgateway/pkg/coalescing"
	"github.com/spidernet-io/egressgateway/pkg/config"
)

type endpointReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r *endpointReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(
		zap.String("namespace", req.NamespacedName.Namespace),
		zap.String("name", req.NamespacedName.Name),
		zap.String("kind", "EgressPolicy"),
	)

	log.Info("reconcile")
	deleted := false
	policy := new(v1.EgressPolicy)
	err := r.client.Get(ctx, req.NamespacedName, policy)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !policy.GetDeletionTimestamp().IsZero()

	if deleted {
		// Don't need to do anything.
		return reconcile.Result{}, nil
	}

	pods, err := listPodsByPolicy(ctx, r.client, policy)
	if err != nil {
		return reconcile.Result{}, err
	}

	podMap := make(map[types.NamespacedName]*corev1.Pod)
	for _, pod := range pods.Items {
		podMap[types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}] = &pod
	}

	endpointSlices, err := listEndpointSlices(ctx, r.client, policy.Namespace, policy.Name)
	if err != nil {
		return reconcile.Result{}, err
	}

	existingKeyMap := make(map[types.NamespacedName]bool)
	slicesToUpdate := make([]v1.EgressEndpointSlice, 0)
	slicesToCreate := make([]v1.EgressEndpointSlice, 0)
	slicesToDelete := make([]v1.EgressEndpointSlice, 0)
	slicesNotChange := make([]v1.EgressEndpointSlice, 0)

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

	needToCreateEp := make([]v1.EgressEndpoint, 0)

	for _, pod := range pods.Items {
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
					needToCreateEp = make([]v1.EgressEndpoint, 0)
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
					needToCreateEp = make([]v1.EgressEndpoint, 0)
					break
				}
				slicesToUpdate = append(slicesToUpdate, slice)
			}
		}
	}

	for len(needToCreateEp) > 0 {
		epSlice := newEndpointSlice(policy)
		if len(needToCreateEp) > r.config.FileConfig.MaxNumberEndpointPerSlice {
			tmp := needToCreateEp[:r.config.FileConfig.MaxNumberEndpointPerSlice]
			needToCreateEp = needToCreateEp[r.config.FileConfig.MaxNumberEndpointPerSlice:]
			epSlice.Endpoints = append(epSlice.Endpoints, tmp...)
			slicesToCreate = append(slicesToCreate, *epSlice)
		} else {
			epSlice.Endpoints = append(epSlice.Endpoints, needToCreateEp...)
			needToCreateEp = make([]v1.EgressEndpoint, 0)
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

func newEndpointSlice(policy *v1.EgressPolicy) *v1.EgressEndpointSlice {
	// TODO: change it on release v1
	gvk := schema.GroupVersionKind{
		Group:   "egressgateway.spidernet.io",
		Version: "v1beta1",
		Kind:    "EgressPolicy",
	}
	ownerRef := metav1.NewControllerRef(policy, gvk)
	return &v1.EgressEndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    getEndpointSlicePrefix(policy.Name),
			Namespace:       policy.Namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
			Labels: map[string]string{
				v1.LabelPolicyName: policy.Name,
			},
		},
	}
}

func getEndpointSlicePrefix(name string) string {
	prefix := fmt.Sprintf("%s-", name)
	if len(validation.NameIsDNSSubdomain(prefix, true)) != 0 {
		prefix = name
	}
	return prefix
}

func newEndpoint(pod corev1.Pod) *v1.EgressEndpoint {
	ipv4List := make([]string, 0)
	ipv6List := make([]string, 0)

	for _, podIP := range pod.Status.PodIPs {
		ip := net.ParseIP(podIP.IP)
		if ip.To4() != nil {
			ipv4List = append(ipv4List, podIP.IP)
		} else if ip.To16() != nil {
			ipv6List = append(ipv6List, podIP.IP)
		}
	}

	if len(ipv4List) == 0 && len(ipv6List) == 0 {
		return nil
	}

	return &v1.EgressEndpoint{
		Namespace: pod.Namespace,
		Pod:       pod.Name,
		IPv4:      ipv4List,
		IPv6:      ipv6List,
		Node:      pod.Spec.NodeName,
	}
}

func (r *endpointReconciler) initEndpoint() error {
	return nil
}

func listPodsByPolicy(ctx context.Context, cli client.Client, policy *v1.EgressPolicy) (*corev1.PodList, error) {
	pods := new(corev1.PodList)
	selector, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.PodSelector)
	if err != nil {
		return pods, err
	}
	opt := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     policy.Namespace,
	}
	err = cli.List(ctx, pods, opt)
	return pods, err
}

func listEndpointSlices(ctx context.Context, cli client.Client, namespace, policyName string) (*v1.EgressEndpointSliceList, error) {
	slices := new(v1.EgressEndpointSliceList)
	labelSelector := &metav1.LabelSelector{MatchLabels: map[string]string{
		v1.LabelPolicyName: policyName,
	}}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	opt := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
	}
	err = cli.List(ctx, slices, opt)
	return slices, err
}

func newEgressEndpointSliceController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	r := &endpointReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}
	log.Sugar().Infof("new endpoint controller")

	cache, err := coalescing.NewRequestCache(time.Second)
	if err != nil {
		return err
	}
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New("endpoint", mgr, controller.Options{Reconciler: reduce})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(enqueuePod(r.client)),
		podPredicate{},
	)
	if err != nil {
		return fmt.Errorf("failed to watch pod: %v", err)
	}

	err = c.Watch(&source.Kind{Type: &v1.EgressPolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("failed to watch EgressPolicy: %v", err)
	}

	err = c.Watch(&source.Kind{Type: &v1.EgressEndpointSlice{}}, &handler.EnqueueRequestForOwner{
		OwnerType:    &v1.EgressPolicy{},
		IsController: true,
	})

	if err != nil {
		return fmt.Errorf("failed to watch EgressEndpointSlice: %v", err)
	}

	return nil
}

type podPredicate struct {
}

func (p podPredicate) Create(createEvent event.CreateEvent) bool {
	pod, ok := createEvent.Object.(*corev1.Pod)
	if !ok {
		return false
	}
	if len(pod.Status.PodIPs) == 0 {
		return false
	}
	return true
}

func (p podPredicate) Delete(_ event.DeleteEvent) bool {
	return true
}

func (p podPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldPod, ok := updateEvent.ObjectOld.(*corev1.Pod)
	if !ok {
		return false
	}
	newPod, ok := updateEvent.ObjectNew.(*corev1.Pod)
	if !ok {
		return false
	}

	// case by pods labels are changed
	if reflect.DeepEqual(oldPod.Labels, newPod.Labels) &&
		reflect.DeepEqual(oldPod.Status.PodIPs, newPod.Status.PodIPs) &&
		oldPod.Spec.NodeName != newPod.Spec.NodeName {
		return false
	}

	return true
}

func (p podPredicate) Generic(_ event.GenericEvent) bool {
	return true
}

func enqueuePod(cli client.Client) handler.MapFunc {
	return func(obj client.Object) []reconcile.Request {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}

		policyList := new(v1.EgressPolicyList)
		err := cli.List(context.Background(), policyList)
		if err != nil {
			return nil
		}

		res := make([]reconcile.Request, 0)

		for _, policy := range policyList.Items {
			selPods, err := metav1.LabelSelectorAsSelector(policy.Spec.AppliedTo.PodSelector)
			if err != nil {
				return nil
			}
			match := selPods.Matches(labels.Set(pod.Labels))
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
