/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	"net"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"reflect"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/compute"
	"k8s.io/apimachinery/pkg/types"

	"strconv"
)

// IPPoolHandler handles IPPool object
// - general handling: Get, List, Delete
// - update IPPool according to CIDR
type IPPoolHandler struct {
	client.Client
	Log logr.Logger
}

// GetIPPool gets IPPool from IPPool name
func (h *IPPoolHandler) GetIPPool(name string) (*netcogadvisoriov1.IPPool, error) {
	ippool := &netcogadvisoriov1.IPPool{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, ippool)
	return ippool, err
}

// ListIPPool returns a map from IPPool name to IPPool
func (h *IPPoolHandler) ListIPPool() (map[string]netcogadvisoriov1.IPPool, error) {
	poolList := &netcogadvisoriov1.IPPoolList{}
	err := h.Client.List(context.TODO(), poolList)
	poolMap := make(map[string]netcogadvisoriov1.IPPool)
	if err == nil {
		for _, pool := range poolList.Items {
			poolName := pool.GetName()
			poolMap[poolName] = pool
		}
	}
	return poolMap, err
}

// DeleteIPPool deletes IPPool if exists
func (h *IPPoolHandler) DeleteIPPool(netAttachDef string, podCIDR string) error {
	name := h.GetIPPoolName(netAttachDef, podCIDR)
	instance, err := h.GetIPPool(name)
	if err == nil {
		err = h.Client.Delete(context.TODO(), instance)
	}
	return err
}

// UpdateIPPool creates or updates IPPool from:
//              - network config: NetworkAttachmentDefinition name, excluded CIDR ranges
//              - VLAN CIDR: PodCIDR, vlanCIDR
//              - host-interface information: host name, interface name
// IPPool name is composed of NetworkAttachmentDefinition name and PodCIDR
func (h *IPPoolHandler) UpdateIPPool(netAttachDef string, podCIDR string, vlanCIDR string, hostName string, interfaceName string, excludes []compute.IPValue) error {
	excludesInterface := []string{}

	// find CIDR ranges that excluded in the PodCIDR range
	for _, item := range excludes {
		excludeCIDR := item.Address
		_, podSubnet, _ := net.ParseCIDR(podCIDR)
		// split network address of excluded CIDR and CIDR block (bits)
		excludeIPSplits := strings.Split(excludeCIDR, "/")
		excludeIPStr := excludeIPSplits[0]
		// default CIDR block bits
		excludeBlock := int64(32)
		if len(excludeIPSplits) >= 2 {
			// update excludeBlock to defined CIDR block bits
			// convert block string to number
			excludeBlock, _ = strconv.ParseInt(excludeIPSplits[1], 10, 64)
		}
		// split network address of pod CIDR and CIDR block (bits)
		podBlockStr := strings.Split(podCIDR, "/")[1]
		// convert block string to number
		podBlock, _ := strconv.ParseInt(podBlockStr, 10, 64)
		if podBlock >= excludeBlock {
			// excludeBlock covers podBlock, should be handled by interface indexing step, continue
			continue
		}
		excludeIP, _, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", excludeIPStr, excludeBlock))
		if podSubnet.Contains(excludeIP) {
			// exclude CIDR is in pod CIDR, append to the exclude list of this pod CIDR
			excludesInterface = append(excludesInterface, excludeCIDR)
		}
	}

	// init spec
	spec := netcogadvisoriov1.IPPoolSpec{
		PodCIDR:          podCIDR,
		VlanCIDR:         vlanCIDR,
		NetAttachDefName: netAttachDef,
		HostName:         hostName,
		InterfaceName:    interfaceName,
		Excludes:         excludesInterface,
	}

	ippoolName := h.GetIPPoolName(netAttachDef, podCIDR)

	ippool, err := h.GetIPPool(ippoolName)
	if err == nil {
		// ippool exists, update
		prevSpec := ippool.Spec
		ippool.Spec = spec
		ippool.Spec.Allocations = prevSpec.Allocations
		err = h.Client.Update(context.TODO(), ippool)
		if !reflect.DeepEqual(prevSpec.Excludes, excludesInterface) {
			// report if allocated ip addresses have conflicts with the new IPPool (for example, in exclude list)
			invalidAllocations := h.checkPoolValidity(excludesInterface, prevSpec.Allocations)
			if len(invalidAllocations) > 0 {
				h.Log.Info(fmt.Sprintf("Update IPPool %s - conflict allocation: %v", ippoolName, invalidAllocations))
			}
			if prevSpec.HostName != hostName && len(prevSpec.Allocations) > 0 {
				h.Log.Info(fmt.Sprintf("Update IPPool %s - changes with exist %d allocations", ippoolName, len(prevSpec.Allocations)))
			}
		}
	} else {
		// create new ippool
		spec.Allocations = []netcogadvisoriov1.Allocation{}
		newIPPool := &netcogadvisoriov1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ippoolName,
				Namespace: metav1.NamespaceAll,
			},
			Spec: spec,
		}
		err = h.Client.Create(context.Background(), newIPPool)
		h.Log.Info(fmt.Sprintf("New IPPool %s: %v, %v", ippoolName, newIPPool, err))
	}
	return err
}

// GetIPPoolName returns IPPool name = <NetworkAttachmentDefinition name> - <Pod CIDR IP> - <Pod CIDR block>
func (h *IPPoolHandler) GetIPPoolName(netAttachDef string, podCIDR string) string {
	return netAttachDef + "-" + strings.ReplaceAll(podCIDR, "/", "-")
}

// checkPoolValidity checks list of allocated IPs that is in exclude CIDRs
func (h *IPPoolHandler) checkPoolValidity(excludeCIDRs []string, allocations []netcogadvisoriov1.Allocation) []netcogadvisoriov1.Allocation {
	var invalidAllocations []netcogadvisoriov1.Allocation
	if excludeCIDRs == nil || allocations == nil {
		// no exclude list or no allocation, return empty
		return invalidAllocations
	}
	for _, allocation := range allocations {
		for _, cidr := range excludeCIDRs {
			_, subnet, _ := net.ParseCIDR(cidr)
			ip, _, _ := net.ParseCIDR(allocation.Address + "/32")
			if subnet.Contains(ip) {
				// allocated IP in exclude list, append to invalid list
				invalidAllocations = append(invalidAllocations, allocation)
			}
		}
	}
	return invalidAllocations
}
