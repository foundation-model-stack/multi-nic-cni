/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HostInterfaceHandler handles HostInterface object
// - general handling: Get, List, Delete
type HostInterfaceHandler struct {
	client.Client
	Log logr.Logger
}

// initHostInterface
func (h *HostInterfaceHandler) initHostInterface(hostName string, interfaces []multinicv1.InterfaceInfoType) *multinicv1.HostInterface {
	newHif := &multinicv1.HostInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostName,
		},
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   hostName,
			Interfaces: interfaces,
		},
	}
	return newHif
}

// CreateHostInterface creates new HostInterface from an interface list get from daemon pods
func (h *HostInterfaceHandler) CreateHostInterface(hostName string, interfaces []multinicv1.InterfaceInfoType) error {
	newHif := h.initHostInterface(hostName, interfaces)
	return h.Client.Create(context.TODO(), newHif)
}

// UpdateHostInterface updates HostInterface
func (h *HostInterfaceHandler) UpdateHostInterface(oldObj multinicv1.HostInterface, interfaces []multinicv1.InterfaceInfoType) (*multinicv1.HostInterface, error) {
	updateHif := &multinicv1.HostInterface{
		ObjectMeta: oldObj.ObjectMeta,
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   oldObj.Spec.HostName,
			Interfaces: interfaces,
		},
	}
	return updateHif, h.Client.Update(context.TODO(), updateHif)
}

// GetHostInterface gets HostInterface from hostname
func (h *HostInterfaceHandler) GetHostInterface(name string) (*multinicv1.HostInterface, error) {
	instance := &multinicv1.HostInterface{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, instance)
	return instance, err
}

// ListHostInterface returns a map from hostname to HostInterface
func (h *HostInterfaceHandler) ListHostInterface() (map[string]multinicv1.HostInterface, error) {
	hifList := &multinicv1.HostInterfaceList{}
	err := h.Client.List(context.TODO(), hifList)
	hifMap := make(map[string]multinicv1.HostInterface)
	if err == nil {
		for _, hif := range hifList.Items {
			name := hif.GetName()
			hifMap[name] = hif
		}
	}
	return hifMap, err
}

// DeleteHostInterface deletes HostInterface from hostname
func (h *HostInterfaceHandler) DeleteHostInterface(name string) error {
	instance, err := h.GetHostInterface(name)
	if err == nil {
		err = h.Client.Delete(context.TODO(), instance)
	}
	return err
}
