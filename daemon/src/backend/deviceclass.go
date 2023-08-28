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
	DEVICECLASS_RESOURCE = "deviceclasses.v1.multinic.fms.io"
	DEVICECLASS_KIND     = "DeviceClass"
)

type DeviceID struct {
	Vendor   string   `json:"vendor"`
	Products []string `json:"products"`
}

type DeviceClassSpec struct {
	DeviceIDs []DeviceID `json:"ids"`
}

type DeviceClassHandler struct {
	*DynamicHandler
}

func NewDeviceClassHandler(config *rest.Config) *DeviceClassHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &DeviceClassHandler{
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: DEVICECLASS_RESOURCE,
			Kind:         DEVICECLASS_KIND,
		},
	}
	return handler
}

func (h *DeviceClassHandler) Get(name string) (DeviceClassSpec, error) {
	deviceclass, err := h.DynamicHandler.Get(name, metav1.NamespaceAll, metav1.GetOptions{})
	if err != nil {
		return DeviceClassSpec{DeviceIDs: []DeviceID{}}, err
	}
	spec := DeviceClassSpec{}
	jsonBytes, err := json.Marshal(deviceclass.Object["spec"])
	if err != nil {
		return DeviceClassSpec{DeviceIDs: []DeviceID{}}, err
	}
	err = json.Unmarshal(jsonBytes, &spec)
	return spec, nil
}
