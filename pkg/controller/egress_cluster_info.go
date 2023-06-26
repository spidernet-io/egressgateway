// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	exv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type eciReconciler struct {
	eci               *egressv1beta1.EgressClusterInfo
	client            client.Client
	log               logr.Logger
	config            *config.Config
	doOnce            sync.Once
	nodeIPv4Map       map[string]string
	nodeIPv6Map       map[string]string
	calicoV4IPPoolMap map[string]string
	calicoV6IPPoolMap map[string]string
}

const (
	defaultEgressClusterInfoName = "default"
	calico                       = "calico"
	k8s                          = "k8s"
	serviceClusterIpRange        = "service-cluster-ip-range"
	clusterCidr                  = "cluster-cidr"
	ippoolCrdName                = "ippools.crd.projectcalico.org"
)

const (
	kindNode   = "Node"
	KindIPPool = "IPPool"
)

var kubeControllerManagerPodLabel = map[string]string{"component": "kube-controller-manager"}

func (r *eciReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}
	log := r.log.WithValues("kind", kind)

	r.doOnce.Do(func() {
		r.log.Info("first reconcile of egressClusterInfo controller, init egressClusterInfo")
	redo:
		err := r.initEgressClusterInfo(ctx)
		if err != nil {
			r.log.Error(err, "first reconcile of egressClusterInfo controller, init egressClusterInfo")
			goto redo
		}
	})

	switch kind {
	case kindNode:
		return r.reconcileNode(ctx, newReq, log)
	case KindIPPool:
		return r.reconcileCalicoIPPool(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileCalicoIPPool reconcile calico IPPool
func (r *eciReconciler) reconcileCalicoIPPool(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("reconciling")

	// eci
	err := r.getEgressClusterInfo(ctx)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	deleted := false
	ippool := new(calicov1.IPPool)
	err = r.client.Get(ctx, req.NamespacedName, ippool)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.V(2).Error(err, "failed to get calico ippool")
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !ippool.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		log.Info("delete event",
			"calicoV4IPPoolMap", r.calicoV4IPPoolMap,
			"calicoV6IPPoolMap", r.calicoV6IPPoolMap)
		// check calicoV4IPPoolMap and calicoV6IPPoolMap
		cidr, v4ok := r.calicoV4IPPoolMap[req.Name]
		if v4ok {
			// need to delete cidr from calicoV4IPPoolMap
			log.V(1).Info("remove IPPool from calicoV4IPPoolMap")
			delete(r.calicoV4IPPoolMap, req.Name)
			// update eci status
			cidrs := r.getCalicoV4IPPoolsCidrs()
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4 = cidrs
			log.V(1).Info("update EgressClusterInfo", "context", cidrs)
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				log.Error(err, "failed to update EgressClusterInfo")
				r.calicoV4IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		}

		cidr, v6ok := r.calicoV6IPPoolMap[req.Name]
		if v6ok {
			// need to delete cidr from calicoV6IPPoolMap
			log.V(1).Info("remove IPPool from calicoV6IPPoolMap")
			delete(r.calicoV6IPPoolMap, req.Name)
			// update eci status
			cidrs := r.getCalicoV6IPPoolsCidrs()
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6 = cidrs
			log.V(1).Info("update egress cluster info", "context", r.eci)
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.calicoV6IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		}
		// need not update calicoIPPoolMap
		return reconcile.Result{}, nil
	}

	// not delete event
	log.Info("update event")

	// check if cidr about ippools changed
	isv4Cidr, err := utils.IsIPv4Cidr(ippool.Spec.CIDR)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	isv6Cidr, err := utils.IsIPv6Cidr(ippool.Spec.CIDR)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	cidr, ok := r.calicoV4IPPoolMap[req.Name]
	if ok {
		// v4PoolName but v6Cidr, delete it from calicoV4IPPoolMap
		if isv6Cidr {
			// update calicoV4IPPoolMap
			delete(r.calicoV4IPPoolMap, req.Name)
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4 = r.getCalicoV4IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.calicoV4IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		} else if ippool.Spec.CIDR != cidr {
			// need to update calicoV4IPPoolMap
			r.calicoV4IPPoolMap[req.Name] = ippool.Spec.CIDR
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4 = r.getCalicoV4IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.calicoV4IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		}
	} else {
		if isv4Cidr {
			// need to update calicoV4IPPoolMap
			r.calicoV4IPPoolMap[req.Name] = ippool.Spec.CIDR
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4 = r.getCalicoV4IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				delete(r.calicoV4IPPoolMap, req.Name)
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	cidr, ok = r.calicoV6IPPoolMap[req.Name]
	if ok {
		// v6PoolName but v4Cidr, delete it from calicoV6IPPoolMap
		if isv4Cidr {
			// update calicoV6IPPoolMap
			delete(r.calicoV6IPPoolMap, req.Name)
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6 = r.getCalicoV6IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.calicoV6IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		} else if ippool.Spec.CIDR != cidr {
			// need to update calicoV6IPPoolMap
			r.calicoV6IPPoolMap[req.Name] = ippool.Spec.CIDR
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6 = r.getCalicoV6IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.calicoV6IPPoolMap[req.Name] = cidr
				return reconcile.Result{Requeue: true}, err
			}
		}
	} else {
		if isv6Cidr {
			// need to update calicoV6IPPoolMap
			r.calicoV6IPPoolMap[req.Name] = ippool.Spec.CIDR
			r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6 = r.getCalicoV6IPPoolsCidrs()
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				delete(r.calicoV6IPPoolMap, req.Name)
				return reconcile.Result{Requeue: true}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

// reconcileNode reconcile node
func (r *eciReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	log = log.WithValues("name", req.Name)
	log.Info("reconciling")

	// eci
	err := r.getEgressClusterInfo(ctx)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	deleted := false
	node := new(corev1.Node)
	err = r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	// delete event
	if deleted {
		log.Info("delete event")
		// check map
		nodeIPv4, v4Ok := r.nodeIPv4Map[req.Name]
		nodeIPv6, v6Ok := r.nodeIPv6Map[req.Name]
		if v4Ok {
			// update map
			delete(r.nodeIPv4Map, req.Name)
			// update eci
			r.eci.Status.EgressIgnoreCIDR.NodeIP.IPv4 = r.getNodesIPs(r.nodeIPv4Map)
		}
		if v6Ok {
			// update map
			delete(r.nodeIPv6Map, req.Name)
			// update eci
			r.eci.Status.EgressIgnoreCIDR.NodeIP.IPv6 = r.getNodesIPs(r.nodeIPv6Map)
		}

		// eci need to update
		if v4Ok && v6Ok {
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.nodeIPv4Map[req.Name] = nodeIPv4
				r.nodeIPv6Map[req.Name] = nodeIPv6
				return reconcile.Result{Requeue: true}, err
			}
		}
		if v4Ok && !v6Ok {
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.nodeIPv4Map[req.Name] = nodeIPv4
				return reconcile.Result{Requeue: true}, err
			}
		}
		if !v4Ok && v6Ok {
			err := r.updateEgressClusterInfo(ctx)
			if err != nil {
				r.nodeIPv6Map[req.Name] = nodeIPv6
				return reconcile.Result{Requeue: true}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// not delete event
	log.Info("update event")

	// get nodeIP, check if its changed
	nodeIPv4, nodeIPv6 := utils.GetNodeIP(node)

	_, v4Ok := r.nodeIPv4Map[req.Name]
	_, v6Ok := r.nodeIPv6Map[req.Name]

	needUpdateECI := false
	if (!v4Ok || r.nodeIPv4Map[req.Name] != nodeIPv4) && len(nodeIPv4) != 0 {
		needUpdateECI = true
		// update map
		r.nodeIPv4Map[req.Name] = nodeIPv4

		// need to update node ip from eci status
		r.eci.Status.EgressIgnoreCIDR.NodeIP.IPv4 = r.getNodesIPs(r.nodeIPv4Map)

	}

	if (!v6Ok || r.nodeIPv6Map[req.Name] != nodeIPv6) && len(nodeIPv6) != 0 {
		needUpdateECI = true
		// update map
		r.nodeIPv6Map[req.Name] = nodeIPv6

		// need to update node ip from eci status
		r.eci.Status.EgressIgnoreCIDR.NodeIP.IPv6 = r.getNodesIPs(r.nodeIPv6Map)
	}

	if needUpdateECI {
		err = r.updateEgressClusterInfo(ctx)
		if err != nil {
			delete(r.nodeIPv4Map, req.Name)
			delete(r.nodeIPv6Map, req.Name)

			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func newEgressClusterInfoController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	r := &eciReconciler{
		eci:               new(egressv1beta1.EgressClusterInfo),
		client:            mgr.GetClient(),
		log:               log,
		config:            cfg,
		doOnce:            sync.Once{},
		nodeIPv4Map:       make(map[string]string),
		nodeIPv6Map:       make(map[string]string),
		calicoV4IPPoolMap: make(map[string]string),
		calicoV6IPPoolMap: make(map[string]string),
	}

	log.Info("new egressClusterInfo controller")
	c, err := controller.New("egressClusterInfo", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	podCidr, _, ignoreNodeIP := r.getEgressIgnoreCIDRConfig()

	if ignoreNodeIP {
		log.Info("egressClusterInfo controller watch Node")
		if err := watchSource(c, source.Kind(mgr.GetCache(), &corev1.Node{}), kindNode); err != nil {
			return err
		}
	}

	switch podCidr {
	case calico:
		log.V(1).Info("find calico in egressgateway configmap, check if calico exists in cluster")
		go checkCalicoExists(mgr, log, c)
	default:
		log.V(1).Info("invalid filed in egressgateway configmap", "PodCIDR", podCidr)
	}

	return nil
}

// initEgressClusterInfo create EgressClusterInfo cr when first reconcile
func (r *eciReconciler) initEgressClusterInfo(ctx context.Context) error {
	r.log.Info("start init EgressClusterInfo", "name", defaultEgressClusterInfoName)
	err := r.getOrCreateEgressClusterInfo(ctx)
	if err != nil {
		return err
	}

	ignorePod, ignoreClusterCidr, _ := r.getEgressIgnoreCIDRConfig()
	if !ignoreClusterCidr && (ignorePod == k8s || ignorePod == "") {
		return nil
	}

	// get service-cluster-ip-range from kube-controller-manager pod
	pod, err := getPod(r.client, kubeControllerManagerPodLabel)
	if err != nil {
		return err
	}

	if ignoreClusterCidr {
		// get service-cluster-ip-range
		ipv4Range, ipv6Range, err := r.getServiceClusterIPRange(pod)
		if err != nil {
			return err
		}
		r.eci.Status.EgressIgnoreCIDR.ClusterIP.IPv4 = ipv4Range
		r.eci.Status.EgressIgnoreCIDR.ClusterIP.IPv6 = ipv6Range
	}

	if ignorePod == k8s || ignorePod == "" {
		// get cluster-cidr
		ipv4Range, ipv6Range, err := r.getClusterCidr(pod)
		if err != nil {
			return err
		}
		r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4 = ipv4Range
		r.eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6 = ipv6Range
	}

	r.log.V(1).Info("update EgressClusterInfo", "context", r.eci)
	err = r.updateEgressClusterInfo(ctx)
	if err != nil {
		return err
	}
	return nil
}

func watchSource(c controller.Controller, source source.Source, kind string) error {
	if err := c.Watch(source, handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat(kind))); err != nil {
		return fmt.Errorf("failed to watch %s: %w", kind, err)
	}
	return nil
}

// getOrCreateEgressClusterInfo get EgressClusterInfo, if not found create
func (r *eciReconciler) getOrCreateEgressClusterInfo(ctx context.Context) error {
	err := r.getEgressClusterInfo(ctx)
	if err != nil {
		err = r.createEgressClusterInfo(ctx)
		if err != nil {
			return err
		}
	}
	r.eci.Status = egressv1beta1.EgressClusterStatus{}
	return nil
}

// createEgressClusterInfo create EgressClusterInfo
func (r *eciReconciler) createEgressClusterInfo(ctx context.Context) error {
	r.eci.Name = defaultEgressClusterInfoName
	r.log.Info("create EgressClusterInfo")
	err := r.client.Create(ctx, r.eci)
	if err != nil {
		return err
	}
	return nil
}

// getCalicoV4IPPoolsCidrs get calico all ipv4 ippools cidrs from calicoV4IPPoolMap
func (r *eciReconciler) getCalicoV4IPPoolsCidrs() []string {
	cidrs := make([]string, 0)
	for _, cidr := range r.calicoV4IPPoolMap {
		cidrs = append(cidrs, cidr)
	}
	return cidrs
}

// getCalicoV6IPPoolsCidrs get calico all ipv6 ippools cidrs from calicoV6IPPoolMap
func (r *eciReconciler) getCalicoV6IPPoolsCidrs() []string {
	cidrs := make([]string, 0)
	for _, cidr := range r.calicoV6IPPoolMap {
		cidrs = append(cidrs, cidr)
	}
	return cidrs
}

// getNodesIPs get all node IP from nodeMap
func (r *eciReconciler) getNodesIPs(nodeMap map[string]string) []string {
	nodeIPs := make([]string, 0)
	nodeIPMap := make(map[string]struct{})
	for _, v := range nodeMap {
		nodeIPMap[v] = struct{}{}
	}
	for ip := range nodeIPMap {
		nodeIPs = append(nodeIPs, ip)
	}
	return nodeIPs
}

// getEgressClusterInfo get EgressClusterInfo cr
func (r *eciReconciler) getEgressClusterInfo(ctx context.Context) error {
	return r.client.Get(ctx, types.NamespacedName{Name: defaultEgressClusterInfoName}, r.eci)
}

// updateEgressClusterInfo update EgressClusterInfo cr
func (r *eciReconciler) updateEgressClusterInfo(ctx context.Context) error {
	eci := new(egressv1beta1.EgressClusterInfo)
	err := r.client.Get(ctx, types.NamespacedName{Name: defaultEgressClusterInfoName}, eci)
	if err != nil {
		return err
	}
	r.eci.ResourceVersion = eci.ResourceVersion
	patch := client.MergeFrom(eci.DeepCopy())
	data, err := patch.Data(r.eci.DeepCopy())
	if err != nil {
		return err
	}
	r.log.V(1).Info("update", "patch.Data", string(data))
	return r.client.Status().Patch(ctx, r.eci, patch)
}

// getEgressIgnoreCIDRConfig get config about EgressIgnoreCIDR from egressgateway configmap
func (r *eciReconciler) getEgressIgnoreCIDRConfig() (string, bool, bool) {
	i := r.config.FileConfig.EgressIgnoreCIDR
	return i.PodCIDR, i.ClusterIP, i.NodeIP
}

// getServiceClusterIPRange get service-cluster-ip-range from kube controller manager
func (r *eciReconciler) getServiceClusterIPRange(pod *corev1.Pod) (ipv4Range, ipv6Range []string, err error) {
	return getCidr(pod, serviceClusterIpRange)
}

// getClusterCidr get cluster-cidr from kube controller manager
func (r *eciReconciler) getClusterCidr(pod *corev1.Pod) (ipv4Range, ipv6Range []string, err error) {
	return getCidr(pod, clusterCidr)
}

// getCidr get cidr value from kube controller manager
func getCidr(pod *corev1.Pod, param string) (ipv4Range, ipv6Range []string, err error) {
	containers := pod.Spec.Containers
	if len(containers) == 0 {
		return nil, nil, fmt.Errorf("failed to found containers")
	}
	commands := containers[0].Command
	ipRange := ""
	for _, c := range commands {
		if strings.Contains(c, param) {
			ipRange = strings.Split(c, "=")[1]
			break
		}
	}
	if len(ipRange) == 0 {
		return nil, nil, fmt.Errorf("failed to found %s", param)
	}
	// get cidr
	ipRanges := strings.Split(ipRange, ",")
	if len(ipRanges) == 1 {
		if isV4, _ := utils.IsIPv4Cidr(ipRanges[0]); isV4 {
			ipv4Range = ipRanges
			ipv6Range = []string{}
		}
		if isV6, _ := utils.IsIPv6Cidr(ipRanges[0]); isV6 {
			ipv6Range = ipRanges
			ipv4Range = []string{}

		}
	}
	if len(ipRanges) == 2 {
		ipv4Range, ipv6Range = ipRanges[:1], ipRanges[1:]
	}
	return
}

// getPod get pod by label
func getPod(c client.Client, label map[string]string) (*corev1.Pod, error) {
	podList := corev1.PodList{}
	opts := client.MatchingLabels(label)
	err := c.List(context.Background(), &podList, opts)
	if err != nil {
		return nil, err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return nil, fmt.Errorf("failed to get pod")
	}
	return &pods[0], nil
}

// checkCalicoExists once calico is detected, start watch
func checkCalicoExists(mgr manager.Manager, log logr.Logger, c controller.Controller) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	crdList := new(exv1.CustomResourceDefinitionList)
FOUND:
	for {
		err := mgr.GetClient().List(ctx, crdList)
		if err != nil {
			log.V(1).Info("failed list crd, try again", "details", err)
			continue
		}
		for i, item := range crdList.Items {
			log.V(1).Info("list crd", strconv.Itoa(i)+"th crd", item.Name)
			if item.Name == ippoolCrdName {
				log.V(1).Info("find calico ippool crd, egressClusterInfo controller begin to watch calico ippool")
			RETRY:
				if err = watchSource(c, source.Kind(mgr.GetCache(), &calicov1.IPPool{}), KindIPPool); err != nil {
					log.V(1).Info("failed watch calico ippool, try again", "details", err)
					time.Sleep(time.Millisecond * 100)
					goto RETRY
				}
				log.V(1).Info("egressClusterInfo controller succeeded to watch calico ippool")
				break FOUND
			}
		}
		time.Sleep(time.Second)
	}
}
