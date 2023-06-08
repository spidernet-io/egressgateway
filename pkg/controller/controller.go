// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/controller/metrics"
	"github.com/spidernet-io/egressgateway/pkg/controller/webhook"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/types"
)

type Controller struct {
	client  client.Client
	manager manager.Manager
}

func New(cfg *config.Config, log *zap.Logger) (types.Service, error) {
	mgrOpts := manager.Options{
		Scheme:                  schema.GetScheme(),
		Logger:                  logr.New(logger.NewLogSink(log, cfg.KLOGLevel)),
		LeaderElection:          false,
		HealthProbeBindAddress:  cfg.HealthProbeBindAddress,
		LeaderElectionID:        cfg.LeaderElectionID,
		LeaderElectionNamespace: cfg.LeaderElectionNamespace,
	}

	if cfg.MetricsBindAddress != "" {
		mgrOpts.MetricsBindAddress = cfg.MetricsBindAddress
	}

	if cfg.HealthProbeBindAddress != "" {
		mgrOpts.HealthProbeBindAddress = cfg.HealthProbeBindAddress
	}

	mgr, err := ctrl.NewManager(cfg.KubeConfig, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	err = mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		return nil, fmt.Errorf("failed to AddHealthzCheck: %w", err)
	}
	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		return nil, fmt.Errorf("failed to AddReadyzCheck: %w", err)
	}

	metrics.RegisterMetricCollectors()

	mgr.GetWebhookServer().Port = cfg.WebhookPort
	mgr.GetWebhookServer().CertDir = cfg.TLSCertDir
	mgr.GetWebhookServer().Register("/validate", webhook.ValidateHook(mgr.GetClient(), cfg))
	mgr.GetWebhookServer().Register("/mutate", webhook.MutateHook(mgr.GetClient(), cfg))

	err = egressgateway.NewEgressGatewayController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress gateway controller: %w", err)
	}

	err = newEgressPolicyController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress policy controller: %w", err)
	}

	err = newEgressClusterPolicyController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress cluster policy controller: %w", err)
	}

	err = newEgressNodeController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress node controller: %w", err)
	}
	err = newEgressClusterInfoController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress cluster info controller: %w", err)
	}

	err = newEgressEndpointSliceController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint slice controller: %w", err)
	}

	err = newEgressClusterEpSliceController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster endpoint slice controller: %w", err)
	}

	return &Controller{
		client:  mgr.GetClient(),
		manager: mgr,
	}, err
}

func (c *Controller) Start(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		errChan <- c.manager.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}
