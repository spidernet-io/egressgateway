// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/spidernet-io/egressgateway/pkg/controller/allocator"
)

type nodeReconciler struct {
	client client.Client
	log    *zap.Logger
	//nolint
	v4Allocator allocator.Interface
	//nolint
	v6Allocator allocator.Interface
	//nolint
	enableIPv4 bool
	//nolint
	enableIPv6 bool
}

func (r nodeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	r.log.Info("reconciling node")
	return reconcile.Result{}, nil
}

func newNodeController(mgr manager.Manager, log *zap.Logger) error {
	r := &nodeReconciler{
		client: mgr.GetClient(),
		log:    log,
	}

	c, err := controller.New("node", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Node: %w", err)
	}

	return nil
}
