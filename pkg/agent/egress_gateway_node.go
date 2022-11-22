// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/agent/iptables"
	"github.com/spidernet-io/egressgateway/pkg/agent/route"
	v1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
)

type egressGatewayNodeReconciler struct {
	client client.Client
	log    *zap.Logger
	//nolint
	iptables iptables.Interface
	//nolint
	route route.Interface
}

func (n egressGatewayNodeReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	n.log.Info("reconciling EgressGatewayNode")
	return reconcile.Result{}, nil
}

func newEgressGatewayNodeController(mgr manager.Manager, log *zap.Logger) error {
	r := &egressGatewayNodeReconciler{
		client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New("egressGatewayNode", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &v1.EgressGatewayNode{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch EgressGatewayNode: %w", err)
	}

	return nil
}
