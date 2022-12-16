// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

type WrapperLogSink struct {
	level int
	log   *zap.Logger
}

func (w *WrapperLogSink) Init(info logr.RuntimeInfo) {
	w.log = w.log.WithOptions(zap.AddCallerSkip(info.CallDepth + 1))
}

func (w *WrapperLogSink) Enabled(level int) bool {
	return level <= w.level
}

func (w *WrapperLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	if level > w.level {
		return
	}
	log := w.withValues(keysAndValues)
	log.With(zap.Int("level", level)).Sugar().Infof(msg)
}

func (w *WrapperLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	log := w.withValues(keysAndValues)
	log.With(zap.Error(err)).Sugar().Errorf(msg, keysAndValues)
}

func (w *WrapperLogSink) withValues(keysAndValues ...interface{}) *zap.Logger {
	log := w.log
	if keysAndValues != nil {
		if len(keysAndValues)%2 == 0 {
			for i := 0; i < len(keysAndValues); i += 2 {
				key := keysAndValues[i]
				value := keysAndValues[i+1]
				log = log.With(zap.Any(fmt.Sprintf("%v", key), value))
			}
		}
	}
	return log
}

func (w *WrapperLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	log := w.withValues(keysAndValues)
	return &WrapperLogSink{w.level, log}
}

func (w *WrapperLogSink) WithName(name string) logr.LogSink {
	return &WrapperLogSink{w.level, w.log.Named(name)}
}

func NewLogSink(log *zap.Logger, level int) logr.LogSink {
	return &WrapperLogSink{log: log, level: level}
}
