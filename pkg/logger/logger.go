// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	czap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Config struct {
	UseDevMode bool          `mapstructure:"LOG_USE_DEV_MODE"`
	Level      zapcore.Level `mapstructure:"LOG_LEVEL"`
	WithCaller bool          `mapstructure:"LOG_WITH_CALLER"`
	Encoder    string        `mapstructure:"LOG_ENCODER"`
}

func NewLogger(cfg Config) logr.Logger {
	var opts []czap.Opts
	opts = append(opts,
		czap.UseDevMode(cfg.UseDevMode),
		czap.Level(cfg.Level),
		czap.RawZapOpts(zap.WithCaller(true)),
	)
	if cfg.Encoder == "console" {
		opts = append(opts, czap.ConsoleEncoder())
	} else {
		opts = append(opts, czap.JSONEncoder(
			func(config *zapcore.EncoderConfig) {
				config.EncodeTime = zapcore.ISO8601TimeEncoder
				config.EncodeDuration = zapcore.StringDurationEncoder
			}))
	}
	return czap.New(opts...)
}
