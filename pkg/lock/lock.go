// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium
// Based on: https://github.com/cilium/cilium/commit/32d2ef8e0445a79b1979312f33e0b514ca7650a5

package lock

// RWMutex is equivalent to sync.RWMutex but applies deadlock detection if the
// built tag "lockdebug" is set
type RWMutex struct {
	internalRWMutex
}

// Mutex is equivalent to sync.Mutex but applies deadlock detection if the
// built tag "lockdebug" is set
type Mutex struct {
	internalMutex
}
