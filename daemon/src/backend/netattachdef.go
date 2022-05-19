/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package backend

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"log"
)

const (
	NET_ATTACH_DEF_RESOURCE = "network-attachment-definitions.v1.k8s.cni.cncf.io"
	NET_ATTACH_DEF_KIND     = "NetworkAttachmentDefinition"
	RESOURCE_ANNOTATION     = "k8s.v1.cni.cncf.io/resourceName"
)



type NetAttachDefHandler struct {
	*DynamicHandler
}

func NewNetAttachDefHandler(config *rest.Config) *NetAttachDefHandler {
	dc, _ := discovery.NewDiscoveryClientForConfig(config)
	dyn, _ := dynamic.NewForConfig(config)

	handler := &NetAttachDefHandler{
		DynamicHandler: &DynamicHandler{
			DC:           dc,
			DYN:          dyn,
			ResourceName: NET_ATTACH_DEF_RESOURCE,
			Kind:         NET_ATTACH_DEF_KIND,
		},
	}
	return handler
}

func (h *NetAttachDefHandler) GetResourceName(name string, namespace string) string {
	netattachdef, err := h.DynamicHandler.Get(name, namespace, metav1.GetOptions{})
	if err != nil {
		log.Printf("Cannot get NetworkAttachDef %v \n", err)
		return ""
	}
	metadata := netattachdef.Object["metadata"].(map[string]interface{})
	if annotations, exist := metadata["annotations"]; exist {
		if resourceName, exist := annotations.(map[string]interface{})[RESOURCE_ANNOTATION]; exist {
			return resourceName.(string)
		}
	}
	log.Printf("No target annotation: %v\n", metadata)
	return ""
}

