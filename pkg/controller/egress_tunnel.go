// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cilium/ipam/service/ipallocator"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/markallocator"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

var (
	countNumIPAllocateNextCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "egress_ip_allocate_next_restore_calls",
		Help: "Total number of number of ip allocate next calls",
	}, []string{"version"})
	countNumIPReleaseCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "egress_ip_allocate_release_calls",
		Help: "Total number of number of ip release calls",
	}, []string{"version"})

	countNumIPAllocateNextCallsIpv4 = countNumIPAllocateNextCalls.WithLabelValues("ipv4")
	countNumIPAllocateNextCallsIpv6 = countNumIPAllocateNextCalls.WithLabelValues("ipv6")

	countNumIPReleaseCallsIpv4 = countNumIPReleaseCalls.WithLabelValues("ipv4")
	countNumIPReleaseCallsIpv6 = countNumIPReleaseCalls.WithLabelValues("ipv6")

	countNumMarkAllocateNextCalls = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "egress_mark_allocate_next_calls",
		Help: "Total number of mark allocate next count calls",
	})

	countNumMarkReleaseCalls = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "egress_mark_release_calls",
		Help: "Total number of mark release calls",
	})
)

var (
	egressTunnelFinalizers = "egressgateway.spidernet.io/egresstunnel"
)

func egressTunnelControllerMetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		countNumIPAllocateNextCalls,
		countNumIPReleaseCalls,
		countNumMarkAllocateNextCalls,
		countNumMarkReleaseCalls,
	}
}

type egReconciler struct {
	client      client.Client
	log         logr.Logger
	config      *config.Config
	doOnce      sync.Once
	mark        markallocator.Interface
	allocatorV4 *ipallocator.Range
	allocatorV6 *ipallocator.Range
	initDone    chan struct{}
	recorder    record.EventRecorder
}

func (r *egReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.doOnce.Do(func() {
		r.log.Info("first reconcile of egresstunnel controller, init egresstunnel")
	redo:
		err := r.initEgressTunnel()
		if err != nil {
			r.log.Error(err, "init egress tunnel controller with error")
			time.Sleep(time.Second)
			goto redo
		}
		r.initDone <- struct{}{}
	})

	log := r.log.WithValues("name", newReq.Name, "kind", kind)
	log.V(1).Info("reconciling")
	switch kind {
	case "EgressTunnel":
		return r.reconcileEGN(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEGN reconcile egress tunnel
// goal:
// - update egress tunnel
func (r *egReconciler) reconcileEGN(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	egresstunnel := new(egressv1.EgressTunnel)
	err := r.client.Get(ctx, req.NamespacedName, egresstunnel)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !egresstunnel.GetDeletionTimestamp().IsZero()

	if deleted {
		if len(egresstunnel.Finalizers) > 0 {
			// For the existence of Node, when the user manually deletes EgressTunnel,
			// we first release the EgressTunnel and then regenerate it.
			err := r.releaseEgressTunnel(*egresstunnel, log, func() error {
				cleanFinalizers(egresstunnel)
				err = r.client.Update(context.Background(), egresstunnel)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return reconcile.Result{Requeue: true}, err
			}
			return r.reconcileNode(ctx, req, log)
		}
		return reconcile.Result{Requeue: false}, nil
	}

	err = r.keepEgressTunnel(*egresstunnel, log)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{Requeue: false}, nil
}

func cleanFinalizers(node *egressv1.EgressTunnel) {
	for i, item := range node.Finalizers {
		if item == egressTunnelFinalizers {
			node.Finalizers = append(node.Finalizers[:i], node.Finalizers[i+1:]...)
		}
	}
}

// reconcileNode reconcile node
// not goal:
// - add    node
// - remove node
func (r *egReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	node := new(corev1.Node)
	err := r.client.Get(ctx, req.NamespacedName, node)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	if deleted {
		egressTunnel := new(egressv1.EgressTunnel)
		err := r.client.Get(ctx, req.NamespacedName, egressTunnel)
		if err != nil {
			if !k8serr.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{}, nil
		}
		err = r.deleteEgressTunnel(*egressTunnel, log)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{Requeue: false}, nil
	}

	egressTunnel := new(egressv1.EgressTunnel)
	err = r.client.Get(ctx, req.NamespacedName, egressTunnel)
	if err != nil {
		log.Info("create egress tunnel")
		if !k8serr.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		err := r.createEgressTunnel(ctx, node.Name, log)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	}

	// If GatewayFailover is not enabled, we watch the health status of the node,
	// and switch the active node of the egress IP based on the status of the node.
	if !r.config.FileConfig.GatewayFailover.Enable {
		phase := egressv1.EgressTunnelReady
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				if condition.Status != corev1.ConditionTrue {
					phase = egressv1.EgressTunnelNodeNotReady
				}
			}
		}
		// don't overwrite the EgressTunnelHeartbeatTimeout status
		if egressTunnel.Status.Phase != egressv1.EgressTunnelHeartbeatTimeout {
			if egressTunnel.Status.Phase != phase {
				log.Info("update egress tunnel", "status", egressTunnel.Status)
				egressTunnel.Status.Phase = phase
				err := r.updateEgressTunnel(*egressTunnel)
				if err != nil {
					return reconcile.Result{Requeue: true}, err
				}
			}
		}
	}

	return reconcile.Result{Requeue: false}, nil
}

func (r *egReconciler) createEgressTunnel(ctx context.Context, name string, log logr.Logger) error {
	log.V(1).Info("try to create egress tunnel")
	egressTunnel := &egressv1.EgressTunnel{ObjectMeta: metav1.ObjectMeta{
		Name:       name,
		Finalizers: []string{egressTunnelFinalizers},
	}}
	err := r.client.Create(ctx, egressTunnel)
	if err != nil {
		return fmt.Errorf("failed to create egress tunnel: %v", err)
	}
	log.V(1).Info("create egress tunnel succeeded")
	return nil
}

func (r *egReconciler) releaseEgressTunnel(node egressv1.EgressTunnel, log logr.Logger, commit func() error) error {
	rollback := make([]func(), 0)
	var err error

	defer func() {
		if err != nil {
			for _, f := range rollback {
				f()
			}
		}
	}()

	if node.Status.Mark != "" {
		log.V(1).Info("try to release egress tunnel mark", "mark", node.Status.Mark)
		err := r.mark.Release(node.Status.Mark)
		if err != nil {
			return fmt.Errorf("failed to release egress tunnel mark: %v", err)
		}
		log.V(1).Info("release egress tunnel mark succeeded", "mark", node.Status.Mark)
		countNumMarkReleaseCalls.Inc()

		rollback = append(rollback, func() {
			_ = r.mark.Allocate(node.Status.Mark)
		})
	}
	if node.Status.Tunnel.IPv4 != "" && r.allocatorV4 != nil {
		log.V(1).Info("try to release egress tunnel tunnel ipv4", "ipv4", node.Status.Tunnel.IPv4)

		ip := net.ParseIP(node.Status.Tunnel.IPv4)
		if ipv4 := ip.To4(); ipv4 != nil {
			err := r.allocatorV4.Release(ipv4)
			if err != nil {
				return fmt.Errorf("failed to release egress tunnel tunnel ipv4: %v", err)
			}
			countNumIPReleaseCallsIpv4.Inc()
		}
		log.V(1).Info("release egress tunnel ipv4 succeeded", "ipv4", node.Status.Tunnel.IPv4)

		rollback = append(rollback, func() {
			_ = r.allocatorV4.Allocate(ip)
		})
	}
	if node.Status.Tunnel.IPv6 != "" && r.allocatorV6 != nil {
		log.V(1).Info("try to release egress tunnel tunnel ipv6", "ipv6", node.Status.Tunnel.IPv6)
		ip := net.ParseIP(node.Status.Tunnel.IPv6)
		if ipv6 := ip.To16(); ipv6 != nil {
			err := r.allocatorV6.Release(ipv6)
			if err != nil {
				return fmt.Errorf("failed to release egress tunnel tunnel ipv6: %v", err)
			}
			countNumIPReleaseCallsIpv6.Inc()
		}
		log.V(1).Info("release egress tunnel ipv6 succeeded", "ipv6", node.Status.Tunnel.IPv6)

		rollback = append(rollback, func() {
			_ = r.allocatorV6.Allocate(ip)
		})
	}

	return commit()
}

func (r *egReconciler) deleteEgressTunnel(node egressv1.EgressTunnel, log logr.Logger) error {
	err := r.releaseEgressTunnel(node, log, func() error {
		log.V(1).Info("try to delete egress tunnel")
		err := r.client.Delete(context.Background(), &node)
		if err != nil {
			return err
		}
		log.V(1).Info("delete egress tunnel succeeded")
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *egReconciler) reBuildCache(node egressv1.EgressTunnel, log logr.Logger) error {
	rollback := make([]func(), 0)
	var err error
	needUpdate := false
	newNode := node.DeepCopy()

	defer func() {
		if err != nil {
			for _, f := range rollback {
				f()
			}
		}
	}()

	if newNode.Status.Mark != "" {
		log.V(1).Info("rebuild mark cache", "mark", newNode.Status.Mark)
		err := r.mark.Allocate(newNode.Status.Mark)
		if err != nil {
			newNode.Status.Tunnel.MAC = ""
			needUpdate = true
			log.V(1).Error(err, "can't reused mark")
		} else {
			log.V(1).Info("rebuild mark cache succeeded")
			rollback = append(rollback, func() {
				if err := r.mark.Release(newNode.Status.Mark); err != nil {
					log.Error(err, "rollback can't release", "mark", newNode.Status.Mark)
				}
			})
		}
	}

	if newNode.Status.Tunnel.IPv4 != "" && r.allocatorV4 != nil {
		log.V(1).Info("rebuild ipv4 cache", "ipv4", newNode.Status.Tunnel.IPv4)
		ip := net.ParseIP(newNode.Status.Tunnel.IPv4)
		if ipv4 := ip.To4(); ipv4 != nil {
			err := r.allocatorV4.Allocate(ipv4)
			if err != nil {
				log.Error(err, "can't reused ipv4", "ipv4", ipv4)
				newNode.Status.Tunnel.IPv4 = ""
				needUpdate = true
			} else {
				log.V(1).Info("rebuild ipv4 cache succeeded")
				rollback = append(rollback, func() {
					if err := r.allocatorV4.Release(ip); err != nil {
						log.Error(err, "rollback can't release ipv4", "ipv4", ip)
					}
				})
			}
		}
	} else if r.allocatorV4 == nil && newNode.Status.Tunnel.IPv4 != "" {
		needUpdate = true
		newNode.Status.Tunnel.IPv4 = ""
	}

	if newNode.Status.Tunnel.IPv6 != "" && r.allocatorV6 != nil {
		log.V(1).Info("rebuild ipv4 cache", "ipv4", newNode.Status.Tunnel.IPv6)
		ip := net.ParseIP(newNode.Status.Tunnel.IPv6)
		if ipv6 := ip.To16(); ipv6 != nil {
			err := r.allocatorV6.Allocate(ipv6)
			if err != nil {
				if errors.Is(err, ipallocator.ErrAllocated) {
					log.Error(err, "can't reused ipv6", "ipv6", ipv6)
					newNode.Status.Tunnel.IPv6 = ""
					needUpdate = true
				} else {
					log.Info("rebuild ipv6 cache succeeded")
					rollback = append(rollback, func() {
						if err := r.allocatorV6.Release(ip); err != nil {
							log.Error(err, "rollback can't release ipv6", "ipv6", ip)
						}
					})
				}
			}
		}
	} else if r.allocatorV6 == nil && newNode.Status.Tunnel.IPv6 != "" {
		needUpdate = true
		newNode.Status.Tunnel.IPv6 = ""
	}

	if needUpdate {
		log.V(1).Info("try to update egress tunnel")
		err := r.updateEgressTunnel(*newNode)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress tunnel: %v", err)
		}
		log.V(1).Info("update egress tunnel succeeded")
	}

	return nil
}

func (r *egReconciler) keepEgressTunnel(node egressv1.EgressTunnel, log logr.Logger) error {
	rollback := make([]func(), 0)
	var err error
	needUpdate := false
	newNode := node.DeepCopy()

	defer func() {
		if err != nil {
			for _, f := range rollback {
				f()
			}
		}
	}()

	if newNode.Status.Tunnel.MAC == "" {
		log.V(1).Info("try to generate new mac address")
		newNode.Status.Tunnel.MAC, err = generateMACAddress(newNode.Name)
		if err != nil {
			return err
		}
		log.V(1).Info("generate new mac address succeeded", "mac", newNode.Status.Tunnel.MAC)
	}

	if newNode.Status.Mark == "" {
		log.V(1).Info("try to allocate next mark")
		newNode.Status.Mark, err = r.mark.AllocateNext()
		if err != nil {
			return fmt.Errorf("can't allocate next mark: %v", err)
		}
		countNumMarkAllocateNextCalls.Inc()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.mark.Release(newNode.Status.Mark); err != nil {
				log.Error(err, "rollback can't release", "mark", newNode.Status.Mark)
			}
		})
		log.V(1).Info("allocate next ipv4 address succeeded", "ipv4", newNode.Status.Tunnel.IPv4)
	}

	if newNode.Status.Tunnel.IPv4 == "" && r.allocatorV4 != nil {
		log.V(1).Info("try to allocate next ipv4")
		ip, err := r.allocatorV4.AllocateNext()
		if err != nil {
			return fmt.Errorf("can't allocate next ipv4: %v", err)
		}
		countNumIPAllocateNextCallsIpv4.Inc()
		newNode.Status.Tunnel.IPv4 = ip.String()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.allocatorV4.Release(ip); err != nil {
				log.Error(err, "rollback can't release ipv4", "ip", ip)
			}
		})
	}

	if newNode.Status.Tunnel.IPv6 == "" && r.allocatorV6 != nil {
		log.V(1).Info("try to allocate next ipv6")
		ip, err := r.allocatorV6.AllocateNext()
		if err != nil {
			log.Error(err, "can't allocate next ipv6")
		}
		countNumIPAllocateNextCallsIpv6.Inc()
		newNode.Status.Tunnel.IPv6 = ip.String()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.allocatorV6.Release(ip); err != nil {
				log.Error(err, "rollback can't release ipv6", "ip", ip)
			}
		})
		log.V(1).Info("allocate next ipv6 address succeeded", "ipv6", ip)
	}

	if needUpdate {
		err := r.updateEgressTunnel(*newNode)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress tunnel: %v", err)
		}
	}

	return nil
}

func (r *egReconciler) updateEgressTunnel(node egressv1.EgressTunnel) error {
	if node.Status.Phase == "" {
		node.Status.Phase = egressv1.EgressTunnelInit
	}
	if node.Status.Tunnel.Parent.Name == "" {
		node.Status.Phase = egressv1.EgressTunnelInit
	}
	if node.Status.Mark == "" {
		node.Status.Phase = egressv1.EgressTunnelPending
	}
	if node.Status.Tunnel.IPv4 == "" && r.allocatorV4 != nil {
		node.Status.Phase = egressv1.EgressTunnelPending
	}
	if node.Status.Tunnel.IPv6 == "" && r.allocatorV6 != nil {
		node.Status.Phase = egressv1.EgressTunnelPending
	}
	if node.Status.Tunnel.MAC == "" {
		node.Status.Phase = egressv1.EgressTunnelPending
	}

	err := r.client.Status().Update(context.Background(), &node)
	if err != nil {
		return fmt.Errorf("rebuild failed to update egress tunnel: %v", err)
	}
	return nil
}

func generateMACAddress(nodeName string) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(nodeName + "egress"))
	if err != nil {
		return "", err
	}
	sha := h.Sum(nil)
	hw := net.HardwareAddr(append([]byte("f"), sha[0:5]...))
	return hw.String(), nil
}

func (r *egReconciler) initEgressTunnel() error {
	nodes := &egressv1.EgressTunnelList{}
	err := r.client.List(context.Background(), nodes)
	if err != nil {
		return fmt.Errorf("failed to list node: %v", err)
	}

	start := time.Now()

	for _, node := range nodes.Items {
		log := r.log.WithValues("name", node.Name, "kind", "EgressTunnel")

		i := 0
		for {
			err := r.reBuildCache(node, log)
			if err != nil {
				log.Error(err, "failed to rebuild cache", "retry", i)
				time.Sleep(time.Second)
				continue
			}
			log.Info("succeeded to rebuild cache")
			break
		}
	}

	end := time.Now()
	delta := end.Sub(start)

	r.log.Info("rebuild egresstunnel cache", "total", len(nodes.Items), "speed", delta)

	return nil
}

func (r *egReconciler) healthCheck(ctx context.Context) {
	r.log.Info("health check", "second", r.config.FileConfig.GatewayFailover.TunnelMonitorPeriod)
	period := time.Second * time.Duration(r.config.FileConfig.GatewayFailover.TunnelMonitorPeriod)
	t := time.NewTimer(period)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := r.tunnelListCheck(ctx)
			if err != nil {
				r.log.Error(err, "tunnel list check")
				t.Reset(time.Second)
				continue
			}
			t.Reset(period)
		}
	}
}

func (r *egReconciler) tunnelListCheck(ctx context.Context) error {
	timeout := time.Second * time.Duration(r.config.FileConfig.GatewayFailover.EipEvictionTimeout)

	startTime := time.Now()
	r.log.V(1).Info("start tunnel list health check", "time", startTime)
	defer func() {
		endTime := time.Now()
		elapsedTime := endTime.Sub(startTime)
		r.log.V(1).Info("end tunnel list health check", "time", endTime, "spend_time", elapsedTime)
	}()

	tunnels := new(egressv1.EgressTunnelList)
	if err := r.client.List(ctx, tunnels); err != nil {
		return err
	}

	for _, item := range tunnels.Items {
		tunnel := new(egressv1.EgressTunnel)
		key := types.NamespacedName{Name: item.Name}
		err := r.client.Get(ctx, key, tunnel)
		if err != nil {
			r.log.Error(err, "get tunnel")
			continue
		}

		if time.Now().After(tunnel.Status.LastHeartbeatTime.Add(timeout)) {
			if tunnel.Status.Phase == egressv1.EgressTunnelHeartbeatTimeout {
				continue
			}
			tunnel.Status.Phase = egressv1.EgressTunnelHeartbeatTimeout
			r.log.Info("update tunnel status to HeartbeatTimeout", "tunnel", tunnel.Name)
			err := r.client.Status().Update(ctx, tunnel)
			if err != nil {
				r.log.Error(err, "update tunnel status to TunnelHeartbeatTimeout")
				continue
			}

			r.recorder.Event(
				tunnel, corev1.EventTypeNormal,
				egressv1.ReasonStatusChanged,
				"EgressTunnel status changes to HeartbeatTimeout.",
			)
		}
	}
	return nil
}

func (r *egReconciler) Start(ctx context.Context) error {
	if r.config.FileConfig.GatewayFailover.Enable {
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-r.initDone:
				go r.healthCheck(ctx)
			}
		}()
	}
	return nil
}

func newEgressTunnelController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	mark, err := markallocator.NewAllocatorMarkRange(cfg.FileConfig.Mark)
	if err != nil {
		return fmt.Errorf("markallocator.NewAllocatorCID with error: %v", err)
	}

	r := &egReconciler{
		client:   mgr.GetClient(),
		log:      log,
		config:   cfg,
		doOnce:   sync.Once{},
		mark:     mark,
		initDone: make(chan struct{}, 1),
	}

	if cfg.FileConfig.EnableIPv4 {
		_, cidr, err := net.ParseCIDR(cfg.FileConfig.TunnelIpv4Subnet)
		if err != nil {
			return err
		}
		r.allocatorV4, err = ipallocator.NewCIDRRange(cidr)
		if err != nil {
			return fmt.Errorf("ipallocator.NewCIDRRange with error: %v", err)
		}
	}
	if cfg.FileConfig.EnableIPv6 {
		_, cidr, err := net.ParseCIDR(cfg.FileConfig.TunnelIpv6Subnet)
		if err != nil {
			return err
		}
		r.allocatorV6, err = ipallocator.NewCIDRRange(cidr)
		if err != nil {
			return fmt.Errorf("ipallocator.NewCIDRRange with error: %v", err)
		}
	}

	log.Info("new egresstunnel controller")
	c, err := controller.New("egresstunnel", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	err = mgr.Add(r)
	if err != nil {
		return err
	}

	log.Info("egresstunnel controller watch EgressTunnel")
	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressTunnel{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressTunnel"))); err != nil {
		return fmt.Errorf("failed to watch EgressTunnel: %w", err)
	}

	log.Info("egresstunnel controller watch Node")
	if err := c.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	r.recorder = mgr.GetEventRecorderFor("egress-tunnel")

	return nil
}
