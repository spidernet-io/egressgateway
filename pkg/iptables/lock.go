// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
//
// Copyright (c) 2017-2022 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is based on that extracted from Kubernetes at pkg/util/iptables/iptables_linux.go.

package iptables

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	wlock "github.com/spidernet-io/egressgateway/pkg/lock"
	"golang.org/x/sys/unix"
)

var (
	summaryLockAcquisitionTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "iptables_lock_acquire_secs",
		Help: "Time in seconds that it took to acquire the iptables lock(s).",
	})
	countLockRetries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "iptables_lock_retries",
		Help: "Number of times the iptables lock was held by someone else and we had to retry.",
	}, []string{"version"})

	countLockRetriesV14 = countLockRetries.WithLabelValues("1.4")
	countLockRetriesV16 = countLockRetries.WithLabelValues("1.6")
)

func NewSharedLock(lockFilePath string, lockTimeout, lockProbeInterval time.Duration) *SharedLock {
	return &SharedLock{
		lockFilePath:      lockFilePath,
		lockTimeout:       lockTimeout,
		lockProbeInterval: lockProbeInterval,
		GrabIptablesLocks: GrabIptablesLocks,
	}
}

// SharedLock allows multiple goroutines to share the iptables lock without blocking each other.
// This is safe because each of our goroutines is accessing a different iptables table, so they
// don't conflict with each other.
type SharedLock struct {
	lock              wlock.Mutex
	referenceCount    int
	lockHandle        io.Closer
	lockFilePath      string
	lockTimeout       time.Duration
	lockProbeInterval time.Duration
	GrabIptablesLocks func(lockFilePath, socketName string, timeout, probeInterval time.Duration) (io.Closer, error)
}

func (l *SharedLock) Lock() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.referenceCount == 0 {
		// The lock isn't currently held. Acquire it.
		lockHandle, err := l.GrabIptablesLocks(
			l.lockFilePath,
			"@xtables",
			l.lockTimeout,
			l.lockProbeInterval,
		)
		if err != nil {
			// we give the lock plenty of time so err on the side of assuming a programming bug.
			panic(fmt.Sprintf("failed to acquire iptables lock: %v", err))
		}
		l.lockHandle = lockHandle
	}
	l.referenceCount++
}

func (l *SharedLock) Unlock() {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.referenceCount--
	if l.referenceCount < 0 {
		panic("Unmatched Unlock()")
	}
	if l.referenceCount == 0 {
		// log.Debug("Releasing iptables lock.")
		err := l.lockHandle.Close()
		if err != nil {
			// We haven't done anything with the file or socket, so we shouldn't be
			// able to hit any "deferred flush" type errors from the close.  Panic
			// since we're not sure what's going on.
			panic(fmt.Sprintf("error while closing iptables lock: %v", err))
		}
		l.lockHandle = nil
	}
}

type Locker struct {
	Lock16 io.Closer
	Lock14 io.Closer
}

func (l *Locker) Close() error {
	var err16 error
	if l.Lock16 != nil {
		err16 = l.Lock16.Close()
	}
	if l.Lock14 != nil {
		err14 := l.Lock14.Close()
		if err14 != nil {
			if err16 == nil {
				return err14
			}
			return fmt.Errorf("lock16 error: %v; lock14 error: %v", err14, err16)
		}
	}
	return nil
}

var (
	Err14LockTimeout = errors.New("timed out waiting for iptables 1.4 lock")
	Err16LockTimeout = errors.New("timed out waiting for iptables 1.6 lock")
)

func GrabIptablesLocks(lockFilePath, socketName string,
	timeout, probeInterval time.Duration) (io.Closer, error) {

	var err error
	var success bool

	l := &Locker{}
	defer func(l *Locker) {
		// clean up immediately on failure
		if !success {
			l.Close()
		}
	}(l)

	// Grab both 1.6.x and 1.4.x-style locks; we don't know what the iptables-restore version
	// is if it doesn't support --wait, so we can't assume which lock method it'll use.

	// Roughly duplicate iptables 1.6.x xtables_lock() function.
	f, err := os.OpenFile(lockFilePath, os.O_CREATE, 0600)
	l.Lock16 = f
	if err != nil {
		return nil, fmt.Errorf("failed to open iptables lock %s: %v", lockFilePath, err)
	}

	startTime := time.Now()
	for {
		if err := grabIptablesFileLock(f); err == nil {
			break
		}
		if time.Since(startTime) > timeout {
			return nil, Err16LockTimeout
		}
		time.Sleep(probeInterval)
		countLockRetriesV16.Inc()
	}

	startTime14 := time.Now()
	for {
		l.Lock14, err = net.ListenUnix("unix", &net.UnixAddr{Name: socketName, Net: "unix"})
		if err == nil {
			break
		}
		if time.Since(startTime14) > timeout {
			return nil, Err14LockTimeout
		}
		time.Sleep(probeInterval)
		countLockRetriesV14.Inc()
	}

	summaryLockAcquisitionTime.Observe(time.Since(startTime).Seconds())

	success = true
	return l, nil
}

func grabIptablesFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

type DummyLock struct{}

func (d DummyLock) Lock() {
}

func (d DummyLock) Unlock() {
}
