// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"

	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"

	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/lock"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

type eciReconciler struct {
	mgr                          manager.Manager
	c                            controller.Controller
	isWatchingCalico             bool // whether controller is watching calico
	isWatchingNode               bool
	k8sPodCidr                   map[string]egressv1beta1.IPListPair
	v4ClusterCidr, v6ClusterCidr []string
	eci                          *egressv1beta1.EgressClusterInfo
	client                       client.Client
	log                          logr.Logger
	eciMutex                     lock.RWMutex
	// stopCheckChan Stop the goroutine that detect the existence of the cni
	stopCheckChan                 chan struct{}
	isCheckCalicoGoroutineRunning atomic.Bool
	// taskToken Avoid multiple goroutines in the program that detect the existence of a cni
	taskToken atomic.Bool
}

const (
	defaultEgressClusterInfoName = "default"
	k8s                          = "k8s"
	serviceClusterIpRange        = "service-cluster-ip-range"
	clusterCidr                  = "cluster-cidr"
)

const (
	kindNode         = "Node"
	kindCalicoIPPool = "CalicoIPPool"
	kindEGCI         = "EGCI"
)

var kubeControllerManagerPodLabel = map[string]string{"component": "kube-controller-manager"}

func NewEgressClusterInfoController(mgr manager.Manager, log logr.Logger) error {
	r := &eciReconciler{
		mgr:           mgr,
		eci:           new(egressv1beta1.EgressClusterInfo),
		client:        mgr.GetClient(),
		log:           log,
		k8sPodCidr:    make(map[string]egressv1beta1.IPListPair),
		v4ClusterCidr: make([]string, 0),
		v6ClusterCidr: make([]string, 0),
	}

	log.Info("new egressClusterInfo controller")
	c, err := controller.New("egressClusterInfo", mgr,
		controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	r.c = c

	log.Info("egressClusterInfo controller watch EgressClusterInfo")
	return watchSource(c, source.Kind(mgr.GetCache(), &egressv1beta1.EgressClusterInfo{}), kindEGCI)
}

// Reconcile support to reconcile of nodes, calicoIPPool and egressClusterInfo
func (r *eciReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}
	log := r.log.WithValues("kind", kind)

	r.eciMutex.Lock()
	defer r.eciMutex.Unlock()

	eciStatusCopy := r.eci.Status.DeepCopy()

	// get egressClusterInfo
	err = r.getEgressClusterInfo(ctx)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	switch kind {
	case kindNode:
		err = r.reconcileNode(ctx, newReq, log)
	case kindCalicoIPPool:
		if r.eci.Spec.AutoDetect.PodCidrMode != egressv1beta1.CniTypeCalico {
			return reconcile.Result{}, nil
		}
		err = r.reconcileCalicoIPPool(ctx, newReq, log)
	case kindEGCI:
		err = r.reconcileEgressClusterInfo(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	if !reflect.DeepEqual(eciStatusCopy, r.eci.Status) {
		err = r.client.Status().Update(ctx, r.eci)
		if err != nil {
			//r.eci = eciCopy
			if errors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

// reconcileEgressClusterInfo reconcile cr egressClusterInfo
func (r *eciReconciler) reconcileEgressClusterInfo(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("reconciling")

	// ignore nodeIP
	if r.eci.Spec.AutoDetect.NodeIP {
		if !r.isWatchingNode {
			// need watch node
			if err := watchSource(r.c, source.Kind(r.mgr.GetCache(), &corev1.Node{}), kindNode); err != nil {
				return err
			}
			r.isWatchingNode = true
			// need to list all node
			nodesIP, err := r.listNodeIPs(ctx)
			if err != nil {
				return err
			}
			r.eci.Status.NodeIP = nodesIP
		}
	} else {
		r.eci.Status.NodeIP = nil
	}

	// ignore podCidr
	switch r.eci.Spec.AutoDetect.PodCidrMode {
	case egressv1beta1.CniTypeAuto:
		err := r.checkSomeCniExists()
		if err != nil {
			return err
		}
	case egressv1beta1.CniTypeCalico:
		if !r.isWatchingCalico && !r.isCheckCalicoGoroutineRunning.Load() {
			if r.stopCheckChan == nil {
				r.stopCheckChan = make(chan struct{})
			}
			// set empty before check calico
			r.eci.Status.PodCIDR = nil
			r.eci.Status.PodCidrMode = egressv1beta1.CniTypeEmpty

			// check if calico is exists
			r.startCheckCalico(r.stopCheckChan)
		} else {
			pools, err := r.listCalicoIPPools(ctx)
			if err != nil {
				return err
			}
			r.eci.Status.PodCIDR = pools
			r.eci.Status.PodCidrMode = egressv1beta1.CniTypeCalico
		}
	case egressv1beta1.CniTypeK8s:
		// close all check goroutine
		r.stopAllCheckGoroutine()

		if _, ok := r.k8sPodCidr[k8s]; !ok {
			cidr, err := r.getK8sPodCidr()
			if err != nil {
				return err
			}
			r.k8sPodCidr = cidr
		}
		r.eci.Status.PodCIDR = r.k8sPodCidr
		r.eci.Status.PodCidrMode = egressv1beta1.CniTypeK8s
	case egressv1beta1.CniTypeEmpty:
		// close all check goroutine
		r.stopAllCheckGoroutine()

		r.eci.Status.PodCIDR = nil
		r.eci.Status.PodCidrMode = ""
	default:
		r.log.Error(fmt.Errorf("invalid podCidrMode"), "invalid podCidrMode", "Spec.AutoDetect.PodCidrMode", r.eci.Spec.AutoDetect.PodCidrMode)
	}

	// ignore clusterIP
	if r.eci.Spec.AutoDetect.ClusterIP {
		if len(r.v4ClusterCidr) == 0 {
			v4Cidr, v6Cidr, err := r.getServiceClusterIPRange()
			if err != nil {
				return err
			}
			r.v4ClusterCidr = v4Cidr
			r.v6ClusterCidr = v6Cidr
		}
		if r.eci.Status.ClusterIP == nil {
			r.eci.Status.ClusterIP = new(egressv1beta1.IPListPair)
		}
		r.eci.Status.ClusterIP.IPv4 = r.v4ClusterCidr
		r.eci.Status.ClusterIP.IPv6 = r.v6ClusterCidr
	} else {
		r.eci.Status.ClusterIP = nil
	}

	// extraCidr
	if r.eci.Spec.ExtraCidr != nil {
		r.eci.Status.ExtraCidr = r.eci.Spec.ExtraCidr
	} else {
		r.eci.Status.ExtraCidr = nil
	}
	return nil
}

// reconcileCalicoIPPool reconcile calico IPPool
func (r *eciReconciler) reconcileCalicoIPPool(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("reconciling")

	deleted := false
	ippool := new(calicov1.IPPool)
	err := r.client.Get(ctx, req.NamespacedName, ippool)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		deleted = true
	}
	deleted = deleted || !ippool.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		log.Info("delete event of calico ippool", "delete", req.Name)
		delete(r.eci.Status.PodCIDR, req.Name)
	} else {
		// not delete event
		log.Info("update event of calico ippool", "update", req.Name)
		poolsMap, err := r.getCalicoIPPools(ctx, req.Name)
		if err != nil {
			return err
		}
		if r.eci.Status.PodCIDR == nil {
			r.eci.Status.PodCIDR = make(map[string]egressv1beta1.IPListPair)
		}
		r.eci.Status.PodCIDR[req.Name] = poolsMap[req.Name]
	}

	return nil
}

// reconcileNode reconcile node
func (r *eciReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) error {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	deleted := false
	node := new(corev1.Node)
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		log.Info("delete event of node", "delete", req.Name)
		delete(r.eci.Status.NodeIP, req.Name)
	} else {
		// not delete event
		log.Info("update event of node", "update", req.Name)
		nodeIPMap, err := r.getNodeIPs(ctx, req.Name)
		if err != nil {
			return err
		}
		if r.eci.Status.NodeIP == nil {
			r.eci.Status.NodeIP = make(map[string]egressv1beta1.IPListPair)
		}
		r.eci.Status.NodeIP[req.Name] = nodeIPMap[req.Name]
	}

	return nil
}

// listCalicoIPPools list all calico ippools
func (r *eciReconciler) listCalicoIPPools(ctx context.Context) (map[string]egressv1beta1.IPListPair, error) {
	ippoolList := new(calicov1.IPPoolList)
	calicoIPPoolMap := make(map[string]egressv1beta1.IPListPair, 0)

	err := r.client.List(ctx, ippoolList)
	if err != nil {
		return nil, err
	}
	for _, item := range ippoolList.Items {
		cidr := item.Spec.CIDR
		ipListPair := egressv1beta1.IPListPair{}

		isV4Cidr, err := ip.IsIPv4Cidr(cidr)
		if err != nil {
			return nil, err
		}
		if isV4Cidr {
			ipListPair.IPv4 = append(ipListPair.IPv4, cidr)
			calicoIPPoolMap[item.Name] = ipListPair
		}
		isV6Cidr, err := ip.IsIPv6Cidr(cidr)
		if err != nil {
			return nil, err
		}
		if isV6Cidr {
			ipListPair.IPv6 = append(ipListPair.IPv6, cidr)
			calicoIPPoolMap[item.Name] = ipListPair
		}
	}
	return calicoIPPoolMap, nil
}

// getCalicoIPPools get calico ippool by name
func (r *eciReconciler) getCalicoIPPools(ctx context.Context, poolName string) (map[string]egressv1beta1.IPListPair, error) {
	ippool := new(calicov1.IPPool)
	calicoIPPoolMap := make(map[string]egressv1beta1.IPListPair, 0)

	err := r.client.Get(ctx, types.NamespacedName{Name: poolName}, ippool)
	if err != nil {
		return nil, err
	}
	cidr := ippool.Spec.CIDR
	ipListPair := egressv1beta1.IPListPair{}

	isV4Cidr, err := ip.IsIPv4Cidr(cidr)
	if err != nil {
		return nil, err
	}
	if isV4Cidr {
		ipListPair.IPv4 = append(ipListPair.IPv4, cidr)
		calicoIPPoolMap[ippool.Name] = ipListPair
	}
	isV6Cidr, err := ip.IsIPv6Cidr(cidr)
	if err != nil {
		return nil, err
	}
	if isV6Cidr {
		ipListPair.IPv6 = append(ipListPair.IPv6, cidr)
		calicoIPPoolMap[ippool.Name] = ipListPair
	}
	return calicoIPPoolMap, nil
}

// listNodeIPs list all node ips
func (r *eciReconciler) listNodeIPs(ctx context.Context) (map[string]egressv1beta1.IPListPair, error) {
	nodeList := new(corev1.NodeList)
	nodesIPMap := make(map[string]egressv1beta1.IPListPair, 0)

	err := r.client.List(ctx, nodeList)
	if err != nil {
		return nil, err
	}

	for _, item := range nodeList.Items {
		var ipv4s, ipv6s []string
		nodeIPv4, nodeIPv6 := utils.GetNodeIP(&item)
		if len(nodeIPv4) != 0 {
			ipv4s = []string{nodeIPv4}
		}
		if len(nodeIPv6) != 0 {
			ipv6s = []string{nodeIPv6}
		}
		nodesIPMap[item.Name] = egressv1beta1.IPListPair{IPv4: ipv4s, IPv6: ipv6s}
	}
	return nodesIPMap, nil
}

// getNodeIPs get node ip by name
func (r *eciReconciler) getNodeIPs(ctx context.Context, nodeName string) (map[string]egressv1beta1.IPListPair, error) {
	node := new(corev1.Node)
	nodesIPMap := make(map[string]egressv1beta1.IPListPair, 0)

	err := r.client.Get(ctx, types.NamespacedName{Name: nodeName}, node)
	if err != nil {
		return nil, err
	}

	nodeIPv4, nodeIPv6 := utils.GetNodeIP(node)
	var ipv4s, ipv6s []string
	if len(nodeIPv4) != 0 {
		ipv4s = []string{nodeIPv4}
	}
	if len(nodeIPv6) != 0 {
		ipv6s = []string{nodeIPv6}
	}
	nodesIPMap[nodeName] = egressv1beta1.IPListPair{IPv4: ipv4s, IPv6: ipv6s}
	return nodesIPMap, nil
}

// getEgressClusterInfo get EgressClusterInfo cr
func (r *eciReconciler) getEgressClusterInfo(ctx context.Context) error {
	return r.client.Get(ctx, types.NamespacedName{Name: defaultEgressClusterInfoName}, r.eci)
}

// getServiceClusterIPRange get service-cluster-ip-range from kube controller manager
func (r *eciReconciler) getServiceClusterIPRange() (ipv4Range, ipv6Range []string, err error) {
	pod, err := GetPodByLabel(r.client, kubeControllerManagerPodLabel)
	if err != nil {
		return nil, nil, err
	}
	return ParseCidrFromControllerManager(pod, serviceClusterIpRange)
}

// getK8sPodCidr get k8s default podCidr
func (r *eciReconciler) getK8sPodCidr() (map[string]egressv1beta1.IPListPair, error) {
	v4Cidr, v6Cidr, err := GetClusterCidr(r.client)
	if err != nil {
		return nil, err
	}
	k8sPodCIDR := make(map[string]egressv1beta1.IPListPair)
	k8sPodCIDR[k8s] = egressv1beta1.IPListPair{IPv4: v4Cidr, IPv6: v6Cidr}
	return k8sPodCIDR, nil
}

// checkCalicoExists once calico is detected, start watch
func (r *eciReconciler) checkCalicoExists(stopChan <-chan struct{}) {
	r.log.V(1).Info("checkCalicoExists...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer r.taskToken.Store(false)
	defer r.isCheckCalicoGoroutineRunning.Store(false)

	for {
		select {
		case <-stopChan:
			r.log.V(1).Info("succeeded to stop check calico exists", "goroutine", "checkCalicoExists")
			return
		default:
			// check if other check-goroutine taken the token
			r.log.V(1).Info("Check if other check-goroutine taken the token")
			if r.taskToken.Load() {
				r.log.V(1).Info("Find other cni, so stop this check goroutine")
				return
			}

		TASK:
			pools, err := r.listCalicoIPPools(ctx)
			if err != nil {
				r.log.V(1).Info("failed listCalicoIPPools when checkCalicoExists, try again", "details", err)
				time.Sleep(time.Second * 3)
				continue
			}
			// find calico ippool, take the token, to do task
			if !r.taskToken.Load() {
				r.taskToken.Store(true)
			}

			if !r.isWatchingCalico {
			RETRY:
				r.log.V(1).Info("find calico ippool, egressClusterInfo controller begin to watch calico ippool")
				if err = watchSource(r.c, source.Kind(r.mgr.GetCache(), &calicov1.IPPool{}), kindCalicoIPPool); err != nil {
					r.log.V(1).Info("failed watch calico ippool, try again", "details", err)
					time.Sleep(time.Second)
					goto RETRY
				}
				r.log.V(1).Info("egressClusterInfo controller succeeded to watch calico ippool")
				r.isWatchingCalico = true
			}
			// find calico update the egci
			r.eciMutex.Lock()
			r.eci.Status.PodCIDR = pools
			r.eci.Status.PodCidrMode = egressv1beta1.CniTypeCalico
			err = r.client.Status().Update(ctx, r.eci)
			r.eciMutex.Unlock()
			if err != nil {
				r.log.V(1).Info("failed update egressClusterInfo, try again", "details", err)
				time.Sleep(time.Second)
				goto TASK
			}
			// finish task
			//  if here to set false, when the goroutine stopped by other process, the value is not set successfully, so it should set defer
			//r.isCheckCalicoGoroutineRunning.Store(false)
			//r.taskToken.Store(false)
			return
		}
	}
}

// startCheckCalico start a goroutine to check calico exists
func (r *eciReconciler) startCheckCalico(stopChan <-chan struct{}) {
	r.log.V(1).Info("startCheckCalico...")
	r.isCheckCalicoGoroutineRunning.Store(true)
	go r.checkCalicoExists(stopChan)
}

// stopCheckCalico close the goroutine that check calico exists
func (r *eciReconciler) stopCheckCalico() {
	if r.isCheckCalicoGoroutineRunning.Load() {
		r.log.V(1).Info("stopCheckCalico...")
		close(r.stopCheckChan)
		r.isCheckCalicoGoroutineRunning.Store(false)
	}
}

// stopAllCheckGoroutine close all check-goroutine
func (r *eciReconciler) stopAllCheckGoroutine() {
	if r.taskToken.Load() {
		r.log.V(1).Info("stopAllCheckGoroutine...")
		close(r.stopCheckChan)
		r.taskToken.Store(false)
	}
}

// checkSomeCniExists
func (r *eciReconciler) checkSomeCniExists() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// check calico exists
	pools, err := r.listCalicoIPPools(ctx)
	if err != nil {
		r.log.V(1).Info("Not find calico ippool")
	} else {
		// controller to watch calico ippool
		r.log.V(1).Info("begin to watch calico ippool when checkSomeCniExists")
		if err = watchSource(r.c, source.Kind(r.mgr.GetCache(), &calicov1.IPPool{}), kindCalicoIPPool); err != nil {
			r.log.V(1).Info("failed watch calico ippool when checkSomeCniExists", "details", err)
			return err
		}

		r.eci.Status.PodCIDR = pools
		r.eci.Status.PodCidrMode = egressv1beta1.CniTypeCalico
		return nil
	}

	// if all cni not found, default is k8s podCidr
	k8sCidr, err := r.getK8sPodCidr()
	if err != nil {
		return err
	}
	r.k8sPodCidr = k8sCidr
	r.eci.Status.PodCIDR = k8sCidr
	r.eci.Status.PodCidrMode = egressv1beta1.CniTypeK8s
	return nil
}

// watchSource controller watch given resource
func watchSource(c controller.Controller, source source.Source, kind string) error {
	if err := c.Watch(source, handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat(kind))); err != nil {
		return fmt.Errorf("failed to watch %s: %w", kind, err)
	}
	return nil
}
