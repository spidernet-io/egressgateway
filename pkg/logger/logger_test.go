// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package logger_test

import (
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"testing"
)

func TestNewLogger(t *testing.T) {

	t.Run("Encoder", func(t *testing.T) {
		cfg := logger.Config{
			Encoder: "Encoder",
		}
		logger.NewLogger(cfg)
	})

	t.Run("not Encoder", func(t *testing.T) {
		cfg := logger.Config{}
		logger.NewLogger(cfg)
	})

	t.Run("console log", func(t *testing.T) {
		cfg := logger.Config{
			Encoder: "console",
		}
		logger.NewLogger(cfg)
	})
}
