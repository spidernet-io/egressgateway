// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cilium/ipam/service/ipallocator"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
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
	egressNodeFinalizers = "egressgateway.spidernet.io/egressnode"
)

func egressNodeControllerMetricCollectors() []prometheus.Collector {
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
}

func (r *egReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.doOnce.Do(func() {
		r.log.Info("first reconcile of egressnode controller, init egressnode")
	redo:
		err := r.initEgressNode()
		if err != nil {
			r.log.Error(err, "init egress node controller with error")
			time.Sleep(time.Second)
			goto redo
		}
	})

	log := r.log.WithValues("name", newReq.Name, "kind", kind)
	log.Info("reconciling")
	switch kind {
	case "EgressTunnel":
		return r.reconcileEN(ctx, newReq, log)
	case "Node":
		return r.reconcileNode(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEN reconcile egress node
// goal:
// - update egress node
func (r *egReconciler) reconcileEN(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	egressnode := new(egressv1.EgressTunnel)
	err := r.client.Get(ctx, req.NamespacedName, egressnode)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !egressnode.GetDeletionTimestamp().IsZero()

	if deleted {
		if len(egressnode.Finalizers) > 0 {
			// For the existence of Node, when the user manually deletes EgressTunnel,
			// we first release the EgressTunnel and then regenerate it.
			err := r.releaseEgressNode(*egressnode, log, func() error {
				cleanFinalizers(egressnode)
				err = r.client.Update(context.Background(), egressnode)
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

	err = r.keepEgressNode(*egressnode, log)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{Requeue: false}, nil
}

func cleanFinalizers(node *egressv1.EgressTunnel) {
	for i, item := range node.Finalizers {
		if item == egressNodeFinalizers {
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
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !node.GetDeletionTimestamp().IsZero()

	if deleted {
		egressNode := new(egressv1.EgressTunnel)
		err := r.client.Get(ctx, req.NamespacedName, egressNode)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{}, nil
		}
		err = r.deleteEgressNode(*egressNode, log)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{Requeue: false}, nil
	}

	en := new(egressv1.EgressTunnel)
	err = r.client.Get(ctx, req.NamespacedName, en)
	if err != nil {
		log.Info("create egress node")
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		err := r.createEgressNode(ctx, node.Name, log)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		return reconcile.Result{}, nil
	}

	return reconcile.Result{Requeue: false}, nil
}

func (r *egReconciler) createEgressNode(ctx context.Context, name string, log logr.Logger) error {
	log.V(1).Info("try to create egress node")
	egressNode := &egressv1.EgressTunnel{ObjectMeta: metav1.ObjectMeta{
		Name:       name,
		Finalizers: []string{egressNodeFinalizers},
	}}
	err := r.client.Create(ctx, egressNode)
	if err != nil {
		return fmt.Errorf("failed to create egress node: %v", err)
	}
	log.V(1).Info("create egress node succeeded")
	return nil
}

func (r *egReconciler) releaseEgressNode(node egressv1.EgressTunnel, log logr.Logger, commit func() error) error {
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
		log.V(1).Info("try to release egress node mark", "mark", node.Status.Mark)
		err := r.mark.Release(node.Status.Mark)
		if err != nil {
			return fmt.Errorf("failed to release egress node mark: %v", err)
		}
		log.V(1).Info("release egress node mark succeeded", "mark", node.Status.Mark)
		countNumMarkReleaseCalls.Inc()

		rollback = append(rollback, func() {
			_ = r.mark.Allocate(node.Status.Mark)
		})
	}
	if node.Status.Tunnel.IPv4 != "" && r.allocatorV4 != nil {
		log.V(1).Info("try to release egress node tunnel ipv4", "ipv4", node.Status.Tunnel.IPv4)

		ip := net.ParseIP(node.Status.Tunnel.IPv4)
		if ipv4 := ip.To4(); ipv4 != nil {
			err := r.allocatorV4.Release(ipv4)
			if err != nil {
				return fmt.Errorf("failed to release egress node tunnel ipv4: %v", err)
			}
			countNumIPReleaseCallsIpv4.Inc()
		}
		log.V(1).Info("release egress node ipv4 succeeded", "ipv4", node.Status.Tunnel.IPv4)

		rollback = append(rollback, func() {
			_ = r.allocatorV4.Allocate(ip)
		})
	}
	if node.Status.Tunnel.IPv6 != "" && r.allocatorV6 != nil {
		log.V(1).Info("try to release egress node tunnel ipv6", "ipv6", node.Status.Tunnel.IPv6)
		ip := net.ParseIP(node.Status.Tunnel.IPv6)
		if ipv6 := ip.To16(); ipv6 != nil {
			err := r.allocatorV6.Release(ipv6)
			if err != nil {
				return fmt.Errorf("failed to release egress node tunnel ipv6: %v", err)
			}
			countNumIPReleaseCallsIpv6.Inc()
		}
		log.V(1).Info("release egress node ipv6 succeeded", "ipv6", node.Status.Tunnel.IPv6)

		rollback = append(rollback, func() {
			_ = r.allocatorV6.Allocate(ip)
		})
	}

	return commit()
}

func (r *egReconciler) deleteEgressNode(node egressv1.EgressTunnel, log logr.Logger) error {
	err := r.releaseEgressNode(node, log, func() error {
		log.V(1).Info("try to delete egress node")
		err := r.client.Delete(context.Background(), &node)
		if err != nil {
			return err
		}
		log.V(1).Info("delete egress node succeeded")
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
				if err == ipallocator.ErrAllocated {
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
		log.V(1).Info("try to update egress node")
		err := r.updateEgressNode(*newNode)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress node: %v", err)
		}
		log.V(1).Info("update egress node succeeded")
	}

	return nil
}

func (r *egReconciler) keepEgressNode(node egressv1.EgressTunnel, log logr.Logger) error {
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
		err := r.updateEgressNode(*newNode)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress node: %v", err)
		}
	}

	return nil
}

func (r *egReconciler) updateEgressNode(node egressv1.EgressTunnel) error {
	phase := egressv1.EgressNodeInit
	if node.Status.Tunnel.Parent.Name == "" {
		phase = egressv1.EgressNodeInit
	}
	if node.Status.Mark == "" {
		phase = egressv1.EgressNodePending
	}
	if node.Status.Tunnel.IPv4 == "" && r.allocatorV4 != nil {
		phase = egressv1.EgressNodePending
	}
	if node.Status.Tunnel.IPv6 == "" && r.allocatorV6 != nil {
		phase = egressv1.EgressNodePending
	}
	if node.Status.Tunnel.MAC == "" {
		phase = egressv1.EgressNodePending
	}
	node.Status.Phase = phase

	err := r.client.Status().Update(context.Background(), &node)
	if err != nil {
		return fmt.Errorf("rebuild failed to update egress node: %v", err)
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

func (r *egReconciler) initEgressNode() error {
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

	r.log.Info("rebuild egressnode cache", "total", len(nodes.Items), "speed", delta)

	return nil
}

func newEgressNodeController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	mark, err := markallocator.NewAllocatorMarkRange(cfg.FileConfig.Mark)
	if err != nil {
		return fmt.Errorf("markallocator.NewAllocatorCID with error: %v", err)
	}

	r := &egReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
		doOnce: sync.Once{},
		mark:   mark,
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

	log.Info("new egressnode controller")
	c, err := controller.New("egressnode", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	log.Info("egressnode controller watch EgressTunnel")
	if err := c.Watch(source.Kind(mgr.GetCache(), &egressv1.EgressTunnel{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressTunnel"))); err != nil {
		return fmt.Errorf("failed to watch EgressTunnel: %w", err)
	}

	log.Info("egressnode controller watch Node")
	if err := c.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	return nil
}
