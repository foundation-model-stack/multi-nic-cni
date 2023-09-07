/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"log"
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

const (
	HIF_RESOURCE = "hostinterfaces.v1.multinic.fms.io"
	HIF_KIND     = "HostInterface"

	UnmanagedLabelName = "multi-nic-unmanaged"
)

type InterfaceInfoType struct {
	InterfaceName string `json:"interfaceName"`
	NetAddress    string `json:"netAddress"`
	HostIP        string `json:"hostIP"`
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	PciAddress    string `json:"pciAddress"`
}

type HostInterfaceSpec struct {
	HostName   string              `json:"hostName"`
	Interfaces []InterfaceInfoType `json:"interfaces"`
}

type HostInterfaceHandler struct {
	*DynamicHandler
	HostName string
}

func NewHostInterfaceHandler(config *rest.Config, hostName string) *HostInterfaceHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &HostInterfaceHandler{
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: HIF_RESOURCE,
			Kind:         HIF_KIND,
		},
		HostName: hostName,
	}
	return handler
}

func (h *HostInterfaceHandler) IsUnmanaged(uobj unstructured.Unstructured) bool {
	var metadata metav1.ObjectMeta
	h.DynamicHandler.Parse(uobj.Object["metadata"].(map[string]interface{}), &metadata)
	labels := metadata.Labels
	unmanaged, found := labels[UnmanagedLabelName]
	return found && (strings.ToLower(unmanaged) == "true")
}

func (h *HostInterfaceHandler) parse(uobj unstructured.Unstructured) HostInterfaceSpec {
	var hifSpec HostInterfaceSpec
	spec := uobj.Object["spec"].(map[string]interface{})
	h.DynamicHandler.Parse(spec, &hifSpec)
	return hifSpec
}

func (h *HostInterfaceHandler) GetInterfaces() ([]InterfaceInfoType, error) {
	unstructuredHif, err := h.DynamicHandler.Get(h.HostName, metav1.NamespaceAll, metav1.GetOptions{})
	if err != nil {
		log.Printf("Cannot get HostInterface %s", h.HostName)
		return []InterfaceInfoType{}, err
	}
	if h.IsUnmanaged(*unstructuredHif) {
		hif := h.parse(*unstructuredHif)
		log.Printf("Get unmanaged HostInterface %s: %v", h.HostName, hif.Interfaces)
		return hif.Interfaces, nil
	}
	// if not unmanaged, ignore the list
	return []InterfaceInfoType{}, nil
}
