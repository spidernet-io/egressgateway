// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/agent/vxlan"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
)

type egressNodeReconciler struct {
	client client.Client
	log    *zap.Logger
	//nolint
	vxlan vxlan.Interface
}

func (n egressNodeReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	n.log.Info("reconciling EgressNode")
	return reconcile.Result{}, nil
}

func newEgressNodeController(mgr manager.Manager, log *zap.Logger) error {
	r := &egressNodeReconciler{
		client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New("egressNode", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &egressv1.EgressNode{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch EgressNode: %w", err)
	}

	return nil
}
