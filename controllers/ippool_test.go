/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var (
	defName       = "multi-nic-ipvlanl3"
	ipPrefixes    = []string{"192.168.0.", "192.168.64."}
	podCIDR       = []string{"192.168.0.0/26", "192.168.64.0/26"}
	vlanCIDR      = []string{"192.168.0.0/18", "192.168.64.0/18"}
	hostName      = "worker0"
	expectedIndex = 2

	newPodName     = "pod1"
	deletedPodName = "pod0"
	namespace      = "default"
)

func genIP(interfaceIndex, ipIndex int) string {
	return fmt.Sprintf("%s%d", ipPrefixes[interfaceIndex], ipIndex)
}

func genAllocations(ipIndexes map[int]int, podName string) []multinicv1.Allocation {
	allocations := []multinicv1.Allocation{}
	for index, ipIndex := range ipIndexes {
		ip := genIP(index, ipIndex)
		allocations = append(allocations, multinicv1.Allocation{
			Pod:       podName,
			Namespace: namespace,
			Index:     ipIndex,
			Address:   ip,
		})
	}
	return allocations
}

func genIPPool(interfaceIndex int, ipIndexes map[int]int, podName string) multinicv1.IPPoolSpec {
	return multinicv1.IPPoolSpec{
		PodCIDR:          podCIDR[interfaceIndex],
		VlanCIDR:         vlanCIDR[interfaceIndex],
		NetAttachDefName: defName,
		HostName:         hostName,
		InterfaceName:    interfaceNames[interfaceIndex],
		Allocations:      genAllocations(ipIndexes, podName),
	}
}

func genAllocationMap(ipIndexes map[int]int, podName string, clearIndex bool) map[string]map[string]multinicv1.Allocation {
	allocationMap := make(map[string]map[string]multinicv1.Allocation)
	allocationMap[defName] = make(map[string]multinicv1.Allocation)
	allocations := genAllocations(ipIndexes, podName)
	for _, allocation := range allocations {
		if clearIndex {
			allocation.Index = 0
		}
		allocationMap[defName][allocation.Address] = allocation
	}
	return allocationMap
}

func checkExpectedAllocation(interfaceIndex int, allocations []multinicv1.Allocation) {
	Expect(len(allocations)).To(Equal(1))
	allocation := allocations[0]
	address := genIP(interfaceIndex, expectedIndex)
	Expect(allocation.Address).To(Equal(address))
	Expect(allocation.Index).To(Equal(expectedIndex))
	Expect(allocation.Pod).To(Equal(newPodName))
	Expect(allocation.Namespace).To(Equal(namespace))
}

func checkSyncAllocation(allocationMap map[string]map[string]multinicv1.Allocation, crIndexes map[int]int, crPodName string, expectedChanged map[int]bool) {
	crAllocationMap := genAllocationMap(crIndexes, crPodName, false)
	ippools := []multinicv1.IPPoolSpec{}
	for interfaceIndex := range interfaceNames {
		ippool := genIPPool(interfaceIndex, map[int]int{interfaceIndex: crIndexes[interfaceIndex]}, crPodName)
		ippools = append(ippools, ippool)
	}
	checkSyncAllocationWithMap(allocationMap, crAllocationMap, ippools, expectedChanged)
}

func checkSyncAllocationWithMap(allocationMap, crAllocationMap map[string]map[string]multinicv1.Allocation, ippools []multinicv1.IPPoolSpec, expectedChanged map[int]bool) {
	for interfaceIndex := range interfaceNames {
		ippool := ippools[interfaceIndex]
		changed, newAllocations := MultiNicnetworkReconcilerInstance.CIDRHandler.GetSyncAllocations(ippool, allocationMap, crAllocationMap)
		// must be updated (deleted)
		Expect(len(allocationMap[defName])).To(Equal(len(interfaceNames) - interfaceIndex - 1))
		// must be changed
		expectedValue := expectedChanged[interfaceIndex]
		Expect(changed).To(Equal(expectedValue))
		checkExpectedAllocation(interfaceIndex, newAllocations)
	}
}

var _ = Describe("Unsync IPPool Test", func() {
	// current allocation map
	currentAllocations := map[int]int{0: 2, 1: 2}
	It("All new", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		emptyIndexes := map[int]int{}
		expectedChanged := map[int]bool{0: true, 1: true}
		checkSyncAllocation(allocationMap, emptyIndexes, deletedPodName, expectedChanged)
	})

	It("Deleted pods pending", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes := map[int]int{0: 1}
		expectedChanged := map[int]bool{0: true, 1: true}
		checkSyncAllocation(allocationMap, pendingIndexes, deletedPodName, expectedChanged)
		allocationMap = genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes = map[int]int{0: 1, 1: 1}
		checkSyncAllocation(allocationMap, pendingIndexes, deletedPodName, expectedChanged)
	})

	It("Deleted pods pending and new pod assigned the same index", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes := map[int]int{0: 2, 1: 2}
		expectedChanged := map[int]bool{0: true, 1: true}
		checkSyncAllocation(allocationMap, pendingIndexes, deletedPodName, expectedChanged)
	})

	It("Same pod different IP", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes := map[int]int{0: 1, 1: 1}
		expectedChanged := map[int]bool{0: true, 1: true}
		checkSyncAllocation(allocationMap, pendingIndexes, newPodName, expectedChanged)
	})

	It("Duplicated allocation", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		deltedIndexes := map[int]int{0: 1}
		expectedChanged := map[int]bool{0: true, 1: false}
		deletedCrAllocationMap := genAllocationMap(deltedIndexes, newPodName, false)
		newCrAlllocationMap := genAllocationMap(currentAllocations, newPodName, false)
		for ip, allocation := range deletedCrAllocationMap[defName] {
			newCrAlllocationMap[defName][ip] = allocation
		}
		ippools := []multinicv1.IPPoolSpec{}
		for interfaceIndex := range interfaceNames {
			newIPPool := genIPPool(interfaceIndex, map[int]int{interfaceIndex: currentAllocations[interfaceIndex]}, newPodName)
			if _, found := deltedIndexes[interfaceIndex]; found {
				deletedIppool := genIPPool(interfaceIndex, map[int]int{interfaceIndex: deltedIndexes[interfaceIndex]}, newPodName)
				newIPPool.Allocations = append(newIPPool.Allocations, deletedIppool.Allocations...)
			}
			ippools = append(ippools, newIPPool)
		}
		checkSyncAllocationWithMap(allocationMap, newCrAlllocationMap, ippools, expectedChanged)
	})

	It("Assignment on one interface is missing", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes := map[int]int{1: 2}
		expectedChanged := map[int]bool{0: true, 1: false}
		checkSyncAllocation(allocationMap, pendingIndexes, newPodName, expectedChanged)
	})

	It("Already synced", func() {
		allocationMap := genAllocationMap(currentAllocations, newPodName, true)
		pendingIndexes := map[int]int{0: 2, 1: 2}
		expectedChanged := map[int]bool{0: false, 1: false}
		checkSyncAllocation(allocationMap, pendingIndexes, newPodName, expectedChanged)
	})

	It("Should all clean", func() {
		emptyIndexes := map[int]int{}
		allocationMap := genAllocationMap(emptyIndexes, newPodName, true)
		pendingIndexes := map[int]int{0: 1, 1: 1}
		crAllocationMap := genAllocationMap(pendingIndexes, newPodName, false)
		ippools := []multinicv1.IPPoolSpec{}
		for interfaceIndex := range interfaceNames {
			ippool := genIPPool(interfaceIndex, pendingIndexes, newPodName)
			ippools = append(ippools, ippool)
		}
		for interfaceIndex := range interfaceNames {
			ippool := ippools[interfaceIndex]
			changed, newAllocations := MultiNicnetworkReconcilerInstance.CIDRHandler.GetSyncAllocations(ippool, allocationMap, crAllocationMap)
			// must be changed
			Expect(changed).To(Equal(true))
			Expect(len(newAllocations)).To(Equal(0))
		}
	})
})
