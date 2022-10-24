// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spidernet-io/egressgateway/pkg/debug"
	"time"
)

func SetupUtility() {
	// run gops
	d := debug.New(rootLogger)
	if globalConfig.GopsPort != 0 {
		d.RunGops(int(globalConfig.GopsPort))
	}

	if globalConfig.PyroscopeServerAddress != "" {
		d.RunPyroscope(globalConfig.PyroscopeServerAddress, globalConfig.PodName)
	}
}

func DaemonMain() {
	rootLogger.Sugar().Infof("config: %+v", globalConfig)

	SetupUtility()

	SetupHttpServer()

	RunMetricsServer(globalConfig.PodName)

	rootLogger.Info("hello world")
	time.Sleep(time.Hour)
}
