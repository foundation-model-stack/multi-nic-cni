/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package iface

import (
	"sync"
)

type SafeCache struct {
	mu    sync.RWMutex
	cache map[string]interface{}
}

func InitSafeCache() *SafeCache {
	return &SafeCache{
		cache: make(map[string]interface{}),
		mu:    sync.RWMutex{},
	}
}

func (s *SafeCache) SetCache(key string, value interface{}) {
	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()
}

func (s *SafeCache) UnsetCache(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}

func (s *SafeCache) Contains(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cache[key]
	return ok
}

func (s *SafeCache) GetCache(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if value, ok := s.cache[key]; ok {
		return value
	}
	return nil
}

func (s *SafeCache) Lock() {
	s.mu.RLock()
}

func (s *SafeCache) Unlock() {
	s.mu.RUnlock()
}

func (s *SafeCache) GetSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}
