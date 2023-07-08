// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log"
	czap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Config struct {
	UseDevMode bool
	Level      zapcore.Level
	WithCaller bool
	Encoder    string
}

func NewLogger(cfg Config) logr.Logger {
	var opts []czap.Opts
	opts = append(opts,
		czap.UseDevMode(cfg.UseDevMode),
		czap.Level(cfg.Level),
		czap.RawZapOpts(zap.WithCaller(cfg.WithCaller)),
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
	logger := czap.New(opts...)
	log.SetLogger(logger)
	return logger
}
