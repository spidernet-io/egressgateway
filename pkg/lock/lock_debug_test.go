// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

//go:build lockdebug

package lock

import "testing"

func TestLockDebugMutexOperations(t *testing.T) {
	var m Mutex
	m.Lock()
	m.Unlock()

	m.Lock()
	m.UnlockIgnoreTime()
}

func TestLockDebugRWMutexOperations(t *testing.T) {
	var rw RWMutex
	rw.Lock()
	rw.Unlock()

	rw.RLock()
	rw.RUnlock()

	rw.Lock()
	rw.UnlockIgnoreTime()
}
