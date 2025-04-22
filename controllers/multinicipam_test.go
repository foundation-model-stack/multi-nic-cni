/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

func updateCIDR(multinicnetwork *multinicv1.MultiNicNetwork, cidr multinicv1.CIDRSpec, new, expectChange bool) multinicv1.CIDRSpec {
	def := cidr.Config
	excludes := compute.SortAddress(def.ExcludeCIDRs)
	entriesMap, changed := MultiNicnetworkReconcilerInstance.CIDRHandler.UpdateEntries(cidr, excludes, new)
	fmt.Printf("EntryMap: %v\n", entriesMap)
	Expect(changed).To(Equal(expectChange))

	expectedPodCIDR := 0
	if changed {
		snapshot := MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.ListCache()
		Expect(len(snapshot)).Should(BeNumerically(">", 0))
		for _, hif := range snapshot {
			for _, iface := range hif.Spec.Interfaces {
				for _, masterAddresses := range networkAddresses {
					if iface.NetAddress == masterAddresses {
						expectedPodCIDR += 1
						break
					}
				}
			}
		}
		Expect(expectedPodCIDR).Should(BeNumerically(">", 0))
		totalPodCIDR := 0
		for _, entry := range entriesMap {
			totalPodCIDR += len(entry.Hosts)
			fmt.Printf("hosts: %v\n", entry.Hosts)
		}
		fmt.Printf("cidr total pod: %d\n", totalPodCIDR)
		Expect(totalPodCIDR).To(Equal(expectedPodCIDR))
	}
	reservedInterfaceIndex := make(map[int]bool)
	newEntries := []multinicv1.CIDREntry{}
	for _, entry := range entriesMap {
		Expect(entry.InterfaceIndex).Should(BeNumerically(">=", 0))
		found := reservedInterfaceIndex[entry.InterfaceIndex]
		Expect(found).To(Equal(false))
		reservedInterfaceIndex[entry.InterfaceIndex] = true
		newEntries = append(newEntries, entry)
	}
	return multinicv1.CIDRSpec{
		Config: def,
		CIDRs:  newEntries,
	}
}

var _ = Describe("Test Multi-NIC IPAM", func() {
	cniVersion := "0.3.0"
	cniType := "ipvlan"
	mode := "l2"
	mtu := 1500
	cniArgs := make(map[string]string)
	cniArgs["mode"] = mode
	cniArgs["mtu"] = fmt.Sprintf("%d", mtu)
	multinicnetwork := GetMultiNicCNINetwork("test-ipam", cniVersion, cniType, cniArgs)
	It("Dynamically compute CIDR", func() {
		ipamConfig, err := MultiNicnetworkReconcilerInstance.GetIPAMConfig(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		cidr, err := MultiNicnetworkReconcilerInstance.CIDRHandler.NewCIDR(*ipamConfig, multinicnetwork.GetNamespace())
		Expect(err).NotTo(HaveOccurred())
		cidr = updateCIDR(multinicnetwork, cidr, true, true)
		cidr = updateCIDR(multinicnetwork, cidr, false, false)
		// Add Host
		fmt.Println("Add Host")
		newHostName := "newHost"
		newHostIndex := MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SafeCache.GetSize()
		newHif := generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface
		fmt.Println("Add Interface")
		newInterfaceName := "eth99"
		newNetworkPrefix := "0.0.0.0"
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Interface
		fmt.Println("Remove Interface")
		newHif = generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface back
		fmt.Println("Add Interface Back")
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Host
		fmt.Println("Remove Host")
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Host back
		fmt.Println("Add Host Back")
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		updateCIDR(multinicnetwork, cidr, false, true)
		// Clean up
		MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
	})

	It("Sync CIDR/IPPool", func() {
		// get index at bytes[3]
		podCIDR := "192.168.0.0/16"
		unsyncedIp := "192.168.0.10"
		contains, index := MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(10))
		// get index at bytes[2]
		unsyncedIp = "192.168.1.1"
		contains, index = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(257))
		// get index at bytes[1]
		podCIDR = "10.0.0.0/8"
		unsyncedIp = "10.1.1.1"
		contains, index = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(256*256 + 256 + 1))
		// uncontain
		podCIDR = "192.168.0.0/26"
		unsyncedIp = "192.168.1.1"
		contains, _ = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(false))
	})

	It("Empty subnet", func() {
		emptySubnetMultinicnetwork := GetMultiNicCNINetwork("empty-ipam", cniVersion, cniType, cniArgs)
		emptySubnetMultinicnetwork.Spec.Subnet = ""
		ipamConfig, err := MultiNicnetworkReconcilerInstance.GetIPAMConfig(emptySubnetMultinicnetwork)
		ipamConfig.InterfaceBlock = 0
		ipamConfig.HostBlock = 4
		Expect(err).NotTo(HaveOccurred())
		cidrSpec, err := MultiNicnetworkReconcilerInstance.CIDRHandler.GenerateCIDRFromHostSubnet(*ipamConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
		fmt.Println(cidrSpec)
		hostIPs := MultiNicnetworkReconcilerInstance.CIDRHandler.GetHostAddressesToExclude()
		Expect(len(hostIPs)).To(BeEquivalentTo(len(interfaceNames) * MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.GetSize()))
	})
})
