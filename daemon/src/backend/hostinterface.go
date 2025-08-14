/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

type InterfaceInfoType struct {
	InterfaceName string `json:"interfaceName"`
	NetAddress    string `json:"netAddress"`
	HostIP        string `json:"hostIP"`
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	PciAddress    string `json:"pciAddress"`
}

const (
	HOSTINTERFACE_RESOURCE = "hostinterfaces.v1.multinic.fms.io"
	HOSTINTERFACE_KIND     = "hostinterfaces"
)

var (
	unmanagedLabelName = "multi-nic-unmanaged"
)

type HostInterfaceHandler struct {
	*DynamicHandler
	hostName string
}

func NewHostInterfaceHandler(config *rest.Config, hostName string) *HostInterfaceHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &HostInterfaceHandler{
		hostName: hostName,
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: HOSTINTERFACE_RESOURCE,
			Kind:         HOSTINTERFACE_KIND,
		},
	}
	return handler
}

func (h *HostInterfaceHandler) parse(uobj *unstructured.Unstructured) ([]InterfaceInfoType, error) {
	var infos []InterfaceInfoType
	var err error
	spec := uobj.Object["spec"].(map[string]interface{})
	if v, found := spec["interfaces"]; found {
		if vals, ok := v.([]interface{}); ok {
			for _, v := range vals {
				var info InterfaceInfoType
				h.DynamicHandler.Parse(v.(map[string]interface{}), &info)
				infos = append(infos, info)
			}
		} else {
			err = fmt.Errorf("cannot parse value of interfaces")
		}
	} else {
		err = fmt.Errorf("`interfaces` field not found")
	}
	return infos, err
}

func (h *HostInterfaceHandler) GetHostInterfaces() ([]InterfaceInfoType, error) {
	hifobj, err := h.DynamicHandler.Get(h.hostName, metav1.NamespaceAll, metav1.GetOptions{})
	if err == nil {
		infos, err := h.parse(hifobj)
		return infos, err
	}
	return []InterfaceInfoType{}, err
}

func (h *HostInterfaceHandler) GetUnmanagedHostInterfaces() ([]InterfaceInfoType, error) {
	hifobj, err := h.DynamicHandler.Get(h.hostName, metav1.NamespaceAll, metav1.GetOptions{})
	if err == nil {
		if labels := hifobj.GetLabels(); labels != nil {
			if labels[unmanagedLabelName] == "true" {
				infos, err := h.parse(hifobj)
				return infos, err
			}
		}
	}
	return []InterfaceInfoType{}, err
}
