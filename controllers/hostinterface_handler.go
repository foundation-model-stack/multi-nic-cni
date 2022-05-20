/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
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
func (h *HostInterfaceHandler) initHostInterface(hostName string, interfaces []netcogadvisoriov1.InterfaceInfoType) *netcogadvisoriov1.HostInterface {
	newHif := &netcogadvisoriov1.HostInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostName,
		},
		Spec: netcogadvisoriov1.HostInterfaceSpec{
			HostName:   hostName,
			Interfaces: interfaces,
		},
	}
	return newHif
}

// CreateHostInterface creates new HostInterface from an interface list get from daemon pods
func (h *HostInterfaceHandler) CreateHostInterface(hostName string, interfaces []netcogadvisoriov1.InterfaceInfoType) error {
	newHif := h.initHostInterface(hostName, interfaces)
	return h.Client.Create(context.TODO(), newHif)
}

// UpdateHostInterface updates HostInterface
func (h *HostInterfaceHandler) UpdateHostInterface(oldObj netcogadvisoriov1.HostInterface, interfaces []netcogadvisoriov1.InterfaceInfoType) error {
	updateHif := &netcogadvisoriov1.HostInterface{
		ObjectMeta: oldObj.ObjectMeta,
		Spec: netcogadvisoriov1.HostInterfaceSpec{
			HostName:   oldObj.Spec.HostName,
			Interfaces: interfaces,
		},
	}
	return h.Client.Update(context.TODO(), updateHif)
}

// GetHostInterface gets HostInterface from hostname
func (h *HostInterfaceHandler) GetHostInterface(name string) (*netcogadvisoriov1.HostInterface, error) {
	instance := &netcogadvisoriov1.HostInterface{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, instance)
	return instance, err
}

// ListHostInterface returns a map from hostname to HostInterface
func (h *HostInterfaceHandler) ListHostInterface() (map[string]netcogadvisoriov1.HostInterface, error) {
	hifList := &netcogadvisoriov1.HostInterfaceList{}
	err := h.Client.List(context.TODO(), hifList)
	hifMap := make(map[string]netcogadvisoriov1.HostInterface)
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
