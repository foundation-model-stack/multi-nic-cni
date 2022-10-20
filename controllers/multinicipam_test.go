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
	entriesMap, changed := multinicnetworkReconciler.CIDRHandler.updateEntries(cidr, excludes, new)
	fmt.Printf("EntryMap: %v\n", entriesMap)
	Expect(changed).To(Equal(expectChange))

	expectedPodCIDR := 0
	if changed {
		for _, hif := range HostInterfaceCache {
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
		ipamConfig, err := multinicnetworkReconciler.getIPAMConfig(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		cidr, err := multinicnetworkReconciler.CIDRHandler.newCIDR(*ipamConfig, multinicnetwork.GetNamespace())
		Expect(err).NotTo(HaveOccurred())
		cidr = updateCIDR(multinicnetwork, cidr, true, true)
		cidr = updateCIDR(multinicnetwork, cidr, false, false)
		// Add Host
		fmt.Println("Add Host")
		newHostName := "newHost"
		newHostIndex := len(HostInterfaceCache)
		newHif := generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		HostInterfaceCache[newHostName] = newHif
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface
		fmt.Println("Add Interface")
		newInterfaceName := "eth99"
		newNetworkPrefix := "0.0.0.0"
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		HostInterfaceCache[newHostName] = newHif
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Interface
		fmt.Println("Remove Interface")
		newHif = generateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
		HostInterfaceCache[newHostName] = newHif
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Interface back
		fmt.Println("Add Interface Back")
		newHif = generateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
		HostInterfaceCache[newHostName] = newHif
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Remove Host
		fmt.Println("Remove Host")
		delete(HostInterfaceCache, newHostName)
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Add Host back
		fmt.Println("Add Host Back")
		HostInterfaceCache[newHostName] = newHif
		cidr = updateCIDR(multinicnetwork, cidr, false, true)
		// Clean up
		delete(HostInterfaceCache, newHostName)
	})

	It("Empty subnet", func() {
		emptySubnetMultinicnetwork := getMultiNicCNINetwork("empty-ipam", cniVersion, cniType, cniArgs)
		emptySubnetMultinicnetwork.Spec.Subnet = ""
		ipamConfig, err := multinicnetworkReconciler.getIPAMConfig(emptySubnetMultinicnetwork)
		ipamConfig.InterfaceBlock = 0
		ipamConfig.HostBlock = 4
		Expect(err).NotTo(HaveOccurred())
		cidrSpec, err := multinicnetworkReconciler.CIDRHandler.generateCIDRFromHostSubnet(*ipamConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
		fmt.Println(cidrSpec)
		hostIPs := getHostAddressesToExclude()
		Expect(len(hostIPs)).To(BeEquivalentTo(len(interfaceNames) * len(HostInterfaceCache)))
	})
})
