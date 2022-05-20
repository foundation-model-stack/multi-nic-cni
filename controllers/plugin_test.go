/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/plugin"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("Test GetConfig of main plugins", func() {
	cniVersion := "0.3.0"

	It("ipvlan main plugin", func() {
		cniType := "ipvlan"
		mode := "l2"
		mtu := 1500
		cniArgs := make(map[string]string)
		cniArgs["mode"] = mode
		cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

		multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)

		mainPlugin, _, err := ipvlanPlugin.GetConfig(*multinicnetwork, hifList)
		Expect(err).NotTo(HaveOccurred())
		expected := plugin.IPVLANTypeNetConf{
			NetConf: types.NetConf{
				CNIVersion: cniVersion,
				Type:       cniType,
			},
			Mode: mode,
			MTU:  mtu,
		}
		expectedBytes, _ := json.Marshal(expected)
		Expect(mainPlugin).To(Equal(string(expectedBytes)))
		isMultiNicIPAM, err := IsMultiNICIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(isMultiNicIPAM).To(Equal(true))
		err = ipvlanPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("macvlan main plugin", func() {
		cniType := "macvlan"
		mode := "l2"
		mtu := 1500
		cniArgs := make(map[string]string)
		cniArgs["mode"] = mode
		cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

		multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)

		mainPlugin, _, err := macvlanPlugin.GetConfig(*multinicnetwork, hifList)
		Expect(err).NotTo(HaveOccurred())
		expected := plugin.MACVLANTypeNetConf{
			NetConf: types.NetConf{
				CNIVersion: cniVersion,
				Type:       cniType,
			},
			Mode: mode,
			MTU:  mtu,
		}
		expectedBytes, _ := json.Marshal(expected)
		Expect(mainPlugin).To(Equal(string(expectedBytes)))
		isMultiNicIPAM, err := IsMultiNICIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(isMultiNicIPAM).To(Equal(true))
		err = macvlanPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("sriov main plugin without resource name", func() {
		cniType := "sriov"
		cniArgs := make(map[string]string)
		numVfs := 2
		isRdma := true
		cniArgs["numVfs"] = fmt.Sprintf("%d", numVfs)
		cniArgs["isRdma"] = fmt.Sprintf("%v", isRdma)
		multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)

		_, annotations, err := sriovPlugin.GetConfig(*multinicnetwork, hifList)
		Expect(err).NotTo(HaveOccurred())
		defaultResourceName := sriovPlugin.ValidateResourceName(multinicnetwork.Name)

		netName := sriovPlugin.SriovnetworkName(multinicnetwork.Name)
		sriovpolicy := &plugin.SriovNetworkNodePolicy{}
		err = sriovPlugin.SriovNetworkNodePolicyHandler.Get(multinicnetwork.Name, plugin.SRIOV_NAMESPACE, sriovpolicy)
		// SriovPolicy is created
		Expect(err).NotTo(HaveOccurred())
		Expect(sriovpolicy.Spec.NumVfs).To(Equal(numVfs))
		Expect(sriovpolicy.Spec.IsRdma).To(Equal(isRdma))

		sriovnet := &plugin.SriovNetwork{}
		err = sriovPlugin.SriovNetworkHandler.Get(netName, plugin.SRIOV_NAMESPACE, sriovnet)
		// SriovNetwork is created
		Expect(err).NotTo(HaveOccurred())
		Expect(sriovnet.Spec.ResourceName).To(Equal(defaultResourceName))
		Expect(sriovnet.Spec.ResourceName).To(Equal(sriovpolicy.Spec.ResourceName))
		Expect(sriovnet.Spec.NetworkNamespace).To(Equal(multinicnetwork.Namespace))
		Expect(annotations[plugin.RESOURCE_ANNOTATION]).To(Equal(plugin.SRIOV_RESOURCE_PREFIX + "/" + sriovnet.Spec.ResourceName))

		err = sriovPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("sriov main plugin with resource name", func() {
		cniType := "sriov"
		cniArgs := make(map[string]string)
		numVfs := 2
		isRdma := true
		resourceName := "sriovresource"
		cniArgs["numVfs"] = fmt.Sprintf("%d", numVfs)
		cniArgs["isRdma"] = fmt.Sprintf("%v", isRdma)
		cniArgs["resourceName"] = resourceName
		multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)

		_, annotations, err := sriovPlugin.GetConfig(*multinicnetwork, hifList)
		Expect(err).NotTo(HaveOccurred())
		netName := sriovPlugin.SriovnetworkName(multinicnetwork.Name)

		sriovpolicy := &plugin.SriovNetworkNodePolicy{}
		err = sriovPlugin.SriovNetworkNodePolicyHandler.Get(multinicnetwork.Name, plugin.SRIOV_NAMESPACE, sriovpolicy)
		// SriovPolicy must not be created
		Expect(err).To(HaveOccurred())

		sriovnet := &plugin.SriovNetwork{}
		err = sriovPlugin.SriovNetworkHandler.Get(netName, plugin.SRIOV_NAMESPACE, sriovnet)
		// SriovNetwork is created
		Expect(err).NotTo(HaveOccurred())
		Expect(sriovnet.Spec.ResourceName).To(Equal(resourceName))
		Expect(sriovnet.Spec.NetworkNamespace).To(Equal(multinicnetwork.Namespace))
		Expect(annotations[plugin.RESOURCE_ANNOTATION]).To(Equal(plugin.SRIOV_RESOURCE_PREFIX + "/" + sriovnet.Spec.ResourceName))

		err = sriovPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})
})
