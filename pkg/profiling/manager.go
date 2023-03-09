// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package profiling

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/gops/agent"
	"github.com/pyroscope-io/client/pyroscope"
	"go.uber.org/zap"
)

type Manager interface {
	RunGoPS(port int)
	RunPyroscope(serverAddress string, localHostName string)
}

type debugManager struct {
	logger *zap.Logger
}

var _ Manager = (*debugManager)(nil)

func (s *debugManager) RunGoPS(listerPort int) {
	address := fmt.Sprintf("127.0.0.1:%d", listerPort)
	op := agent.Options{
		ShutdownCleanup: true,
		Addr:            address,
	}
	if err := agent.Listen(op); err != nil {
		s.logger.Sugar().Fatalf("gops failed to listen on port %s, reason: %v", address, err)
	}
	s.logger.Sugar().Infof("gops is listening on %s ", address)
}

func (s *debugManager) RunPyroscope(serverAddress string, localHostName string) {
	// push mode, push to pyroscope server
	s.logger.Sugar().Infof("%v pyroscope works in push mode, server %s ", localHostName, serverAddress)

	_, e := pyroscope.Start(pyroscope.Config{
		ApplicationName: filepath.Base(os.Args[0]),
		ServerAddress:   serverAddress,
		// too much log
		// Logger:          pyroscope.StandardLogger,
		Logger: nil,
		Tags:   map[string]string{"node": localHostName},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	if e != nil {
		s.logger.Sugar().Fatalf("failed to setup pyroscope, reason: %v", e)
	}
}

func New(logger *zap.Logger) Manager {
	return &debugManager{
		logger: logger,
	}
}
