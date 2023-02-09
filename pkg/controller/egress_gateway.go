// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

const (
	indexEgressNodeEgressGateway = "egressNodeEgressGatewayIndex"
	indexNodeEgressGateway       = "nodeEgressGatewayIndex"
)

type egnReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r egnReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Infof("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}
	log := r.log.With(
		zap.String("namespacedName", newReq.NamespacedName.String()),
		zap.String("kind", kind),
	)
	log.Info("reconciling")
	switch kind {
	case "EgressGateway":
		return r.reconcileEG(ctx, newReq, log)
	case "EgressNode":
		return r.reconcileEN(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileNode reconcile node
// goal:
// - in used
//   - ready -> not ready
//   - not ready -> ready
//
// not goal:
// - add    node
// - remove node
func (r egnReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	node := new(corev1.Node)
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	affectedEgressGatewayList := &egressv1.EgressGatewayList{}
	if err := r.client.List(context.Background(), affectedEgressGatewayList,
		&client.ListOptions{FieldSelector: fields.OneTermEqualSelector(
			indexEgressNodeEgressGateway,
			req.NamespacedName.String()),
		}); err != nil {
		return reconcile.Result{}, nil
	}

	eg := new(egressv1.EgressNode)
	err = r.client.Get(ctx, req.NamespacedName, eg)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		log.Info("egress node found, skip reconcile node")
		return reconcile.Result{}, err
	}

	ready := false
	if utils.IsNodeReady(node) && utils.IsNodeVxlanReady(eg,
		r.config.FileConfig.EnableIPv4,
		r.config.FileConfig.EnableIPv6,
	) {
		ready = true
	}

	for _, egn := range affectedEgressGatewayList.Items {
		index := -1
		for i, selNode := range egn.Status.NodeList {
			if selNode.Name == node.Name {
				index = i
			}
		}
		if index == -1 {
			return reconcile.Result{}, err
		}
		change := false
		if egn.Status.NodeList[index].Ready != ready {
			change = true
			egn.Status.NodeList[index].Ready = ready
		}
		if r.updateActive(egn.Status.NodeList) {
			log.Sugar().Debugf("egress node %s active changed", egn.Name)
			change = true
		}
		if change {
			log.Sugar().Debugf("update egress gateway node %s", mustMarshalJson(egn.Status))
			err = r.client.Status().Update(ctx, &egn)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// reconcileEG reconcile egress node
// goal:
// - add egress gateway node
// - update egress gateway node
func (r egnReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	egn := &egressv1.EgressGateway{}
	err := r.client.Get(ctx, req.NamespacedName, egn)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !egn.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Info("request item is deleted")
		return reconcile.Result{}, nil
	}

	if egn.Spec.NodeSelector == nil {
		log.Info("nodeSelector is nil, skip reconcile")
		return reconcile.Result{}, nil
	}

	nodeList := &corev1.NodeList{}
	selNodes, err := metav1.LabelSelectorAsSelector(egn.Spec.NodeSelector)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = r.client.List(ctx, nodeList, &client.ListOptions{
		LabelSelector: selNodes,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	log.Sugar().Debugf("number of selected nodes: %d", len(nodeList.Items))

	egressNodeList := make([]egressv1.SelectedEgressNode, 0)
	for _, node := range nodeList.Items {
		log.Sugar().Debugf("check node: %s", node.Name)

		eNode := new(egressv1.EgressNode)
		err = r.client.Get(ctx, types.NamespacedName{Name: node.Name}, eNode)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{}, fmt.Errorf("get egress node with error: %v", err)
			}
			eNode = &egressv1.EgressNode{ObjectMeta: metav1.ObjectMeta{Name: node.Name}}
			err := r.client.Create(ctx, eNode)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		log.Sugar().Debug("get egress node: ", node.Name)

		isReady := false
		if utils.IsNodeVxlanReady(eNode,
			r.config.FileConfig.EnableIPv4,
			r.config.FileConfig.EnableIPv6) {
			isReady = true
		}

		log.Sugar().Debugf("egress node is ready: %v", isReady)

		egressNodeList = append(egressNodeList, egressv1.SelectedEgressNode{
			Name:  eNode.Name,
			Ready: isReady,
		})
	}

	hasReady := false
	hasActive := false
	diff := difference(egn.Status.NodeList, egressNodeList,
		func(t1, t2 egressv1.SelectedEgressNode) bool {
			if t1.Active {
				hasActive = true
			}
			if t1.Ready {
				hasReady = true
			}
			if t1.Name != t2.Name {
				return true
			}
			return false
		})
	if !diff && hasReady && hasActive {
		log.Info("skip update egress node gateway status, it hasn't changed")
		return reconcile.Result{}, nil
	}

	if diff {
		egn.Status.NodeList = mergeEgressNodes(egn.Status.NodeList, egressNodeList)
	}

	// for there, it has been change, so we do not need to double-check
	r.updateActive(egn.Status.NodeList)
	log.Sugar().Infof("update egress gateway node %s", mustMarshalJson(egn.Status))
	err = r.client.Status().Update(ctx, egn)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func mustMarshalJson(obj interface{}) string {
	raw, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(raw)
}

func difference(a, b egressv1.SelectedEgressNodes, f func(t1, t2 egressv1.SelectedEgressNode) bool) bool {
	if len(a) != len(b) {
		return true
	}
	sort.Sort(a)
	sort.Sort(b)
	for i := range a {
		if f(a[i], b[i]) {
			return true
		}
	}
	return false
}

func mergeEgressNodes(oldList, preList []egressv1.SelectedEgressNode) []egressv1.SelectedEgressNode {
	m := make(map[string][]egressv1.InterfaceStatus, 0)
	for _, node := range oldList {
		m[node.Name] = node.InterfaceStatus
	}
	newList := make([]egressv1.SelectedEgressNode, 0)
	for _, item := range preList {
		interfaceStatus, ok := m[item.Name]
		if ok {
			item.InterfaceStatus = interfaceStatus
		}
		newList = append(newList, item)
	}
	return newList
}

// reconcileEN reconcile egress node
// goal:
// - add node
// - update node
// - remove node
func (r egnReconciler) reconcileEN(ctx context.Context,
	req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {

	deleted := false
	eg := &egressv1.EgressNode{}
	err := r.client.Get(ctx, req.NamespacedName, eg)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
		deleted = true
	}
	deleted = deleted || !eg.GetDeletionTimestamp().IsZero()

	affectedEgressGatewayList := &egressv1.EgressGatewayList{}
	if err := r.client.List(context.Background(), affectedEgressGatewayList,
		&client.ListOptions{FieldSelector: fields.OneTermEqualSelector(
			indexEgressNodeEgressGateway,
			req.NamespacedName.String()),
		}); err != nil {
		return reconcile.Result{}, nil
	}

	if deleted {
		log.Info("request item is deleted")
		// if req obj is deleted, we should be deleted it in EgressGateway used.
		for _, egn := range affectedEgressGatewayList.Items {
			changed := false
			filterFunc := func(node egressv1.SelectedEgressNode) bool {
				if node.Name == req.Name {
					changed = true
					return false
				}
				return true
			}
			preList := filter(egn.Status.NodeList, filterFunc)
			egn.Status.NodeList = preList
			if r.updateActive(preList) {
				log.Sugar().Debugf("update egress gateway node\n%s", mustMarshalJson(egn))
				changed = true
			}
			if changed {
				log.Sugar().Debugf("update egress gateway node\n%s", mustMarshalJson(egn))
				err = r.client.Status().Update(ctx, &egn)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
		return reconcile.Result{}, nil
	}

	for _, item := range affectedEgressGatewayList.Items {
		exits := false
		changed := false

		for _, node := range item.Status.NodeList {
			if node.Name == req.Name {
				exits = true
			}
		}
		if !exits {
			// if it not exits, need to add it
			changed = true
			ready := false
			node := new(corev1.Node)
			err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, node)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Info("not found node, skip reconcile")
				}
				return reconcile.Result{}, err
			}
			// double calculate status
			if utils.IsNodeReady(node) && utils.IsNodeVxlanReady(eg,
				r.config.FileConfig.EnableIPv4,
				r.config.FileConfig.EnableIPv6,
			) {
				ready = true
			}
			item.Status.NodeList = append(item.Status.NodeList, egressv1.SelectedEgressNode{
				Name:   req.Name,
				Ready:  ready,
				Active: false,
			})
		}

		// exits, but renew active egress node
		if r.updateActive(item.Status.NodeList) {
			log.Sugar().Debugf("egress node %s active changed", item.Name)
			changed = true
		}

		if changed {
			log.Sugar().Debugf("update egress gateway node\n%s", mustMarshalJson(item))
			err = r.client.Status().Update(ctx, &item)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r egnReconciler) updateActive(list egressv1.SelectedEgressNodes) bool {
	hasChanged := false

	if r.config.FileConfig.ForwardMethod == config.ForwardMethodActiveActive {
		for i, node := range list {
			tmp := node.Ready
			if node.Ready {
				list[i].Active = true
			}
			if tmp != node.Active {
				hasChanged = true
			}
		}
	} else {
		hasReady := false
		hasActive := false
		for i, node := range list {
			if !node.Ready {
				if node.Active {
					hasChanged = true
					list[i].Active = false
				}
				continue
			}
			if node.Active {
				hasActive = true
				break
			}
			hasChanged = true
			hasReady = true
		}
		if !hasActive && hasReady {
			hasChanged = true
			sort.Sort(list)
			firstReady := -1
			for i, node := range list {
				if node.Ready {
					firstReady = i
					break
				}
			}
			if firstReady != -1 {
				list[firstReady].Active = true
			}
		}
	}

	return hasChanged
}

func filter[T any](ss []T, f func(item T) bool) []T {
	res := make([]T, 0)
	for _, s := range ss {
		if f(s) {
			res = append(res, s)
		}
	}
	return res
}

func newEgressGatewayController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}
	r := &egnReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &egressv1.EgressGateway{},
		indexEgressNodeEgressGateway, func(rawObj client.Object) []string {
			egn := rawObj.(*egressv1.EgressGateway)
			var egressNodes []string
			for _, node := range egn.Status.NodeList {
				egressNodes = append(egressNodes,
					types.NamespacedName{
						Name: node.Name,
					}.String(),
				)
			}
			return egressNodes
		}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &egressv1.EgressGateway{},
		indexNodeEgressGateway, func(rawObj client.Object) []string {
			egn := rawObj.(*egressv1.EgressGateway)
			var egressNodes []string
			for _, node := range egn.Status.NodeList {
				egressNodes = append(egressNodes,
					types.NamespacedName{
						Name: node.Name,
					}.String(),
				)
			}
			return egressNodes
		}); err != nil {
		return err
	}

	c, err := controller.New("egressGateway", mgr,
		controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressNode{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressNode"))); err != nil {
		return fmt.Errorf("failed to watch EgressNode: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Node{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	return nil
}
