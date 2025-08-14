/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"errors"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"reflect"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	"k8s.io/apimachinery/pkg/types"

	"strconv"
)

// IPPoolHandler handles IPPool object
// - general handling: Get, List, Delete
// - update IPPool according to CIDR
type IPPoolHandler struct {
	client.Client
	*SafeCache
}

// GetIPPool gets IPPool from IPPool name
func (h *IPPoolHandler) GetIPPool(name string) (*multinicv1.IPPool, error) {
	ippool := &multinicv1.IPPool{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.Get(ctx, namespacedName, ippool)
	return ippool, err
}

// ListIPPool returns a map from IPPool name to IPPool
func (h *IPPoolHandler) ListIPPool() (map[string]multinicv1.IPPool, error) {
	poolList := &multinicv1.IPPoolList{}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.List(ctx, poolList)
	poolMap := make(map[string]multinicv1.IPPool)
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
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		err = h.Client.Delete(ctx, instance)
	}
	return err
}

// UpdateIPPool creates or updates IPPool from:
//   - network config: NetworkAttachmentDefinition name, excluded CIDR ranges
//   - VLAN CIDR: PodCIDR, vlanCIDR
//   - host-interface information: host name, interface name
//
// IPPool name is composed of NetworkAttachmentDefinition name and PodCIDR
func (h *IPPoolHandler) UpdateIPPool(netAttachDef string, podCIDR string, vlanCIDR string, hostName string, interfaceName string, excludes []compute.IPValue) error {
	labels := map[string]string{vars.HostNameLabel: hostName, vars.DefNameLabel: netAttachDef}
	ippoolName, spec, excludesInterface := h.initIPPool(netAttachDef, podCIDR, vlanCIDR, hostName, interfaceName, excludes)

	ippool, err := h.GetIPPool(ippoolName)
	if err == nil {
		// ippool exists, update
		prevSpec := ippool.Spec
		ippool.Spec = spec
		ippool.Spec.Allocations = prevSpec.Allocations
		ippool.ObjectMeta.Labels = labels
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		err = h.Client.Update(ctx, ippool)
		if !reflect.DeepEqual(prevSpec.Excludes, excludesInterface) {
			// report if allocated ip addresses have conflicts with the new IPPool (for example, in exclude list)
			invalidAllocations := h.checkPoolValidity(excludesInterface, prevSpec.Allocations)
			if len(invalidAllocations) > 0 {
				vars.IPPoolLog.V(5).Info(fmt.Sprintf("Update IPPool %s - conflict allocation: %v", ippoolName, invalidAllocations))
			}
			if prevSpec.HostName != hostName && len(prevSpec.Allocations) > 0 {
				vars.IPPoolLog.V(5).Info(fmt.Sprintf("Update IPPool %s - changes with exist %d allocations", ippoolName, len(prevSpec.Allocations)))
			}
		}
	} else {
		// create new ippool
		spec.Allocations = []multinicv1.Allocation{}
		newIPPool := &multinicv1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ippoolName,
				Namespace: metav1.NamespaceAll,
				Labels:    labels,
			},
			Spec: spec,
		}
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		err = h.Client.Create(ctx, newIPPool)
		vars.IPPoolLog.V(5).Info(fmt.Sprintf("New IPPool %s: %v, %v", ippoolName, newIPPool, err))
	}
	return err
}

// initIPPool creates IPPool name and spec from provided parameters.
func (h *IPPoolHandler) initIPPool(netAttachDef string, podCIDR string,
	vlanCIDR string, hostName string, interfaceName string, excludes []compute.IPValue) (string, multinicv1.IPPoolSpec, []string) {
	excludesInterface := h.extractMatchExcludesFromPodCIDR(excludes, podCIDR)

	spec := multinicv1.IPPoolSpec{
		PodCIDR:          podCIDR,
		VlanCIDR:         vlanCIDR,
		NetAttachDefName: netAttachDef,
		HostName:         hostName,
		InterfaceName:    interfaceName,
		Excludes:         excludesInterface,
	}

	ippoolName := h.GetIPPoolName(netAttachDef, podCIDR)

	return ippoolName, spec, excludesInterface
}

// extractMatchExcludesFromPodCIDR returns a list of exclude CIDR within given pod CIDR.
func (h *IPPoolHandler) extractMatchExcludesFromPodCIDR(excludes []compute.IPValue, podCIDR string) []string {
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
	return excludesInterface
}

// GetIPPoolName returns IPPool name = <NetworkAttachmentDefinition name> - <Pod CIDR IP> - <Pod CIDR block>
func (h *IPPoolHandler) GetIPPoolName(netAttachDef string, podCIDR string) string {
	return netAttachDef + "-" + strings.ReplaceAll(podCIDR, "/", "-")
}

// checkPoolValidity checks list of allocated IPs that is in exclude CIDRs
func (h *IPPoolHandler) checkPoolValidity(excludeCIDRs []string, allocations []multinicv1.Allocation) []multinicv1.Allocation {
	var invalidAllocations []multinicv1.Allocation
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

func (h *IPPoolHandler) PatchIPPoolAllocations(ippoolName string, newAllocations []multinicv1.Allocation) error {
	ippool, err := h.GetIPPool(ippoolName)
	if err != nil {
		return err
	}
	patch := client.MergeFrom(ippool.DeepCopy())
	ippool.Spec.Allocations = newAllocations
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	return h.Client.Patch(ctx, ippool, patch)
}

func (h *IPPoolHandler) UpdateIPPools(defName string, entries []multinicv1.CIDREntry, excludes []compute.IPValue) {
	for _, entry := range entries {
		for _, host := range entry.Hosts {
			err := h.UpdateIPPool(defName, host.PodCIDR, entry.VlanCIDR, host.HostName, host.InterfaceName, excludes)
			if err != nil {
				vars.IPPoolLog.V(5).Info(fmt.Sprintf("Cannot update IPPools for host %s: error=%v", host.HostName, err))
			}
		}
	}
}

func (h *IPPoolHandler) SetCache(key string, value multinicv1.IPPoolSpec) {
	h.SafeCache.SetCache(key, value)
}

func (h *IPPoolHandler) GetCache(key string) (multinicv1.IPPoolSpec, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return multinicv1.IPPoolSpec{}, errors.New(vars.NotFoundError)
	}
	return value.(multinicv1.IPPoolSpec), nil
}

func (h *IPPoolHandler) ListCache() map[string]multinicv1.IPPoolSpec {
	snapshot := make(map[string]multinicv1.IPPoolSpec)
	h.SafeCache.Lock()
	for key, value := range h.cache {
		snapshot[key] = value.(multinicv1.IPPoolSpec)
	}
	h.SafeCache.Unlock()
	return snapshot
}

func (h *IPPoolHandler) AddLabel(ippool *multinicv1.IPPool) error {
	hostName := ippool.Spec.HostName
	netAttachDef := ippool.Spec.NetAttachDefName
	labels := map[string]string{vars.HostNameLabel: hostName, vars.DefNameLabel: netAttachDef}
	patch := client.MergeFrom(ippool.DeepCopy())
	ippool.ObjectMeta.Labels = labels
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.Patch(ctx, ippool, patch)
	return err
}
