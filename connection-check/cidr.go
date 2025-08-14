/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	CIDR_RESOURCE = "cidrs.v1.multinic.fms.io"
)

type HostInterfaceInfo struct {
	HostIndex     int    `json:"hostIndex"`
	HostName      string `json:"hostName"`
	InterfaceName string `json:"interfaceName"`
	HostIP        string `json:"hostIP"`
	PodCIDR       string `json:"podCIDR"`
}

type CIDREntry struct {
	NetAddress     string              `json:"netAddress"`
	InterfaceIndex int                 `json:"interfaceIndex"`
	VlanCIDR       string              `json:"vlanCIDR"`
	Hosts          []HostInterfaceInfo `json:"hosts"`
}

// CIDRSpec defines the desired state of CIDR
type CIDRSpec struct {
	CIDRs []CIDREntry `json:"cidr"`
}

type CIDRHandler struct {
	DC           *discovery.DiscoveryClient
	DYN          dynamic.Interface
	ResourceName string
	Kind         string
}

func NewCIDRHandler(config *rest.Config) *CIDRHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &CIDRHandler{
		DC:           dc,
		DYN:          dyn,
		ResourceName: CIDR_RESOURCE,
	}
	return handler
}

func (h *CIDRHandler) parse(cidr unstructured.Unstructured) (CIDRSpec, error) {
	var cidrSpec CIDRSpec
	spec := cidr.Object["spec"].(map[string]interface{})
	jsonBytes, err := json.Marshal(spec)
	if err != nil {
		return cidrSpec, err
	}
	err = json.Unmarshal(jsonBytes, &cidrSpec)
	return cidrSpec, err
}

func (h *CIDRHandler) getName(cidr unstructured.Unstructured) string {
	return cidr.Object["metadata"].(map[string]interface{})["name"].(string)
}

func (h *CIDRHandler) List(options metav1.ListOptions) map[string]CIDRSpec {
	cidrSpecMap := make(map[string]CIDRSpec)
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cidrList, err := h.DYN.Resource(*gvr).Namespace(metav1.NamespaceAll).List(ctx, options)
	if err != nil {
		log.Println(fmt.Sprintf("Cannot list CIDR: %v", err))
	} else {
		for _, cidr := range cidrList.Items {
			name := h.getName(cidr)
			cidrSpec, err := h.parse(cidr)
			if err != nil {
				log.Println(fmt.Sprintf("Cannot parse CIDR %s: %v", name, err))
			} else {
				cidrSpecMap[name] = cidrSpec
			}
		}
	}
	return cidrSpecMap
}

func (h *CIDRHandler) GetPodCIDRsMap(spec CIDRSpec) map[string][]string {
	entries := spec.CIDRs
	podCIDRsMap := make(map[string][]string)
	for _, entry := range entries {
		for _, host := range entry.Hosts {
			if _, exist := podCIDRsMap[host.HostName]; exist {
				podCIDRsMap[host.HostName] = append(podCIDRsMap[host.HostName], host.PodCIDR)
			} else {
				podCIDRsMap[host.HostName] = []string{host.PodCIDR}
			}
		}
	}
	return podCIDRsMap
}
