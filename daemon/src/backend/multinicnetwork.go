/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	MULTINICNET_RESOURCE = "multinicnetworks.v1.multinic.fms.io"
	MULTINICNET_KIND     = "MultiNicNetwork"
)

type MultiNicNetworkSpec struct {
	Policy         AttachmentPolicy `json:"attachPolicy,omitempty"`
	MasterNetAddrs []string         `json:"masterNets,omitempty"`
}

type AttachmentPolicy struct {
	Strategy string `json:"strategy"`
	Target   string `json:"target,omitempty"`
}

type MultiNicNetworkHandler struct {
	*DynamicHandler
}

func NewMultiNicNetworkHandler(config *rest.Config) *MultiNicNetworkHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &MultiNicNetworkHandler{
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: MULTINICNET_RESOURCE,
			Kind:         MULTINICNET_KIND,
		},
	}
	return handler
}

func (h *MultiNicNetworkHandler) Get(name string) (MultiNicNetworkSpec, error) {
	multinicnetwork, err := h.DynamicHandler.Get(name, metav1.NamespaceAll, metav1.GetOptions{})
	if err != nil {
		return MultiNicNetworkSpec{Policy: AttachmentPolicy{Strategy: "none"}}, err
	}
	spec := MultiNicNetworkSpec{}
	jsonBytes, err := json.Marshal(multinicnetwork.Object["spec"])
	if err != nil {
		return MultiNicNetworkSpec{Policy: AttachmentPolicy{Strategy: "none"}}, err
	}
	err = json.Unmarshal(jsonBytes, &spec)
	return spec, nil
}
