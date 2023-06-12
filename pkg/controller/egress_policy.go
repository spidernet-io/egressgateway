// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"

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
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type egpReconciler struct {
	client client.Client
	log    *zap.Logger
	config *config.Config
}

func (r *egpReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		r.log.Sugar().Errorf("parse req(%v) with error: %v", req, err)
		return reconcile.Result{}, err
	}

	log := r.log.With(zap.String("name", newReq.Name), zap.String("kind", kind))
	log.Info("egresspolicy controller: reconciling")
	switch kind {
	case "EgressGateway":
		return r.reconcileEG(ctx, newReq, log)
	default:
		return reconcile.Result{}, nil
	}
}

// reconcileEN reconcile egressgateway
// goal:
// - update egresspolicy
func (r *egpReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log *zap.Logger) (reconcile.Result, error) {
	deleted := false
	egw := new(v1.EgressGateway)
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

	egpList := &v1.EgressPolicyList{}
	if err := r.client.List(ctx, egpList); err != nil {
		r.log.Sugar().Errorf("egp->controller, event: eg(%v): Failed to get EgressPolicyList\n", egw.Name)
		return reconcile.Result{Requeue: true}, err
	}

	for _, item := range egpList.Items {
		policy := v1.Policy{Name: item.Name, Namespace: item.Namespace}
		eipStatus, isExist := egressgateway.GetEIPStatusByPolicy(policy, *egw)
		if !isExist {
			continue
		}

		newEGP := item.DeepCopy()
		for _, eip := range eipStatus.Eips {
			for _, p := range eip.Policies {
				if p == policy {
					newEGP.Status.Eip.Ipv4 = eip.IPv4
					newEGP.Status.Eip.Ipv6 = eip.IPv6
					newEGP.Status.Node = eipStatus.Name
				}
			}
		}

		r.log.Sugar().Debugf("update egresspolicy status\n%v", newEGP.Status)
		err = r.client.Status().Update(ctx, newEGP)
		if err != nil {
			r.log.Sugar().Errorf("update egresspolicy status\n%v", newEGP.Status)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: false}, nil
}

func newEgressPolicyController(mgr manager.Manager, log *zap.Logger, cfg *config.Config) error {
	if log == nil {
		return fmt.Errorf("log can not be nil")
	}
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	r := &egpReconciler{
		client: mgr.GetClient(),
		log:    log,
		config: cfg,
	}

	log.Sugar().Infof("new egresspolicy controller")
	c, err := controller.New("egresspolicy", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &v1.EgressGateway{}},
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	return nil
}
