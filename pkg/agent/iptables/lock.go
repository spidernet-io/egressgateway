// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// Copyright 2014 The Kubernetes Authors.

package iptables

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// nolint
type locker struct {
	lock16 *os.File
	lock14 *net.UnixListener
}

// nolint
func (l *locker) Close() error {
	var errList []error
	if l.lock16 != nil {
		if err := l.lock16.Close(); err != nil {
			errList = append(errList, err)
		}
	}
	if l.lock14 != nil {
		if err := l.lock14.Close(); err != nil {
			errList = append(errList, err)
		}
	}
	return utilerrors.NewAggregate(errList)
}

// nolint
func grabIptablesLocks(lockfilePath string) (iptablesLocker, error) {
	var err error
	var success bool

	l := &locker{}
	// nolint: errcheck
	defer func(l *locker) {
		// Clean up immediately on failure
		if !success {
			l.Close()
		}
	}(l)

	// Grab both 1.6.x and 1.4.x-style locks; we don't know what the
	// iptables-restore version is if it doesn't support --wait, so we
	// can't assume which lock method it'll use.

	// Roughly duplicate iptables 1.6.x xtables_lock() function.
	l.lock16, err = os.OpenFile(lockfilePath, os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open iptables lock %s: %v", lockfilePath, err)
	}

	if err := wait.PollImmediate(200*time.Millisecond, 2*time.Second, func() (bool, error) {
		if err := grabIptablesFileLock(l.lock16); err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to acquire new iptables lock: %v", err)
	}

	// Roughly duplicate iptables 1.4.x xtables_lock() function.
	if err := wait.PollImmediate(200*time.Millisecond, 2*time.Second, func() (bool, error) {
		l.lock14, err = net.ListenUnix("unix", &net.UnixAddr{Name: "@xtables", Net: "unix"})
		if err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to acquire old iptables lock: %v", err)
	}

	success = true
	return l, nil
}

// nolint
func grabIptablesFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

// nolint
type iptablesLocker interface {
	Close() error
}
