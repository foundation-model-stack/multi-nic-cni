/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("Test Multi-NIC IPAM", func() {
	cniVersion := "0.3.0"
	cniType := "ipvlan"
	mode := "l2"
	mtu := 1500
	cniArgs := make(map[string]string)
	cniArgs["mode"] = mode
	cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

	multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)
	It("handle multi-nic ipam", func() {
		err := multinicnetworkReconciler.HandleMultiNicIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		cidr, err := multinicnetworkReconciler.CIDRHandler.GetCIDR(multinicnetwork.Name)
		Expect(err).NotTo(HaveOccurred())
		expectedPodCIDR := 0
		for _, hif := range hifList {
			for _, iface := range hif.Spec.Interfaces {
				for _, masterAddresses := range multinicnetwork.Spec.MasterNetAddrs {
					if iface.NetAddress == masterAddresses {
						expectedPodCIDR += 1
						break
					}
				}
			}
		}
		totalPodCIDR := 0
		for _, entry := range cidr.Spec.CIDRs {
			totalPodCIDR += len(entry.Hosts)
		}
		Expect(totalPodCIDR).To(Equal(expectedPodCIDR))

		ippoolMaps, err := multinicnetworkReconciler.CIDRHandler.IPPoolHandler.ListIPPool()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(ippoolMaps)).Should(BeNumerically(">", 0))

		err = multinicnetworkReconciler.CallFinalizer(multinicnetworkReconciler.Log, multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})
})
