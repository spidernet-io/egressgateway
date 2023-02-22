// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/ipam"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type egReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
	doOnce sync.Once
	ipam   *ipam.TunnleIpam
}

func (r *egReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Infof("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}
	log := r.log.With(
		zap.String("namespacedName", newReq.NamespacedName.String()),
		zap.String("kind", kind),
	)

	r.doOnce.Do(func() {
		r.log.Sugar().Info("first reconcile of egressnode controller, init egressnode")
	redo:
		err := r.initEgressNode()
		if err != nil {
			r.log.Sugar().Errorf("first reconcile of egressnode controller, init egressnode, with error: %v", err)
			goto redo
		}
	})

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
		log.Sugar().Infof("reconcileNode: Delete %v event", req.Name)
		en := new(egressv1.EgressNode)
		en.Name = req.Name
		err := r.client.Delete(ctx, en)
		if err != nil {
			if !errors.IsNotFound(err) {
				return reconcile.Result{Requeue: true}, err
			}
		}
		return reconcile.Result{}, nil
	}

	log.Sugar().Infof("reconcileNode: Update %v event", req.Name)
	en := new(egressv1.EgressNode)
	err = r.client.Get(ctx, req.NamespacedName, en)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		en.Name = req.Name
		en.Status.Phase = "Pending"

		r.log.Sugar().Infof("create egressnode=%v", en)
		err = r.client.Create(context.Background(), en)
		if err != nil {
			log.Sugar().Errorln("Failed to create an EgressNode；err(%v)", err)
			return reconcile.Result{Requeue: true}, err
		}
	} else {
		var egIP, egIPv6 string
		if r.ipam.EnableIPv4 {
			egIP = en.Status.VxlanIPv4
		}

		if r.ipam.EnableIPv6 {
			egIPv6 = en.Status.VxlanIPv6
		}

		if (r.ipam.EnableIPv4 && egIP == "") || (r.ipam.EnableIPv6 && egIPv6 == "") {
			log.Sugar().Infof("EnableIPv4=%v, egIP=%v, EnableIPv6=%v, egIPv6=%v; en.Status.Phase set Pending", r.ipam.EnableIPv4, egIP, r.ipam.EnableIPv6, egIPv6)
			en.Status.Phase = "Pending"
		} else {
			if (r.ipam.EnableIPv4 && !r.ipam.CheckIsOK(egIP)) || (r.ipam.EnableIPv6 && !r.ipam.CheckIsOK(egIPv6)) {
				log.Sugar().Infof("ip check failure; EnableIPv4=%v, egIP=%v, EnableIPv6=%v, egIPv6=%v; en.Status.Phase set Pending", r.ipam.EnableIPv4, egIP, r.ipam.EnableIPv6, egIPv6)
				en.Status.Phase = "Pending"
			} else {
				var node, nodeByIPv6 string
				if r.ipam.EnableIPv4 {
					node = r.ipam.GetNode(egIP)
					nodeByIPv6 = node
				}

				if r.ipam.EnableIPv6 {
					nodeByIPv6 = r.ipam.GetNode(egIPv6)
					if node == "" {
						node = nodeByIPv6
					}
				}

				if node != nodeByIPv6 || node != en.Name {
					log.Sugar().Infof("The node(%v) bound to IPv4(%v) and IPV6(%v) is incorrect ", en.Name, node, nodeByIPv6)
					en.Status.Phase = "Pending"
					en.Status.VxlanIPv4 = ""
					en.Status.VxlanIPv6 = ""
				} else {
					log.Sugar().Infof("egressnode(%v) is in Init state", en)
					en.Status.Phase = egressv1.EgressNodeInit

					// To be determined
					// if egIP != "" {
					// 	if err = r.ipam.SetNodeIP(egIP, en.Name); err != nil {
					// 		return reconcile.Result{Requeue: true}, err
					// 	}
					// }

					// if egIPv6 != "" {
					// 	if err = r.ipam.SetNodeIPv6(egIPv6, en.Name); err != nil {
					// 		return reconcile.Result{Requeue: true}, err
					// 	}
					// }

				}
			}
		}
	}

	log.Sugar().Infof("update egressnode=%v", en)
	err = r.client.Status().Update(ctx, en)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	return reconcile.Result{}, nil
}

// reconcileEN reconcile egress node
// goal:
// - add node
// - update node
// - remove node
func (r *egReconciler) reconcileEN(ctx context.Context,
	req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	en := &egressv1.EgressNode{}
	err := r.client.Get(ctx, req.NamespacedName, en)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !en.GetDeletionTimestamp().IsZero()

	if deleted {
		log.Sugar().Infof("reconcileEN: Delete %v event", req.Name)
		node := new(corev1.Node)
		err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, node)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Sugar().Infof("not found node(%v)", req.Name)
				if err := r.ipam.ReleaseByNode(req.Name); err != nil {
					log.Sugar().Errorf("EgressNode(%v) failed to release ip", en.Name)
					return reconcile.Result{Requeue: true}, err
				}
				return reconcile.Result{}, nil
			}
			return reconcile.Result{Requeue: true}, err
		}

		log.Sugar().Infof("If the node(%v) exists, the IP address is released", req.Name)
		if r.ipam.EnableIPv4 {
			if err := r.ipam.ReleaseByNode(req.Name); err != nil {
				log.Sugar().Errorf("EgressNode(%v) failed to release ip", en.Name)
				return reconcile.Result{Requeue: true}, err
			}
		}

		if r.ipam.EnableIPv6 {
			if err := r.ipam.ReleaseIPv6ByNode(req.Name); err != nil {
				log.Sugar().Errorf("EgressNode(%v) failed to release ip", en.Name)
				return reconcile.Result{Requeue: true}, err
			}
		}

		if node.GetDeletionTimestamp().IsZero() {
			log.Sugar().Infof("If the node(%v) exists and is not deleted, create a new EgressNode", req.Name)
			newEn := new(egressv1.EgressNode)
			newEn.Name = req.Name
			newEn.Status.Phase = "Pending"
			r.log.Sugar().Infof("create egressnode=%v", newEn)
			err = r.client.Create(ctx, newEn)
			if err != nil {
				log.Sugar().Errorf("Failed to create an EgressNode；err(%v)", err)
				return reconcile.Result{Requeue: true}, err
			}
		}

		return reconcile.Result{}, nil
	}

	log.Sugar().Infof("reconcileEN: Update %v event, en.Status.Phase=%v", req.Name, en.Status.Phase)
	if en.Status.Phase != egressv1.EgressNodeInit && en.Status.Phase != egressv1.EgressNodeSucceeded {
		log.Sugar().Infof("If the EgressNode(%v) is not in Init or Succeeded state, an IP address is assigned", en.Name)
		if r.ipam.EnableIPv4 {
			ip, err := r.ipam.Acquire(en.Name)
			if err != nil {
				log.Error("EgressNode failed to request IP")
				en.Status.Phase = "Failed"

				if updateErr := r.client.Status().Update(ctx, en); updateErr != nil {
					log.Sugar().Errorf("Description Failed to update the status to Failed")
				}
				return reconcile.Result{Requeue: true}, err
			}
			en.Status.VxlanIPv4 = ip.String()
		}

		if r.ipam.EnableIPv6 {
			ip, err := r.ipam.AcquireIPv6(en.Name)
			if err != nil {
				log.Error("EgressNode failed to request IPV6")
				en.Status.Phase = "Failed"
				if r.ipam.EnableIPv4 {
					if err := r.ipam.ReleaseByIP(en.Status.VxlanIPv4); err != nil {
						log.Sugar().Errorf("IP(%v) release failure", en.Status.VxlanIPv4)
					}
				}

				if updateErr := r.client.Status().Update(ctx, en); updateErr != nil {
					log.Sugar().Errorf("Description Failed to update the status to Failed")
				}
				return reconcile.Result{Requeue: true}, err
			}
			en.Status.VxlanIPv6 = ip.String()
		}

		en.Status.TunnelMac, err = GenerateMACAddress(en.Name)
		if err != nil {
			log.Sugar().Errorf("%v hardware address generation failed", en.Name)
			if r.ipam.EnableIPv4 {
				if err := r.ipam.ReleaseByIP(en.Status.VxlanIPv4); err != nil {
					log.Sugar().Errorf("IP(%v) release failure", en.Status.VxlanIPv4)
				}
			}

			if r.ipam.EnableIPv6 {
				if err := r.ipam.ReleaseByIP(en.Status.VxlanIPv6); err != nil {
					log.Sugar().Errorf("IP(%v) release failure", en.Status.VxlanIPv4)
				}
			}

			en.Status.Phase = egressv1.EgressNodeFailed
			if updateErr := r.client.Status().Update(ctx, en); updateErr != nil {
				log.Sugar().Errorf("Description Failed to update the status to Failed")
			}
			return reconcile.Result{Requeue: true}, err
		}
		log.Sugar().Infof("The Mac address generated for the node(%v) is %v", en.Name, en.Status.TunnelMac)

		en.Status.Phase = egressv1.EgressNodeInit

		log.Sugar().Infof("update egressnode=%v", en)
		err = r.client.Status().Update(ctx, en)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func newEgressNodeController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	r := &egReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
		doOnce: sync.Once{},
		ipam:   &ipam.TunnleIpam{EnableIPv4: cfg.FileConfig.EnableIPv4, EnableIPv6: cfg.FileConfig.EnableIPv6},
	}

	if err := r.ipam.Init(cfg.FileConfig.TunnelIpv4Subnet, cfg.FileConfig.TunnelIpv6Subnet, log); err != nil {
		log.Error("Failed to initialize the tunnel")
		return err
	}

	log.Sugar().Infof("new egressnode controller")
	c, err := controller.New("egressNode", mgr,
		controller.Options{Reconciler: r})
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

func GenerateMACAddress(nodeName string) (string, error) {
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
	r.log.Sugar().Infof("Start initEgressNode")
	nodes := &corev1.NodeList{}
	err := r.client.List(context.Background(), nodes)
	if err != nil {
		return fmt.Errorf("Failed to obtain the node list（err: %v）", err)
	}

	for _, node := range nodes.Items {
		r.log.Sugar().Infof("Init egressnode=%v", node.Name)
		en := new(egressv1.EgressNode)
		err := r.client.Get(context.Background(), types.NamespacedName{Namespace: node.Namespace, Name: node.Name}, en)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("Failed to obtain the EgressNode；err(%v)", err)
			}
			en.Name = node.Name
			en.Status.Phase = "Pending"
			r.log.Sugar().Infof("create egressnode=%v", en)
			err = r.client.Create(context.Background(), en)
			if err != nil {
				r.log.Sugar().Errorf("Failed to create an EgressNode(%v)", en.Name)
				return fmt.Errorf("Failed to create an EgressNode；err(%v)", err)
			}
		}

		ip := en.Status.VxlanIPv4
		ipv6 := en.Status.VxlanIPv6

		r.log.Sugar().Infof("name=%v, ipv4=%v, ipv6=%v", en.Name, ip, ipv6)

		if ip != "" {
			if err := r.ipam.SetNodeIP(ip, en.Name); err != nil {
				r.log.Sugar().Errorf("IP(%v) failed to bind egressnode(%v); err(%v)", ip, en.Name, err)
				return err
			}
		}

		if ipv6 != "" {
			if err := r.ipam.SetNodeIPv6(ipv6, en.Name); err != nil {
				r.log.Sugar().Errorf("IP(%v) failed to bind egressnode(%v); err(%v)", ip, en.Name, err)
				return err
			}
		}
	}

	return nil
}
