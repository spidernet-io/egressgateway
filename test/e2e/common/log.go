// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"strings"
	"time"
)

type Logger struct {
	logs      []string
	startTime time.Time
	endTime   time.Time
}

func NewLogger() *Logger {
	return &Logger{
		startTime: time.Now(),
	}
}

func (l *Logger) Log(message string) {
	l.logs = append(l.logs, message)
}

func (l *Logger) Save() string {
	endTime := time.Now()
	duration := endTime.Sub(l.startTime)

	var sb strings.Builder

	// Append all log entries
	for _, log := range l.logs {
		sb.WriteString(log)
		sb.WriteString("\n")
	}

	// Append the total time spent in seconds
	sb.WriteString(fmt.Sprintf("Total duration: %.2f seconds\n", duration.Seconds()))

	return sb.String()
}
