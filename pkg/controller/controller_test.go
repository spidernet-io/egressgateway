// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/spidernet-io/egressgateway/pkg/config"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := new(config.Config)
	cfg.MetricsBindAddress = "127.0.0.1"
	cfg.HealthProbeBindAddress = "127.0.0.1"
	_, _ = New(cfg)
}
