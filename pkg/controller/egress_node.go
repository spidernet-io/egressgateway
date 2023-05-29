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
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
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
	log         *zap.Logger
	config      *config.Config
	doOnce      sync.Once
	mark        markallocator.Interface
	allocatorV4 *ipallocator.Range
	allocatorV6 *ipallocator.Range
}

func (r *egReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Infof("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}

	r.doOnce.Do(func() {
		r.log.Sugar().Info("first reconcile of egressnode controller, init egressnode")
	redo:
		err := r.initEgressNode()
		if err != nil {
			r.log.Sugar().Errorf("init egreee node controller with error: %v", err)
			time.Sleep(time.Second)
			goto redo
		}
	})

	log := r.log.With(zap.String("name", newReq.Name), zap.String("kind", kind))
	log.Info("reconciling")
	switch kind {
	case "EgressNode":
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
func (r *egReconciler) reconcileEN(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	egressnode := new(egressv1.EgressNode)
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
			// For the existence of Node, when the user manually deletes EgressNode,
			// we first release the EgressNode and then regenerate it.
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

func cleanFinalizers(node *egressv1.EgressNode) {
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
func (r *egReconciler) reconcileNode(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
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
		egressNode := new(egressv1.EgressNode)
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

	en := new(egressv1.EgressNode)
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

func (r *egReconciler) createEgressNode(ctx context.Context, name string, log *zap.Logger) error {
	log.Sugar().Debug("try to create egress node")
	egressNode := &egressv1.EgressNode{ObjectMeta: metav1.ObjectMeta{
		Name:       name,
		Finalizers: []string{egressNodeFinalizers},
	}}
	err := r.client.Create(ctx, egressNode)
	if err != nil {
		return fmt.Errorf("failed to create egress node: %v", err)
	}
	log.Sugar().Debug("create egress node succeeded")
	return nil
}

func (r *egReconciler) releaseEgressNode(node egressv1.EgressNode, log *zap.Logger, commit func() error) error {
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
		log.Sugar().Debugf("try to release egress node mark: %v", node.Status.Mark)
		err := r.mark.Release(node.Status.Mark)
		if err != nil {
			return fmt.Errorf("failed to release egress node mark: %v", err)
		}
		log.Sugar().Debugf("release egress node mark succeeded: %v", node.Status.Mark)
		countNumMarkReleaseCalls.Inc()

		rollback = append(rollback, func() {
			_ = r.mark.Allocate(node.Status.Mark)
		})
	}
	if node.Status.Tunnel.IPv4 != "" && r.allocatorV4 != nil {
		log.Sugar().Debugf("try to release egress node tunnel ipv4: %v", node.Status.Tunnel.IPv4)
		ip := net.ParseIP(node.Status.Tunnel.IPv4)
		if ipv4 := ip.To4(); ipv4 != nil {
			err := r.allocatorV4.Release(ipv4)
			if err != nil {
				return fmt.Errorf("failed to release egress node tunnel ipv4: %v", err)
			}
			countNumIPReleaseCallsIpv4.Inc()
		}
		log.Sugar().Debugf("release egress node ipv4 succeeded: %v", node.Status.Tunnel.IPv4)

		rollback = append(rollback, func() {
			_ = r.allocatorV4.Allocate(ip)
		})
	}
	if node.Status.Tunnel.IPv6 != "" && r.allocatorV6 != nil {
		log.Sugar().Debugf("try to release egress node tunnel ipv6: %v", node.Status.Tunnel.IPv6)
		ip := net.ParseIP(node.Status.Tunnel.IPv6)
		if ipv6 := ip.To16(); ipv6 != nil {
			err := r.allocatorV6.Release(ipv6)
			if err != nil {
				return fmt.Errorf("failed to release egress node tunnel ipv6: %v", err)
			}
			countNumIPReleaseCallsIpv6.Inc()
		}
		log.Sugar().Debugf("release egress node ipv4 succeeded: %v", node.Status.Tunnel.IPv6)

		rollback = append(rollback, func() {
			_ = r.allocatorV6.Allocate(ip)
		})
	}

	return commit()
}

func (r *egReconciler) deleteEgressNode(node egressv1.EgressNode, log *zap.Logger) error {
	err := r.releaseEgressNode(node, log, func() error {
		log.Debug("try to delete egress node")
		err := r.client.Delete(context.Background(), &node)
		if err != nil {
			return err
		}
		log.Debug("delete egress node succeeded")
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *egReconciler) reBuildCache(node egressv1.EgressNode, log *zap.Logger) error {
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
		log.Sugar().Debugf("rebuild mark cache: %v", newNode.Status.Mark)
		err := r.mark.Allocate(newNode.Status.Mark)
		if err != nil {
			newNode.Status.Tunnel.MAC = ""
			needUpdate = true
			log.Sugar().Debugf("can't reused mark: %v", err)
		} else {
			log.Debug("rebuild mark cache succeeded")
			rollback = append(rollback, func() {
				if err := r.mark.Release(newNode.Status.Mark); err != nil {
					log.Sugar().Infof("rollback can't release %v: %v", newNode.Status.Mark, err)
				}
			})
		}
	}

	if newNode.Status.Tunnel.IPv4 != "" {
		log.Sugar().Debugf("rebuild ipv4 cache: %v", newNode.Status.Tunnel.IPv4)
		ip := net.ParseIP(newNode.Status.Tunnel.IPv4)
		if ipv4 := ip.To4(); ipv4 != nil {
			err := r.allocatorV4.Allocate(ipv4)
			if err != nil {
				newNode.Status.Tunnel.IPv4 = ""
				needUpdate = true
				log.Sugar().Debugf("can't reused ipv4: %v", err)
			} else {
				log.Debug("rebuild ipv4 cache succeeded")
				rollback = append(rollback, func() {
					if err := r.allocatorV4.Release(ip); err != nil {
						log.Sugar().Infof("rollback can't release ipv4 %v: %v", ip, err)
					}
				})
			}
		}
	}

	if newNode.Status.Tunnel.IPv6 != "" {
		log.Sugar().Debugf("rebuild ipv6 cache: %v", newNode.Status.Tunnel.IPv6)
		ip := net.ParseIP(newNode.Status.Tunnel.IPv6)
		if ipv6 := ip.To16(); ipv6 != nil {
			err := r.allocatorV6.Allocate(ipv6)
			if err != nil {
				if err == ipallocator.ErrAllocated {
					newNode.Status.Tunnel.IPv6 = ""
					needUpdate = true
					log.Sugar().Debugf("can't reused ipv6: %v", err)
				} else {
					log.Debug("rebuild ipv6 cache succeeded")
					rollback = append(rollback, func() {
						if err := r.allocatorV6.Release(ip); err != nil {
							log.Sugar().Infof("rollback can't release ipv6 %v: %v", ip, err)
						}
					})
				}
			}
		}
	}

	if needUpdate {
		log.Debug("try to update egress node")
		err := r.updateEgressNode(node)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress node: %v", err)
		}
		log.Debug("update egress node succeeded")
	}

	return nil
}

func (r *egReconciler) keepEgressNode(node egressv1.EgressNode, log *zap.Logger) error {
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
		log.Debug("try to generate new mac address")
		newNode.Status.Tunnel.MAC, err = generateMACAddress(newNode.Name)
		if err != nil {
			return err
		}
		log.Sugar().Debugf("generate new mac address succeeded: %v", newNode.Status.Tunnel.MAC)
	}

	if newNode.Status.Mark == "" {
		log.Debug("try to allocate next mark")
		newNode.Status.Mark, err = r.mark.AllocateNext()
		if err != nil {
			return fmt.Errorf("can't allocate next mark: %v", err)
		}
		countNumMarkAllocateNextCalls.Inc()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.mark.Release(newNode.Status.Mark); err != nil {
				log.Sugar().Infof("rollback can't release %v: %v", newNode.Status.Mark, err)
			}
		})
		log.Sugar().Debugf("allocate next ipv4 address succeeded: %v", newNode.Status.Tunnel.IPv4)
	}

	if newNode.Status.Tunnel.IPv4 == "" && r.allocatorV4 != nil {
		log.Debug("try to allocate next ipv4")
		ip, err := r.allocatorV4.AllocateNext()
		if err != nil {
			return fmt.Errorf("can't allocate next ipv4: %v", err)
		}
		countNumIPAllocateNextCallsIpv4.Inc()
		newNode.Status.Tunnel.IPv4 = ip.String()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.allocatorV4.Release(ip); err != nil {
				log.Sugar().Infof("rollback can't release ipv4 %v: %v", ip, err)
			}
		})
	}

	if newNode.Status.Tunnel.IPv6 == "" && r.allocatorV6 != nil {
		log.Debug("try to allocate next ipv6")
		ip, err := r.allocatorV6.AllocateNext()
		if err != nil {
			return fmt.Errorf("can't allocate next ipv6: %v", err)
		}
		countNumIPAllocateNextCallsIpv6.Inc()
		newNode.Status.Tunnel.IPv6 = ip.String()
		needUpdate = true
		rollback = append(rollback, func() {
			if err := r.allocatorV6.Release(ip); err != nil {
				log.Sugar().Infof("rollback can't release ipv6 %v: %v", ip, err)
			}
		})
		log.Sugar().Debugf("allocate next ipv6 address succeeded: %v", newNode.Status.Tunnel.IPv4)
	}

	if needUpdate {
		err := r.updateEgressNode(*newNode)
		if err != nil {
			return fmt.Errorf("rebuild failed to update egress node: %v", err)
		}
	}

	return nil
}

func (r *egReconciler) updateEgressNode(node egressv1.EgressNode) error {
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
	nodes := &egressv1.EgressNodeList{}
	err := r.client.List(context.Background(), nodes)
	if err != nil {
		return fmt.Errorf("failed to list node: %v", err)
	}

	start := time.Now()

	for _, node := range nodes.Items {
		log := r.log.With(
			zap.String("name", node.Name),
			zap.String("kind", "EgressNode"),
		)

		i := 0
		for {
			err := r.reBuildCache(node, log)
			if err != nil {
				log.Sugar().Errorf("failed to rebuild cache: %v. retry %d", err, i)
				time.Sleep(time.Second)
				continue
			}
			log.Sugar().Infof("succeeded to rebuild cache")
			break
		}
	}

	end := time.Now()
	delta := end.Sub(start)

	r.log.Sugar().Infof("rebuild egreeenode cache: total %d, speed %v", len(nodes.Items), delta)

	return nil
}

func newEgressNodeController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
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

	log.Sugar().Infof("new egressnode controller")
	c, err := controller.New("egressnode", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	log.Sugar().Infof("egressnode controller watch EgressNode")
	if err := c.Watch(&source.Kind{Type: &egressv1.EgressNode{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressNode"))); err != nil {
		return fmt.Errorf("failed to watch EgressNode: %w", err)
	}

	log.Sugar().Infof("egressnode controller watch Node")
	if err := c.Watch(&source.Kind{Type: &corev1.Node{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("Node"))); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	return nil
}
