// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

//go:build !lockdebug

package lock

import "testing"

func TestLockFastMutexOperations(t *testing.T) {
	var m Mutex
	m.Lock()
	m.Unlock()

	m.Lock()
	m.UnlockIgnoreTime()

	m.Lock()
	m.Unlock()
}

func TestLockFastRWMutexOperations(t *testing.T) {
	var rw RWMutex
	rw.Lock()
	rw.Unlock()

	rw.RLock()
	rw.RUnlock()

	rw.Lock()
	rw.UnlockIgnoreTime()

	rw.Lock()
	rw.Unlock()
}
