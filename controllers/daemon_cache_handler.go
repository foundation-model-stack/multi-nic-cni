/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"errors"

	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

type DaemonPod struct {
	Name      string
	Namespace string
	HostIP    string
	NodeName  string
	Labels    map[string]string
}

type DaemonCacheHandler struct {
	*SafeCache
}

func (h *DaemonCacheHandler) SetCache(key string, value DaemonPod) {
	h.SafeCache.SetCache(key, value)
}

func (h *DaemonCacheHandler) GetCache(key string) (DaemonPod, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return DaemonPod{}, errors.New(vars.NotFoundError)
	}
	return value.(DaemonPod), nil
}

func (h *DaemonCacheHandler) ListCache() map[string]DaemonPod {
	snapshot := make(map[string]DaemonPod)
	h.SafeCache.Lock()
	for key, value := range h.cache {
		snapshot[key] = value.(DaemonPod)
	}
	h.SafeCache.Unlock()
	return snapshot
}
