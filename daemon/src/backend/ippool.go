/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package backend

import (
	"log"
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/types"
)

const (
	IPPOOL_RESOURCE = "ippools.v1.multinic.fms.io"
	IPPOOL_KIND     = "IPPool"
)

type IPPoolType struct {
	PodCIDR          string       `json:"podCIDR"`
	VlanCIDR         string       `json:"vlanCIDR"`
	NetAttachDefName string       `json:"netAttachDef"`
	HostName         string       `json:"hostName"`
	InterfaceName    string       `json:"interfaceName"`
	Excludes         []string     `json'"excludes"`
	Allocations      []Allocation `json:"allocations"`
}

type Allocation struct {
	Pod       string `json:"pod"`
	Namespace string `json:"namespace"`
	Index     int    `json:"index"`
	Address   string `json:"address"`
}

type IPPoolHandler struct {
	*DynamicHandler
}

func NewIPPoolHandler(config *rest.Config) *IPPoolHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &IPPoolHandler{
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: IPPOOL_RESOURCE,
			Kind:         IPPOOL_KIND,
		},
	}
	return handler
}

func (h *IPPoolHandler) parse(uobj unstructured.Unstructured) IPPoolType {
	var ippoolType IPPoolType
	spec := uobj.Object["spec"].(map[string]interface{})
	h.DynamicHandler.Parse(spec, &ippoolType)
	return ippoolType
}

func (h *IPPoolHandler) getIPPoolName(netAttachDef string, podCIDR string) string {
	return netAttachDef + "-" + strings.ReplaceAll(podCIDR, "/", "-")
}

func (h *IPPoolHandler) ListIPPool(listOptions metav1.ListOptions) (map[string]IPPoolType, error) {
	log.Println(fmt.Sprintf("ListIPPool with selector: %s", listOptions.LabelSelector))
	poolList, err := h.DynamicHandler.List(metav1.NamespaceAll, listOptions)
	poolSpecMap := make(map[string]IPPoolType)
	if err == nil {
		for _, pool := range poolList.Items {
			poolName := h.DynamicHandler.GetName(pool)
			poolInfo := h.parse(pool)
			poolSpecMap[poolName] = poolInfo
		}
	}
	return poolSpecMap, err
}

func (h *IPPoolHandler) PatchIPPool(poolname string, allocations []Allocation) (*unstructured.Unstructured, error) {
	allocationInByte, _ := json.Marshal(allocations)
	allocationReplace := fmt.Sprintf(`{"op":"replace","path":"/spec/allocations","value":%s}`, allocationInByte)
	dataStr := fmt.Sprintf(`[%s]`, allocationReplace)
	return h.DynamicHandler.Patch(poolname, metav1.NamespaceAll, types.JSONPatchType, []byte(dataStr), metav1.PatchOptions{})
}
