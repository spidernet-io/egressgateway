// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	runtimeWebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/controller/egress_cluster_info"
	"github.com/spidernet-io/egressgateway/pkg/controller/metrics"
	"github.com/spidernet-io/egressgateway/pkg/controller/webhook"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/profiling"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/types"
)

type Controller struct {
	client  client.Client
	manager manager.Manager
}

func New(cfg *config.Config) (types.Service, error) {
	log := logger.NewLogger(cfg.EnvConfig.Logger)
	mgrOpts := manager.Options{
		Scheme:                  schema.GetScheme(),
		Logger:                  log,
		LeaderElection:          cfg.LeaderElection,
		HealthProbeBindAddress:  cfg.HealthProbeBindAddress,
		LeaderElectionID:        cfg.LeaderElectionID,
		LeaderElectionNamespace: cfg.LeaderElectionNamespace,
		WebhookServer: runtimeWebhook.NewServer(runtimeWebhook.Options{
			Port:    cfg.WebhookPort,
			CertDir: cfg.TLSCertDir,
		}),
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
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to AddHealthzCheck: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("failed to AddReadyzCheck: %w", err)
	}
	mgr.GetWebhookServer().Register("/validate", webhook.ValidateHook(mgr.GetClient(), cfg))
	mgr.GetWebhookServer().Register("/mutate", webhook.MutateHook(mgr.GetClient(), cfg))

	err = mgr.Add(&profiling.GoPS{Port: cfg.GopsPort, Log: log})
	if err != nil {
		return nil, err
	}
	err = mgr.Add(&profiling.Pyroscope{Addr: cfg.PyroscopeServerAddr, Name: cfg.PodName, Log: log})
	if err != nil {
		return nil, err
	}

	metrics.RegisterMetricCollectors()

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
	err = egressclusterinfo.NewEgressClusterInfoController(mgr, log)
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

	return &Controller{client: mgr.GetClient(), manager: mgr}, err
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
