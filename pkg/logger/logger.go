// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
)

func NewStdoutLogger(loglevel string) *zap.Logger {

	// log level
	l := zapcore.InfoLevel
	switch strings.ToLower(loglevel) {
	case "debug":
		l = zapcore.DebugLevel
	case "info":
		l = zapcore.InfoLevel
	case "warn":
		l = zapcore.WarnLevel
	case "error":
		l = zapcore.ErrorLevel
	case "fatal":
		l = zapcore.FatalLevel
	case "panic":
		l = zapcore.PanicLevel
	case "":
		l = zapcore.InfoLevel
	default:
		panic(fmt.Sprintf("unknown log level %s", loglevel))
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = nil
	encoderConfig.EncodeCaller = nil
	encoderConfig.EncodeLevel = nil
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	m := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(l),
	)
	// configures the Logger to annotate each message with the filename, line number, and function name of zap's caller
	t := zap.AddCaller()
	logger := zap.New(m, t)

	return logger
}
