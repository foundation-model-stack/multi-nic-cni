/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"fmt"
	"time"

	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
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
	multinicnetwork := getMultiNicCNINetwork("test-ipam", cniVersion, cniType, cniArgs)
	It("handle multi-nic ipam", func() {
		err := multinicnetworkReconciler.Client.Create(context.Background(), multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		defer multinicnetworkReconciler.Client.Delete(context.Background(), multinicnetwork)

		var multiNicNetworkInstance *netcogadvisoriov1.MultiNicNetwork
		maxTry := 10
		count := 0
		for {
			multiNicNetworkInstance, err = multinicnetworkReconciler.CIDRHandler.GetNetwork(multinicnetwork.GetName())
			if err == nil || count > maxTry {
				break
			}
			time.Sleep(1)
			count += 1
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(len(multiNicNetworkInstance.Status.ComputeResults)).To(Equal(0))
		multinicnetworkReconciler.HandleMultiNicIPAM(multiNicNetworkInstance)
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

		multiNicNetworkInstance, err = multinicnetworkReconciler.CIDRHandler.GetNetwork(multinicnetwork.GetName())
		Expect(err).NotTo(HaveOccurred())
		Expect(len(multiNicNetworkInstance.Status.ComputeResults)).To(Equal(len(cidr.Spec.CIDRs)))
		for i, entry := range cidr.Spec.CIDRs {
			Expect(multiNicNetworkInstance.Status.ComputeResults[i].NumOfHost).To(Equal(len(entry.Hosts)))
		}
		Expect(multiNicNetworkInstance.Status.Status).To(Equal(netcogadvisoriov1.RouteNoApplied))

		err = multinicnetworkReconciler.CallFinalizer(multinicnetworkReconciler.Log, multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})
})
