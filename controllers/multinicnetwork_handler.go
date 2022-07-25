/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// MultiNicNetworkHandler handles MultiNicNetwork object
// - update MultiNicNetwork status according to CIDR results
type MultiNicNetworkHandler struct {
	client.Client
	Log logr.Logger
}

func (h *MultiNicNetworkHandler) GetNetwork(name string) (*netcogadvisoriov1.MultiNicNetwork, error) {
	instance := &netcogadvisoriov1.MultiNicNetwork{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, instance)
	return instance, err
}

func (h *MultiNicNetworkHandler) SyncStatus(name string, spec netcogadvisoriov1.CIDRSpec, status netcogadvisoriov1.RouteStatus) error {
	instance, err := h.GetNetwork(name)
	if err != nil {
		return err
	}

	var results []netcogadvisoriov1.NicNetworkResult
	for _, entry := range spec.CIDRs {
		result := netcogadvisoriov1.NicNetworkResult{
			NetAddress: entry.NetAddress,
			NumOfHost:  len(entry.Hosts),
		}
		results = append(results, result)
	}

	instance.Status = netcogadvisoriov1.MultiNicNetworkStatus{
		ComputeResults: results,
		LastSyncTime:   metav1.Now(),
		Status:         status,
	}
	err = h.Client.Status().Update(context.Background(), instance)
	return err

}

func (h *MultiNicNetworkHandler) UpdateStatus(cidr netcogadvisoriov1.CIDR, status netcogadvisoriov1.RouteStatus) error {
	return h.SyncStatus(cidr.GetName(), cidr.Spec, status)
}
