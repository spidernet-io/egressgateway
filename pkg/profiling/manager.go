// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package profiling

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/google/gops/agent"
	"github.com/pyroscope-io/client/pyroscope"
)

type GoPS struct {
	Port int
	Log  logr.Logger
}

func (gops *GoPS) Start(ctx context.Context) error {
	if gops.Port == 0 {
		return nil
	}
	address := fmt.Sprintf(":%d", gops.Port)
	op := agent.Options{
		ShutdownCleanup: true,
		Addr:            address,
	}
	if err := agent.Listen(op); err != nil {
		return fmt.Errorf("gops failed to listen on Port %s, reason: %v", address, err)
	}
	gops.Log.Info("gops is started", "addr", address)
	return nil
}

type Pyroscope struct {
	Addr     string
	HostName string
	Log      logr.Logger
}

func (p *Pyroscope) Start(ctx context.Context) error {
	if p.Addr == "" {
		return nil
	}

	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: filepath.Base(os.Args[0]),
		ServerAddress:   p.Addr,
		Logger:          nil,
		Tags:            map[string]string{"node": p.HostName},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to setup pyroscope: %v", err)
	}
	p.Log.Info("pyroscope started")
	return nil
}
