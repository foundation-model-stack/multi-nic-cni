/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

const (
	HOSTINTERFACE_RESOURCE = "hostinterfaces.v1.multinic.fms.io"
	HOSTINTERFACE_KIND     = "hostinterfaces"
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

func (h *HostInterfaceHandler) parse(uobj *unstructured.Unstructured) ([]iface.InterfaceInfoType, error) {
	var infos []iface.InterfaceInfoType
	var err error
	spec := uobj.Object["spec"].(map[string]interface{})
	if v, found := spec["interfaces"]; found {
		if vals, ok := v.([]interface{}); ok {
			for _, v := range vals {
				var info iface.InterfaceInfoType
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

func (h *HostInterfaceHandler) GetHostInterfaces() ([]iface.InterfaceInfoType, error) {
	hifobj, err := h.DynamicHandler.Get(h.hostName, metav1.NamespaceAll, metav1.GetOptions{})
	if err == nil {
		infos, err := h.parse(hifobj)
		return infos, err
	}
	return []iface.InterfaceInfoType{}, err
}
