// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type egcpReconciler struct {
	client client.Client
	log    logr.Logger
	config *config.Config
}

func (r *egcpReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	kind, newReq, err := utils.ParseKindWithReq(req)
	if err != nil {
		return reconcile.Result{}, err
	}

	log := r.log.WithValues("name", newReq.Name, "kind", kind)
	log.Info("reconciling")
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
func (r *egcpReconciler) reconcileEG(ctx context.Context, req reconcile.Request, log logr.Logger) (reconcile.Result, error) {
	deleted := false
	egw := new(v1beta1.EgressGateway)
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

	egcpList := &v1beta1.EgressClusterPolicyList{}
	if err := r.client.List(ctx, egcpList); err != nil {
		log.Error(err, "failed to list")
		return reconcile.Result{Requeue: true}, err
	}

	for _, item := range egcpList.Items {
		policy := v1beta1.Policy{Name: item.Name, Namespace: item.Namespace}
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

		log.V(1).Info("update status", "status", newEGCP.Status)
		err = r.client.Status().Update(ctx, newEGCP)
		if err != nil {
			log.Error(err, "update status", "status", newEGCP.Status)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: false}, nil
}

func newEgressClusterPolicyController(mgr manager.Manager, log logr.Logger, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("cfg can not be nil")
	}

	log.Info("new egressclusterpolicy controller")

	r := &egcpReconciler{client: mgr.GetClient(), log: log, config: cfg}
	c, err := controller.New("egressclusterpolicy", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &v1beta1.EgressGateway{}),
		handler.EnqueueRequestsFromMapFunc(utils.KindToMapFlat("EgressGateway"))); err != nil {
		return fmt.Errorf("failed to watch EgressGateway: %w", err)
	}

	return nil
}
