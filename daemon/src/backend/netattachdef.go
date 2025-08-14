/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

func (h *NetAttachDefHandler) GetResourceNames(name string, namespace string) (resourceNames []string) {
	netattachdef, err := h.DynamicHandler.Get(name, namespace, metav1.GetOptions{})
	if err != nil {
		log.Printf("Cannot get NetworkAttachDef %v \n", err)
		return resourceNames
	}
	metadata := netattachdef.Object["metadata"].(map[string]interface{})
	if annotations, exist := metadata["annotations"]; exist {
		if combinedResourceName, exist := annotations.(map[string]interface{})[RESOURCE_ANNOTATION]; exist {
			resourceNames = strings.Split(combinedResourceName.(string), ",")
		}
	}
	log.Printf("Resource names: %v\n", resourceNames)
	return resourceNames
}
