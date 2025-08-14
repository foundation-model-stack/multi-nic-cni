/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin_test

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//+kubebuilder:scaffold:imports
)

var (
	globalSubnet       string   = "192.168.0.0/16"
	networkAddresses   []string = []string{"10.242.0.0/24", "10.242.1.0/24", "10.242.2.0/24", "10.242.1.0/24"}
	multiNicIPAMConfig string   = `{
		"type":           "multi-nic-ipam",
		"hostBlock":      8,
		"interfaceBlock": 2,
		"excludeCIDRs":   ["192.168.0.64/26","192.168.0.128/30"],
		"vlanMode":       "l2"
	   }`
	interfaceNames  []string = []string{"eth1", "eth2"}
	networkPrefixes []string = []string{"10.242.0.", "10.242.1."}
)

var _ = Describe("Common Plugin Test", func() {
	It("RemoveEmpty", func() {
		pluginStr := `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":""}`
		cniArgs := map[string]string{"mode": "l2"}
		output := RemoveEmpty(cniArgs, pluginStr)
		Expect(output).To(ContainSubstring(`"cniVersion":"0.3.0"`))
		Expect(output).To(ContainSubstring(`"type":"ipvlan"`))
		Expect(output).To(ContainSubstring(`"mode":"l2"`))
		Expect(output).NotTo(ContainSubstring(`"mtu"`))
		invalidPluginStr := "invalid"
		output = RemoveEmpty(cniArgs, invalidPluginStr)
		Expect(output).To(Equal(invalidPluginStr))
	})
})

var _ = Describe("Test GetConfig of main plugins", func() {
	cniVersion := "0.3.0"

	It("ipvlan main plugin", func() {
		ipvlanPlugin := &IPVLANPlugin{}
		cniType := IPVLAN_TYPE
		mode := "l2"
		mtu := 1500
		cniArgs := make(map[string]string)
		cniArgs["mode"] = mode
		cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

		multinicnetwork := getMultiNicCNINetwork("test-ipvlanl2", cniVersion, cniType, cniArgs)

		mainPlugin, _, err := ipvlanPlugin.GetConfig(*multinicnetwork, nil)
		Expect(err).NotTo(HaveOccurred())
		expected := IPVLANTypeNetConf{
			NetConf: types.NetConf{
				CNIVersion: cniVersion,
				Type:       cniType,
			},
			Mode: mode,
			MTU:  mtu,
		}
		expectedBytes, _ := json.Marshal(expected)
		Expect(mainPlugin).To(Equal(string(expectedBytes)))
		isMultiNicIPAM, err := isMultiNICIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(isMultiNicIPAM).To(Equal(true))
		err = ipvlanPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("macvlan main plugin", func() {
		macvlanPlugin := &MACVLANPlugin{}
		cniType := MACVLAN_TYPE
		mode := "l2"
		mtu := 1500
		cniArgs := make(map[string]string)
		cniArgs["mode"] = mode
		cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

		multinicnetwork := getMultiNicCNINetwork("test-macvlan", cniVersion, cniType, cniArgs)

		mainPlugin, _, err := macvlanPlugin.GetConfig(*multinicnetwork, nil)
		Expect(err).NotTo(HaveOccurred())
		expected := MACVLANTypeNetConf{
			NetConf: types.NetConf{
				CNIVersion: cniVersion,
				Type:       cniType,
			},
			Mode: mode,
			MTU:  mtu,
		}
		expectedBytes, _ := json.Marshal(expected)
		Expect(mainPlugin).To(Equal(string(expectedBytes)))
		isMultiNicIPAM, err := isMultiNICIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(isMultiNicIPAM).To(Equal(true))
		err = macvlanPlugin.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("aws-ipvlan main plugin", func() {
		awsIPVlan := &AwsVpcCNIPlugin{}
		cniType := AWS_IPVLAN_TYPE
		mode := "l3"
		mtu := 1500
		cniArgs := make(map[string]string)
		cniArgs["mode"] = mode
		cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

		multinicnetwork := getMultiNicCNINetwork("test-aws-ipvlan", cniVersion, cniType, cniArgs)

		mainPlugin, _, err := awsIPVlan.GetConfig(*multinicnetwork, nil)
		Expect(err).NotTo(HaveOccurred())
		expected := AWSIPVLANNetConf{
			NetConf: types.NetConf{
				CNIVersion: cniVersion,
				Type:       cniType,
			},
			Mode: mode,
			MTU:  mtu,
		}
		expectedBytes, _ := json.Marshal(expected)
		Expect(mainPlugin).To(Equal(string(expectedBytes)))
		isMultiNicIPAM, err := isMultiNICIPAM(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(isMultiNicIPAM).To(Equal(true))
		err = awsIPVlan.CleanUp(*multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("SR-IoV", Ordered, func() {
		sriovPlugin := &SriovPlugin{}
		cniType := SRIOV_TYPE
		numVfs := 2
		isRdma := true
		nodes := generateNodes()
		hifList := generateHostInterfaceList(nodes)

		BeforeAll(func() {
			err := sriovPlugin.Init(Cfg)
			Expect(err).ToNot(HaveOccurred())
		})

		It("without resource name", func() {
			cniArgs := make(map[string]string)
			cniArgs["numVfs"] = fmt.Sprintf("%d", numVfs)
			cniArgs["isRdma"] = fmt.Sprintf("%v", isRdma)
			multinicnetwork := getMultiNicCNINetwork("test-sriov-default", cniVersion, cniType, cniArgs)

			_, annotations, err := sriovPlugin.GetConfig(*multinicnetwork, hifList)
			Expect(err).NotTo(HaveOccurred())
			defaultResourceName := ValidateResourceName(multinicnetwork.Name)

			netName := sriovPlugin.SriovnetworkName(multinicnetwork.Name)
			sriovpolicy := &SriovNetworkNodePolicy{}
			err = sriovPlugin.SriovNetworkNodePolicyHandler.Get(multinicnetwork.Name, SRIOV_NAMESPACE, sriovpolicy)
			// SriovPolicy is created
			Expect(err).NotTo(HaveOccurred())
			Expect(sriovpolicy.Spec.NumVfs).To(Equal(numVfs))
			Expect(sriovpolicy.Spec.IsRdma).To(Equal(isRdma))

			sriovnet := &SriovNetwork{}
			err = sriovPlugin.SriovNetworkHandler.Get(netName, SRIOV_NAMESPACE, sriovnet)
			// SriovNetwork is created
			Expect(err).NotTo(HaveOccurred())
			Expect(sriovnet.Spec.ResourceName).To(Equal(defaultResourceName))
			Expect(sriovnet.Spec.ResourceName).To(Equal(sriovpolicy.Spec.ResourceName))
			Expect(sriovnet.Spec.NetworkNamespace).To(Equal(multinicnetwork.Namespace))
			Expect(annotations[RESOURCE_ANNOTATION]).To(Equal(SRIOV_RESOURCE_PREFIX + "/" + sriovnet.Spec.ResourceName))

			err = sriovPlugin.CleanUp(*multinicnetwork)
			Expect(err).NotTo(HaveOccurred())
		})

		It("with resource name", func() {
			cniArgs := make(map[string]string)
			cniArgs["numVfs"] = fmt.Sprintf("%d", numVfs)
			cniArgs["isRdma"] = fmt.Sprintf("%v", isRdma)
			resourceName := "sriovresource"
			cniArgs["resourceName"] = resourceName

			multinicnetwork := getMultiNicCNINetwork("test-sriov", cniVersion, cniType, cniArgs)

			_, annotations, err := sriovPlugin.GetConfig(*multinicnetwork, hifList)
			Expect(err).NotTo(HaveOccurred())
			netName := sriovPlugin.SriovnetworkName(multinicnetwork.Name)

			sriovpolicy := &SriovNetworkNodePolicy{}
			err = sriovPlugin.SriovNetworkNodePolicyHandler.Get(multinicnetwork.Name, SRIOV_NAMESPACE, sriovpolicy)
			// SriovPolicy must not be created
			Expect(err).To(HaveOccurred())

			sriovnet := &SriovNetwork{}
			err = sriovPlugin.SriovNetworkHandler.Get(netName, SRIOV_NAMESPACE, sriovnet)
			// SriovNetwork is created
			Expect(err).NotTo(HaveOccurred())
			Expect(sriovnet.Spec.ResourceName).To(Equal(resourceName))
			Expect(sriovnet.Spec.NetworkNamespace).To(Equal(multinicnetwork.Namespace))
			Expect(annotations[RESOURCE_ANNOTATION]).To(Equal(SRIOV_RESOURCE_PREFIX + "/" + sriovnet.Spec.ResourceName))

			err = sriovPlugin.CleanUp(*multinicnetwork)
			Expect(err).NotTo(HaveOccurred())

		})
	})

	It("mellanox main plugin - GetSrIoVResource", func() {
		sriovResourceList := `
		 {
			 "resourceList": [
				 {
					 "resourcePrefix": "nvidia.com",
					 "resourceName": "host_dev0",
					 "selectors": {
						 "vendors": ["15b3"],
						 "isRdma": true,
						 "pciAddresses": ["0000:00:00.0"]
					 }
				 },
				 {
					 "resourcePrefix": "nvidia.com",
					 "resourceName": "host_dev1",
					 "selectors": {
						 "vendors": ["15b3"],
						 "isRdma": true,
						 "pciAddresses": ["0000:00:00.1"]
					 }
				 }
			 ]
		 }
		 `
		expectedAnnotation := "nvidia.com/host_dev0,nvidia.com/host_dev1"
		sriovPluginConfig := &DevicePluginSpec{
			ImageSpecWithConfig: ImageSpecWithConfig{
				Config: &sriovResourceList,
				ImageSpec: ImageSpec{
					Image:            "sriov-network-device-plugin",
					Repository:       "ghcr.io/k8snetworkplumbingwg",
					Version:          "v3.5.1",
					ImagePullSecrets: []string{},
				},
			},
		}
		rs, err := GetSrIoVResourcesFromSrIoVPlugin(sriovPluginConfig)
		Expect(err).To(BeNil())
		Expect(len(rs)).To(Equal(2))
		resourceAnnotation := GetCombinedResourceNames(rs)
		Expect(resourceAnnotation).To(Equal(expectedAnnotation))
	})
})

// getMultiNicCNINetwork returns MultiNicNetwork object
func getMultiNicCNINetwork(name string, cniVersion string, cniType string, cniArgs map[string]string) *multinicv1.MultiNicNetwork {
	return &multinicv1.MultiNicNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: multinicv1.MultiNicNetworkSpec{
			Subnet:         globalSubnet,
			IPAM:           multiNicIPAMConfig,
			IsMultiNICIPAM: true,
			MainPlugin: multinicv1.PluginSpec{
				CNIVersion: cniVersion,
				Type:       cniType,
				CNIArgs:    cniArgs,
			},
			MasterNetAddrs: networkAddresses,
		},
	}
}

func isMultiNICIPAM(instance *multinicv1.MultiNicNetwork) (bool, error) {
	simpleIPAM := &types.IPAM{}
	err := json.Unmarshal([]byte(instance.Spec.IPAM), simpleIPAM)
	if err != nil {
		return false, fmt.Errorf("%s: %v", instance.Spec.IPAM, err)
	}
	return simpleIPAM.Type == vars.MultiNICIPAMType, nil
}

func generateNodes() []corev1.Node {
	nodes := []corev1.Node{}
	hostNamePrefix := "worker-"
	hostNum := 5

	for i := 0; i < hostNum; i++ {
		hostName := fmt.Sprintf("%s%d", hostNamePrefix, i)
		node := corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: hostName,
			},
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func generateHostInterfaceList(nodes []corev1.Node) map[string]multinicv1.HostInterface {

	hifList := make(map[string]multinicv1.HostInterface)
	for i, node := range nodes {
		hostName := node.GetName()
		hif := generateNewHostInterface(hostName, interfaceNames, networkPrefixes, i)
		hifList[hostName] = hif
	}
	return hifList
}

// generateNewHostInterface generates new host
func generateNewHostInterface(hostName string, interfaceNames []string, networkPrefixes []string, i int) multinicv1.HostInterface {
	ifaces := []multinicv1.InterfaceInfoType{}
	for index, ifaceName := range interfaceNames {
		iface := multinicv1.InterfaceInfoType{
			InterfaceName: ifaceName,
			NetAddress:    networkAddresses[index],
			HostIP:        fmt.Sprintf("%s%d", networkPrefixes[index], i),
		}
		ifaces = append(ifaces, iface)
	}
	hif := multinicv1.HostInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostName,
			Labels: map[string]string{
				vars.TestModeLabel: "true",
			},
		},
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   hostName,
			Interfaces: ifaces,
		},
	}
	return hif
}
