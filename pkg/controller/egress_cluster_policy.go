// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type egcpReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r *egcpReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Errorf("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}

	log := r.log.With(zap.String("name", newReq.Name), zap.String("kind", kind))
	log.Info("egressclusterpolicy controller: reconciling")
	switch kind {
	case "EgressGateway":
		return r.reconcileEG(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEN reconcile egressgateway
// goal:
// - update egressclusterpolicy
func (r *egcpReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	egw := new(egressv1.EgressGateway)
	err := r.client.Get(ctx, req.NamespacedName, egw)
	if err != nil {
		if !errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, err
		}
		deleted = true
	}
	deleted = deleted || !egw.GetDeletionTimestamp().IsZero()

	if deleted {
		return reconcile.Result{Requeue: false}, nil
	}

	egcpList := &egressv1.EgressClusterPolicyList{}
	if err := r.client.List(ctx, egcpList); err != nil {
		r.log.Sugar().Errorf("egcp->controller, event: eg(%v): Failed to get EgressClusterPolicyList\n", egw.Name)
		return reconcile.Result{Requeue: true}, err
	}

	for _, item := range egcpList.Items {
		policy := egressv1.Policy{Name: item.Name, Namespace: item.Namespace}
		eipStatus, isExist := egressgateway.GetEIPStatusByPolicy(policy, *egw)
		if !isExist {
			continue
		}

		newEGCP := item.DeepCopy()
		for _, eip := range eipStatus.Eips {
			for _, p := range eip.Policies {
				if p == policy {
					newEGCP.Status.Eip.Ipv4 = eip.IPv4
					newEGCP.Status.Eip.Ipv6 = eip.IPv6
					newEGCP.Status.Node = eipStatus.Name
				}
			}
		}

		r.log.Sugar().Debugf("update egressclusterpolicy status\n%v", newEGCP.Status)
		err = r.client.Status().Update(ctx, newEGCP)
		if err != nil {
			r.log.Sugar().Errorf("update egressclusterpolicy status\n%v", newEGCP.Status)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: false}, nil
}

func newEgressClusterPolicyController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	r := &egcpReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}

	log.Sugar().Infof("new egressclusterpolicy controller")
	c, err := controller.New("egressclusterpolicy", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	return nil
}
