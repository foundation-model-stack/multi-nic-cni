/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin_test

import (
	"encoding/json"
	"errors"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type testMellanoxPlugin struct {
	*MellanoxPlugin
	mockResources []SrIoVResource
}

func (p *testMellanoxPlugin) GetSrIoVResources() []SrIoVResource {
	return p.mockResources
}

// Override GetConfig to use our mock GetSrIoVResources
func (p *testMellanoxPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	annotation := make(map[string]string)
	// get resource from nicclusterpolicy
	rs := p.GetSrIoVResources()
	resourceAnnotation := GetCombinedResourceNames(rs)
	if resourceAnnotation == "" {
		msg := "failed to get resource annotation from sriov plugin config"
		return "", annotation, errors.New(msg)
	}

	// Create a map to store the full configuration
	confMap := map[string]interface{}{
		"cniVersion": net.Spec.MainPlugin.CNIVersion,
		"type":       HOST_DEVICE_TYPE,
		"name":       net.ObjectMeta.Name,
	}

	// Parse and add IPAM configuration
	var ipamConfig map[string]interface{}
	if err := json.Unmarshal([]byte(net.Spec.IPAM), &ipamConfig); err != nil {
		return "", annotation, err
	}
	confMap["ipam"] = ipamConfig

	// Add empty DNS configuration
	confMap["dns"] = map[string]interface{}{}

	// Marshal the complete configuration
	confBytes, err := json.Marshal(confMap)
	if err != nil {
		return "", annotation, err
	}

	annotation[RESOURCE_ANNOTATION] = resourceAnnotation
	return string(confBytes), annotation, nil
}

var _ = Describe("Mellanox Plugin", func() {
	var (
		plugin     *testMellanoxPlugin
		testConfig *rest.Config
	)

	BeforeEach(func() {
		plugin = &testMellanoxPlugin{
			MellanoxPlugin: &MellanoxPlugin{},
		}
		testConfig = &rest.Config{
			Host: "https://localhost:8443",
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			APIPath: "api",
			ContentConfig: rest.ContentConfig{
				GroupVersion:         &schema.GroupVersion{Group: "networking.k8s.io", Version: "v1"},
				NegotiatedSerializer: nil,
			},
		}
		// Use the existing config if available
		if Cfg != nil {
			testConfig = Cfg
		}
	})

	Context("Init", Ordered, func() {
		It("should initialize successfully with valid config", func() {
			err := plugin.Init(testConfig)
			Expect(err).NotTo(HaveOccurred())

			By("checking MellanoxNetworkHandler initialization")
			Expect(plugin.MellanoxNetworkHandler).NotTo(BeNil())
			mellanoxNetwork, _ := schema.ParseResourceArg(MELLANOX_NETWORK_RESOURCE)
			Expect(mellanoxNetwork).NotTo(BeNil())
			Expect(plugin.MellanoxNetworkHandler.GVR).To(Equal(*mellanoxNetwork))
			Expect(plugin.MellanoxNetworkHandler.DYN).NotTo(BeNil())

			By("checking MellanoxNicClusterPolicyHandler initialization")
			Expect(plugin.MellanoxNicClusterPolicyHandler).NotTo(BeNil())
			nicClusterPolicy, _ := schema.ParseResourceArg(MELLANOX_NIC_CLUSTER_POLICY_RESOURCE)
			Expect(nicClusterPolicy).NotTo(BeNil())
			Expect(plugin.MellanoxNicClusterPolicyHandler.GVR).To(Equal(*nicClusterPolicy))
			Expect(plugin.MellanoxNicClusterPolicyHandler.DYN).NotTo(BeNil())
		})

		When("config is invalid", func() {
			It("should return error", func() {
				invalidConfig := &rest.Config{
					Host: "https://invalid-host:8443",
					// Make TLS config invalid
					TLSClientConfig: rest.TLSClientConfig{
						CertFile: "non-existent-cert.pem",
						KeyFile:  "non-existent-key.pem",
					},
				}
				err := plugin.Init(invalidConfig)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("open non-existent-cert.pem: no such file or directory"))
			})
		})
	})

	Context("GetConfig", func() {
		var (
			net     multinicv1.MultiNicNetwork
			hifList map[string]multinicv1.HostInterface
		)

		BeforeEach(func() {
			// Initialize plugin
			err := plugin.Init(testConfig)
			Expect(err).NotTo(HaveOccurred())

			// Setup test network
			net = multinicv1.MultiNicNetwork{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-network",
				},
				Spec: multinicv1.MultiNicNetworkSpec{
					MainPlugin: multinicv1.PluginSpec{
						CNIVersion: "0.3.1",
						Type:       MELLANOX_TYPE,
					},
					IPAM: `{
						"type": "host-local",
						"subnet": "10.0.0.0/24",
						"rangeStart": "10.0.0.10",
						"rangeEnd": "10.0.0.20"
					}`,
				},
			}

			hifList = make(map[string]multinicv1.HostInterface)
		})

		It("should generate valid CNI config and resource annotations", func() {
			// Set mock resources
			plugin.mockResources = []SrIoVResource{
				{
					Prefix:       DEFAULT_MELLANOX_PREFIX,
					ResourceName: "test-resource",
				},
			}

			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).NotTo(HaveOccurred())

			// Verify config
			Expect(config).To(ContainSubstring(`"cniVersion":"0.3.1"`))
			Expect(config).To(ContainSubstring(`"type":"host-device"`))
			Expect(config).To(ContainSubstring(`"name":"test-network"`))

			// Parse the config as JSON to verify IPAM fields
			var configMap map[string]interface{}
			err = json.Unmarshal([]byte(config), &configMap)
			Expect(err).NotTo(HaveOccurred())

			// Verify IPAM configuration
			ipam, ok := configMap["ipam"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(ipam["type"]).To(Equal("host-local"))
			Expect(ipam["subnet"]).To(Equal("10.0.0.0/24"))
			Expect(ipam["rangeStart"]).To(Equal("10.0.0.10"))
			Expect(ipam["rangeEnd"]).To(Equal("10.0.0.20"))

			Expect(config).To(ContainSubstring(`"dns":{}`))

			// Verify annotations
			Expect(annotations).To(HaveKey(RESOURCE_ANNOTATION))
			Expect(annotations[RESOURCE_ANNOTATION]).To(Equal("nvidia.com/test-resource"))
		})

		It("should return error when no SrIoV resources are available", func() {
			// Set empty mock resources
			plugin.mockResources = []SrIoVResource{}

			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get resource annotation from sriov plugin config"))
			Expect(config).To(BeEmpty())
			Expect(annotations).To(BeEmpty())
		})

		It("should return error when IPAM config is invalid", func() {
			// Set mock resources
			plugin.mockResources = []SrIoVResource{
				{
					Prefix:       DEFAULT_MELLANOX_PREFIX,
					ResourceName: "test-resource",
				},
			}

			// Set invalid IPAM config
			net.Spec.IPAM = "invalid json"

			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeEmpty())
			Expect(annotations).To(BeEmpty())
		})
	})

	Context("GetSrIoVResources", func() {
		BeforeEach(func() {
			// Initialize plugin
			err := plugin.Init(testConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully retrieve SrIoV resources from policy", func() {
			// Set mock resources
			plugin.mockResources = []SrIoVResource{
				{
					Prefix:       DEFAULT_MELLANOX_PREFIX,
					ResourceName: "host_dev0",
				},
				{
					Prefix:       DEFAULT_MELLANOX_PREFIX,
					ResourceName: "host_dev1",
				},
			}

			resources := plugin.GetSrIoVResources()
			Expect(resources).To(HaveLen(2))
			Expect(resources[0].Prefix).To(Equal(DEFAULT_MELLANOX_PREFIX))
			Expect(resources[0].ResourceName).To(Equal("host_dev0"))
			Expect(resources[1].Prefix).To(Equal(DEFAULT_MELLANOX_PREFIX))
			Expect(resources[1].ResourceName).To(Equal("host_dev1"))

			// Test resource annotation formatting
			annotation := GetCombinedResourceNames(resources)
			Expect(annotation).To(Equal("nvidia.com/host_dev0,nvidia.com/host_dev1"))
		})

		It("should return empty list when no resources are available", func() {
			// Set empty mock resources
			plugin.mockResources = []SrIoVResource{}

			resources := plugin.GetSrIoVResources()
			Expect(resources).To(BeEmpty())

			// Test empty resource annotation
			annotation := GetCombinedResourceNames(resources)
			Expect(annotation).To(BeEmpty())
		})

		It("should handle resources with different prefixes", func() {
			// Set mock resources with different prefixes
			plugin.mockResources = []SrIoVResource{
				{
					Prefix:       DEFAULT_MELLANOX_PREFIX,
					ResourceName: "host_dev0",
				},
				{
					Prefix:       "custom.com",
					ResourceName: "host_dev1",
				},
			}

			resources := plugin.GetSrIoVResources()
			Expect(resources).To(HaveLen(2))
			Expect(resources[0].Prefix).To(Equal(DEFAULT_MELLANOX_PREFIX))
			Expect(resources[1].Prefix).To(Equal("custom.com"))

			// Test resource annotation with different prefixes
			annotation := GetCombinedResourceNames(resources)
			Expect(annotation).To(Equal("nvidia.com/host_dev0,custom.com/host_dev1"))
		})
	})
})
