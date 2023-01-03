// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import "sync"

func NewSyncMap[K, V any]() *SyncMap[K, V] {
	s := new(SyncMap[K, V])
	s.m = new(sync.Map)
	return s
}

type SyncMap[K, V any] struct {
	m *sync.Map
}

func (s *SyncMap[K, V]) Load(key K) (V, bool) {
	v, ok := s.m.Load(key)
	if ok {
		return v.(V), ok
	}
	var empty V
	return empty, false
}

func (s *SyncMap[K, V]) Store(key K, val V) {
	s.m.Store(key, val)
}

func (s *SyncMap[K, V]) Range(f func(key K, val V) bool) {
	s.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V))
	})
}

func (s *SyncMap[K, V]) Delete(key K) {
	s.m.Delete(key)
}
