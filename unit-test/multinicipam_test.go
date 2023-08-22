/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/compute"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

func updateCIDR(multinicnetwork *multinicv1.MultiNicNetwork, cidr multinicv1.CIDRSpec, new, expectChange bool) multinicv1.CIDRSpec {
	def := cidr.Config
	excludes := compute.SortAddress(def.ExcludeCIDRs)
	entriesMap, changed := multinicnetworkReconciler.CIDRHandler.UpdateEntries(cidr, excludes, new)
	fmt.Printf("EntryMap: %v\n", entriesMap)
	Expect(changed).To(Equal(expectChange))

	expectedPodCIDR := 0
	if changed {
		snapshot := multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.ListCache()
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
	multinicnetwork := getMultiNicCNINetwork("test-ipam", cniVersion, cniType, cniArgs)
	It("Dynamically compute CIDR", func() {
		ipamConfig, err := multinicnetworkReconciler.GetIPAMConfig(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		cidr, err := multinicnetworkReconciler.CIDRHandler.NewCIDR(*ipamConfig, multinicnetwork.GetNamespace())
		Expect(err).NotTo(HaveOccurred())
		cidr = updateCIDR(multinicnetwork, cidr, true, true)
		cidr = updateCIDR(multinicnetwork, cidr, false, false)
		// Add Host
		fmt.Println("Add Host")
		newHostName := "newHost"
		newHostIndex := multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SafeCache.GetSize()
		newHif := generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface
		fmt.Println("Add Interface")
		newInterfaceName := "eth99"
		newNetworkPrefix := "0.0.0.0"
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Interface
		fmt.Println("Remove Interface")
		newHif = generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface back
		fmt.Println("Add Interface Back")
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Host
		fmt.Println("Remove Host")
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Host back
		fmt.Println("Add Host Back")
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
		updateCIDR(multinicnetwork, cidr, false, true)
		// Clean up
		multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)

		// Test empty subnet
		fmt.Println("Empty subnet")
		emptySubnetMultinicnetwork := getMultiNicCNINetwork("empty-ipam", cniVersion, cniType, cniArgs)
		emptySubnetMultinicnetwork.Spec.Subnet = ""
		ipamConfig, err = multinicnetworkReconciler.GetIPAMConfig(emptySubnetMultinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		ipamConfig.InterfaceBlock = 0
		ipamConfig.HostBlock = 4
		cidrSpec, err := multinicnetworkReconciler.CIDRHandler.GenerateCIDRFromHostSubnet(*ipamConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
		fmt.Println(cidrSpec)
		hostIPs := multinicnetworkReconciler.CIDRHandler.GetHostAddressesToExclude()
		Expect(len(hostIPs)).To(BeEquivalentTo(len(interfaceNames) * multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.GetSize()))

		// Empty subnet and zero hostblock
		fmt.Println("Empty subnet and zero hostblock")
		ipamConfig.HostBlock = 0
		ipamConfig.ExcludeCIDRs = []string{}
		cidrSpec, err = multinicnetworkReconciler.CIDRHandler.GenerateCIDRFromHostSubnet(*ipamConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
		fmt.Println(cidrSpec)
		hifSnapshot := multinicnetworkReconciler.CIDRHandler.HostInterfaceHandler.ListCache()
		for _, cidr := range cidrSpec.CIDRs {
			for _, hostInfo := range cidr.Hosts {
				hif, found := hifSnapshot[hostInfo.HostName]
				Expect(found).To(Equal(true))
				found = false
				for _, iface := range hif.Spec.Interfaces {
					if hostInfo.InterfaceName == iface.InterfaceName {
						netAddr := iface.NetAddress
						Expect(cidr.VlanCIDR).To(Equal(netAddr))
						Expect(hostInfo.PodCIDR).To(Equal(netAddr))
						found = true
						break
					}
				}
				Expect(found).To(Equal(true))
			}
		}
	})

	It("Sync CIDR/IPPool", func() {
		// get index at bytes[3]
		podCIDR := "192.168.0.0/16"
		unsyncedIp := "192.168.0.10"
		contains, index := multinicnetworkReconciler.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(10))
		// get index at bytes[2]
		unsyncedIp = "192.168.1.1"
		contains, index = multinicnetworkReconciler.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(257))
		// get index at bytes[1]
		podCIDR = "10.0.0.0/8"
		unsyncedIp = "10.1.1.1"
		contains, index = multinicnetworkReconciler.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(true))
		Expect(index).To(Equal(256*256 + 256 + 1))
		// uncontain
		podCIDR = "192.168.0.0/26"
		unsyncedIp = "192.168.1.1"
		contains, _ = multinicnetworkReconciler.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
		Expect(contains).To(Equal(false))
	})
})
