// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

var defaultName = "default"

func NewController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}
	r := &clusterInfo{
		mgr:    mgr,
		log:    log,
		config: cfg,
		cli:    mgr.GetClient(),
	}
	c, err := controller.New("cluster-info", mgr,
		controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	sourceNode := utils.SourceKind(r.mgr.GetCache(),
		&corev1.Node{},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node")),
		nodePredicate{})
	err = c.Watch(sourceNode)
	if err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	sourceInfo := utils.SourceKind(r.mgr.GetCache(),
		&egressv1.EgressClusterInfo{},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressClusterInfo")),
		infoPredicate{})
	err = c.Watch(sourceInfo)
	if err != nil {
		return fmt.Errorf("failed to watch EgressClusterInfo: %w", err)
	}

	r.watchCalico = func() {
		for {
			err = r.cli.List(context.Background(), &calicov1.IPPoolList{})
			if err != nil {
				if meta.IsNoMatchError(err) {
					log.Info("not found CalicoIPPool CRD in current cluster, skipping watch")
				} else {
					log.Error(err, "failed to list Calico IPPool, skipping watch.", "error", err)
				}
				return
			}
			sourceCalicoIPPool := utils.SourceKind(r.mgr.GetCache(),
				&calicov1.IPPool{},
				handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("CalicoIPPool")))
			err = c.Watch(sourceCalicoIPPool)
			if err != nil {
				log.Error(err, "failed to watch CalicoIPPool", "error", err)
				time.Sleep(time.Second * 3)
				continue
			}
			break
		}
	}

	return nil
}

type clusterInfo struct {
	mgr         manager.Manager
	cli         client.Client
	log         logr.Logger
	config      *config.Config
	doOnce      sync.Once
	watchOnce   sync.Once
	watchCalico func()
}

func (r *clusterInfo) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	r.doOnce.Do(func() {
		for {
			err := r.updateK8sServiceCIDR(ctx)
			if err == nil {
				break
			}
			r.log.Error(err, "failed to update k8s Service CIDR")
			time.Sleep(time.Second * 3)
		}
	})

	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}
	log := r.log.WithValues("kind", kind)
	var res reconcile.Result
	switch kind {
	case "Node":
		err = r.reconcileNode(ctx, newReq, log)
	case "CalicoIPPool":
		err = r.reconcileCalicoIPPool(ctx, newReq, log)
	case "EgressClusterInfo":
		err = r.reconcileInfo(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return res, nil
}

func (r *clusterInfo) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	info := new(egressv1.EgressClusterInfo)
	err := r.cli.Get(ctx, client.ObjectKey{Name: defaultName}, info)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		log.Info("not found default EgressClusterInfo")
		return nil
	}
	detectNodeIP := info.Spec.AutoDetect.NodeIP
	detectK8sPodCIDR := info.Spec.AutoDetect.PodCidrMode == egressv1.CniTypeK8s ||
		info.Spec.AutoDetect.PodCidrMode == egressv1.CniTypeAuto
	if !detectNodeIP && !detectK8sPodCIDR {
		return nil
	}

	deleted := false
	node := new(corev1.Node)
	err = r.cli.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()
	changed := false

	if deleted {
		log.Info("delete event of node", "delete", req.Name)
		if detectNodeIP {
			if _, ok := info.Status.NodeIP[req.Name]; ok {
				delete(info.Status.NodeIP, req.Name)
				changed = true
			}
		}
	} else {
		log.Info("update event of node", "update", req.Name)
		if detectNodeIP {
			ipv4, ipv6 := getNodeIPList(node)
			if info.Status.NodeIP == nil {
				info.Status.NodeIP = make(map[string]egressv1.IPListPair)
			}
			if updateIPListPair(info.Status.NodeIP, req.Name, ipv4, ipv6) {
				changed = true
			}
		}
	}

	if detectK8sPodCIDR {
		podCIDRChanged, err := r.syncPodCIDRs(ctx, info)
		if err != nil {
			return err
		}
		changed = changed || podCIDRChanged
	}

	if changed {
		err := r.cli.Status().Update(ctx, info)
		if err != nil {
			return fmt.Errorf("failed to update EgressClusterInfo when reconciling node(%s): %w", req.Name, err)
		}
	}
	return nil
}

func updateIPListPair(items map[string]egressv1.IPListPair, name string, ipv4, ipv6 []string) bool {
	if val, ok := items[name]; ok {
		if utils.EqualStringSlice(val.IPv4, ipv4) && utils.EqualStringSlice(val.IPv6, ipv6) {
			return false
		}
	}

	items[name] = egressv1.IPListPair{
		IPv4: ipv4,
		IPv6: ipv6,
	}
	return true
}

func (r *clusterInfo) reconcileCalicoIPPool(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	info := new(egressv1.EgressClusterInfo)
	err := r.cli.Get(ctx, client.ObjectKey{Name: defaultName}, info)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		log.Info("not found default EgressClusterInfo")
		return nil
	}
	// skip if not enable detect pod cidr
	if info.Spec.AutoDetect.PodCidrMode != egressv1.CniTypeCalico &&
		info.Spec.AutoDetect.PodCidrMode != egressv1.CniTypeAuto {
		return nil
	}

	changed, err := r.syncPodCIDRs(ctx, info)
	if err != nil {
		return err
	}
	if changed {
		if err := r.cli.Status().Update(ctx, info); err != nil {
			return fmt.Errorf("failed to update EgressClusterInfo when reconciling CalicoIPPool(%s): %w", req.Name, err)
		}
	}

	return nil
}

func (r *clusterInfo) reconcileInfo(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	deleted := false
	info := new(egressv1.EgressClusterInfo)
	err := r.cli.Get(ctx, req.NamespacedName, info)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		deleted = true
	}
	deleted = deleted || !info.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Info("delete event of EgressClusterInfo", "delete", req.Name)
		return nil
	} else {
		if info.Spec.AutoDetect.PodCidrMode == "auto" || info.Spec.AutoDetect.PodCidrMode == "calico" {
			r.watchOnce.Do(r.watchCalico)
		}

		log.Info("update event of EgressClusterInfo", "update", req.Name)
		changed := false
		if !utils.EqualStringSlice(info.Spec.ExtraCidr, info.Status.ExtraCidr) {
			info.Status.ExtraCidr = info.Spec.ExtraCidr
			changed = true
		}

		podCIDRChanged, err := r.syncPodCIDRs(ctx, info)
		if err != nil {
			return err
		}
		changed = changed || podCIDRChanged

		if changed {
			err := r.cli.Status().Update(ctx, info)
			if err != nil {
				return fmt.Errorf("failed to update EgressClusterInfo status: %w", err)
			}
		}
	}
	return nil
}

func (r *clusterInfo) syncPodCIDRs(ctx context.Context, info *egressv1.EgressClusterInfo) (bool, error) {
	mode, podCIDRs, err := r.resolvePodCIDRs(ctx, info.Spec.AutoDetect.PodCidrMode)
	if err != nil {
		return false, err
	}

	changed := false
	if info.Status.PodCidrMode != mode {
		info.Status.PodCidrMode = mode
		changed = true
	}
	if !equalIPListPairMap(info.Status.PodCIDR, podCIDRs) {
		info.Status.PodCIDR = podCIDRs
		changed = true
	}

	return changed, nil
}

func (r *clusterInfo) resolvePodCIDRs(ctx context.Context, configuredMode egressv1.PodCidrMode) (egressv1.PodCidrMode, map[string]egressv1.IPListPair, error) {
	switch configuredMode {
	case egressv1.CniTypeEmpty:
		return egressv1.CniTypeEmpty, nil, nil
	case egressv1.CniTypeK8s:
		podCIDRs, err := r.listK8sPodCIDRs(ctx)
		return egressv1.CniTypeK8s, podCIDRs, err
	case egressv1.CniTypeCalico:
		podCIDRs, _, err := r.listCalicoPodCIDRs(ctx)
		return egressv1.CniTypeCalico, podCIDRs, err
	case egressv1.CniTypeAuto:
		podCIDRs, poolCount, err := r.listCalicoPodCIDRs(ctx)
		if err == nil && poolCount > 0 {
			return egressv1.CniTypeCalico, podCIDRs, nil
		}
		if err != nil && !meta.IsNoMatchError(err) {
			return egressv1.CniTypeEmpty, nil, fmt.Errorf("failed to auto-detect Calico Pod CIDRs: %w", err)
		}

		podCIDRs, err = r.listK8sPodCIDRs(ctx)
		return egressv1.CniTypeK8s, podCIDRs, err
	default:
		return egressv1.CniTypeEmpty, nil, fmt.Errorf("unsupported Pod CIDR mode %q", configuredMode)
	}
}

func (r *clusterInfo) listK8sPodCIDRs(ctx context.Context) (map[string]egressv1.IPListPair, error) {
	nodes := new(corev1.NodeList)
	if err := r.cli.List(ctx, nodes); err != nil {
		return nil, fmt.Errorf("failed to list nodes when updating Kubernetes Pod CIDRs: %w", err)
	}

	podCIDRs := make(map[string]egressv1.IPListPair)
	for i := range nodes.Items {
		node := &nodes.Items[i]
		ipv4, ipv6 := getNodePodCIDRList(node)
		if len(ipv4) == 0 && len(ipv6) == 0 {
			continue
		}
		podCIDRs[node.Name] = egressv1.IPListPair{IPv4: ipv4, IPv6: ipv6}
	}
	return podCIDRs, nil
}

func (r *clusterInfo) listCalicoPodCIDRs(ctx context.Context) (map[string]egressv1.IPListPair, int, error) {
	pools := new(calicov1.IPPoolList)
	if err := r.cli.List(ctx, pools); err != nil {
		return nil, 0, err
	}

	podCIDRs := make(map[string]egressv1.IPListPair)
	for i := range pools.Items {
		pool := &pools.Items[i]
		ipv4, ipv6 := getCalicoIPPoolList(pool)
		if len(ipv4) == 0 && len(ipv6) == 0 {
			continue
		}
		podCIDRs[pool.Name] = egressv1.IPListPair{IPv4: ipv4, IPv6: ipv6}
	}
	return podCIDRs, len(pools.Items), nil
}

func equalIPListPairMap(a, b map[string]egressv1.IPListPair) bool {
	if len(a) != len(b) {
		return false
	}
	for name, aPair := range a {
		bPair, ok := b[name]
		if !ok || !utils.EqualStringSlice(aPair.IPv4, bPair.IPv4) ||
			!utils.EqualStringSlice(aPair.IPv6, bPair.IPv6) {
			return false
		}
	}
	return true
}

func (r *clusterInfo) updateK8sServiceCIDR(ctx context.Context) error {
	info := new(egressv1.EgressClusterInfo)
	err := r.cli.Get(ctx, client.ObjectKey{Name: defaultName}, info)
	if err != nil {
		return err
	}
	// skip if not enable detect service cidr
	if !info.Spec.AutoDetect.ClusterIP {
		return nil
	}
	v4, v6, err := GetClusterCIDR(ctx, r.cli)
	if err != nil {
		r.log.Error(err, "failed to get cluster cidr")
		return err
	}

	if info.Status.ClusterIP != nil {
		if !utils.EqualStringSlice(info.Status.ClusterIP.IPv4, v4) ||
			!utils.EqualStringSlice(info.Status.ClusterIP.IPv6, v6) {

			info.Status.ClusterIP = &egressv1.IPListPair{IPv4: v4, IPv6: v6}
			err := r.cli.Status().Update(ctx, info)
			if err != nil {
				return fmt.Errorf("failed to update EgressClusterInfo when update k8s Service CIDR: %w", err)
			}
		}
	} else {
		info.Status.ClusterIP = &egressv1.IPListPair{IPv4: v4, IPv6: v6}
		err := r.cli.Status().Update(ctx, info)
		if err != nil {
			return fmt.Errorf("failed to update EgressClusterInfo when update k8s Service CIDR: %w", err)
		}
	}

	return nil
}

type nodePredicate struct{}

func (p nodePredicate) Create(_ event.CreateEvent) bool { return true }
func (p nodePredicate) Delete(_ event.DeleteEvent) bool { return true }
func (p nodePredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*corev1.Node)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*corev1.Node)
	if !ok {
		return false
	}

	oldV4, oldV6 := getNodeIPList(oldObj)
	newV4, newV6 := getNodeIPList(newObj)
	oldPodV4, oldPodV6 := getNodePodCIDRList(oldObj)
	newPodV4, newPodV6 := getNodePodCIDRList(newObj)
	if utils.EqualStringSlice(oldV4, newV4) &&
		utils.EqualStringSlice(oldV6, newV6) &&
		utils.EqualStringSlice(oldPodV4, newPodV4) &&
		utils.EqualStringSlice(oldPodV6, newPodV6) {
		return false
	}
	return true

}
func (p nodePredicate) Generic(_ event.GenericEvent) bool { return false }

type infoPredicate struct{}

func (p infoPredicate) Create(_ event.CreateEvent) bool { return true }
func (p infoPredicate) Delete(_ event.DeleteEvent) bool { return false }
func (p infoPredicate) Update(updateEvent event.UpdateEvent) bool {
	oldObj, ok := updateEvent.ObjectOld.(*egressv1.EgressClusterInfo)
	if !ok {
		return false
	}
	newObj, ok := updateEvent.ObjectNew.(*egressv1.EgressClusterInfo)
	if !ok {
		return false
	}
	if oldObj.Spec.AutoDetect != newObj.Spec.AutoDetect {
		return true
	}
	if !utils.EqualStringSlice(oldObj.Spec.ExtraCidr, newObj.Spec.ExtraCidr) {
		return true
	}
	return false
}
func (p infoPredicate) Generic(_ event.GenericEvent) bool { return false }
