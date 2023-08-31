// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/spidernet-io/egressgateway/pkg/agent/metrics"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/profiling"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/types"
)

type Agent struct {
	client  client.Client
	manager manager.Manager
}

func New(cfg *config.Config) (types.Service, error) {
	syncPeriod := time.Second * 15
	log := logger.NewLogger(cfg.EnvConfig.Logger)
	mgrOpts := manager.Options{
		Scheme:                 schema.GetScheme(),
		Logger:                 log,
		HealthProbeBindAddress: cfg.HealthProbeBindAddress,
		SyncPeriod:             &syncPeriod,
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

	err = mgr.Add(&profiling.GoPS{Port: cfg.GopsPort, Log: log})
	if err != nil {
		return nil, err
	}
	err = mgr.Add(&profiling.Pyroscope{Addr: cfg.PyroscopeServerAddr, Name: cfg.PodName, Log: log})
	if err != nil {
		return nil, err
	}

	metrics.RegisterMetricCollectors()

	err = newEgressTunnelController(mgr, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create node controller: %w", err)
	}

	err = newPolicyController(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create egress gateway policy controller: %w", err)
	}

	err = newEipCtrl(mgr, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to eip controller: %w", err)
	}

	return &Agent{client: mgr.GetClient(), manager: mgr}, err
}

func (c *Agent) Start(ctx context.Context) error {
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
