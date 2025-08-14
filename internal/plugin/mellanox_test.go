/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type testMellanoxPlugin struct {
	*MellanoxPlugin
	mockSriovPlugin *DevicePluginSpec
}

func (p *testMellanoxPlugin) getPolicy() (*NicClusterPolicy, error) {
	return &NicClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-policy",
		},
		Spec: NicClusterPolicySpec{
			SriovDevicePlugin: p.mockSriovPlugin,
		},
	}, nil
}

func (p *testMellanoxPlugin) GetSrIoVResources() []SrIoVResource {
	policy, err := p.getPolicy()
	if err != nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("failed to get policy: %v", err))
		return nil
	}
	if policy.Spec.SriovDevicePlugin == nil || policy.Spec.SriovDevicePlugin.Config == nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("no sriov device plugin config set in %s", policy.ObjectMeta.Name))
		return nil
	}
	rs, err := GetSrIoVResourcesFromSrIoVPlugin(policy.Spec.SriovDevicePlugin)
	if err != nil || len(rs) == 0 {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("no readable value from sriov config (%s): %v", *policy.Spec.SriovDevicePlugin.Config, err))
	}
	return rs
}

func (p *testMellanoxPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	annotation := make(map[string]string)
	var err error
	// get resource from nicclusterpolicy
	rs := p.GetSrIoVResources()
	resourceAnnotation := GetCombinedResourceNames(rs)
	if resourceAnnotation == "" {
		msg := "failed to get resource annotation from sriov plugin config"
		vars.NetworkLog.V(2).Info(msg)
		return "", annotation, errors.New(msg)
	}
	conf := HostDeviceTypeNetConf{
		NetConf: types.NetConf{
			CNIVersion: net.Spec.MainPlugin.CNIVersion,
			Type:       HOST_DEVICE_TYPE,
			Name:       net.ObjectMeta.Name,
		},
	}
	err = json.Unmarshal([]byte(net.Spec.IPAM), &conf.IPAM)
	if err != nil {
		return "", annotation, err
	}
	confBytes, err := json.Marshal(conf)
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
		config := `{
			"resourceList": [
				{
					"resourcePrefix": "nvidia.com",
					"resourceName": "test-resource"
				}
			]
		}`
		plugin = &testMellanoxPlugin{
			MellanoxPlugin: &MellanoxPlugin{},
			mockSriovPlugin: &DevicePluginSpec{
				ImageSpecWithConfig: ImageSpecWithConfig{
					Config: &config,
				},
			},
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

		It("should generate valid CNI config", func() {
			// Verify GetSrIoVResources returns expected resources
			resources := plugin.GetSrIoVResources()
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Prefix).To(Equal("nvidia.com"))
			Expect(resources[0].ResourceName).To(Equal("test-resource"))

			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).NotTo(HaveOccurred())

			// Parse the config as JSON
			var configMap map[string]interface{}
			err = json.Unmarshal([]byte(config), &configMap)
			Expect(err).NotTo(HaveOccurred())

			// Verify basic CNI fields
			Expect(configMap["cniVersion"]).To(Equal("0.3.1"))
			Expect(configMap["type"]).To(Equal("host-device"))
			Expect(configMap["name"]).To(Equal("test-network"))

			// Verify IPAM configuration
			ipam, ok := configMap["ipam"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(ipam["type"]).To(Equal("host-local"))
			Expect(ipam["subnet"]).To(Equal("10.0.0.0/24"))
			Expect(ipam["rangeStart"]).To(Equal("10.0.0.10"))
			Expect(ipam["rangeEnd"]).To(Equal("10.0.0.20"))

			// Verify annotations
			Expect(annotations).To(HaveKey(RESOURCE_ANNOTATION))
			Expect(annotations[RESOURCE_ANNOTATION]).To(Equal("nvidia.com/test-resource"))
		})

		It("should handle invalid IPAM config", func() {
			net.Spec.IPAM = "invalid json"
			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeEmpty())
			Expect(annotations).To(BeEmpty())
		})

		It("should handle empty resource list", func() {
			emptyConfig := `{
				"resourceList": []
			}`
			plugin.mockSriovPlugin.Config = &emptyConfig
			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get resource annotation from sriov plugin config"))
			Expect(config).To(BeEmpty())
			Expect(annotations).To(BeEmpty())
		})

		It("should handle invalid resource list", func() {
			invalidConfig := `{
				"resourceList": "invalid"
			}`

			plugin.mockSriovPlugin.Config = &invalidConfig
			config, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get resource annotation from sriov plugin config"))
			Expect(config).To(BeEmpty())
			Expect(annotations).To(BeEmpty())
		})

		It("should successfully retrieve SrIoV resources from policy", func() {
			// Test with a valid policy configuration
			resources := plugin.GetSrIoVResources()
			Expect(resources).To(HaveLen(1))
			Expect(resources[0].Prefix).To(Equal("nvidia.com"))
			Expect(resources[0].ResourceName).To(Equal("test-resource"))
		})

		It("should return empty list when no resources are available", func() {
			// Test with nil SriovDevicePlugin
			plugin.mockSriovPlugin = nil
			resources := plugin.GetSrIoVResources()
			Expect(resources).To(BeEmpty())

			// Test with nil Config
			plugin.mockSriovPlugin = &DevicePluginSpec{}
			resources = plugin.GetSrIoVResources()
			Expect(resources).To(BeEmpty())
		})

		It("should handle resources with different prefixes", func() {
			multiResourceConfig := `{
				"resourceList": [
					{
						"resourcePrefix": "nvidia.com",
						"resourceName": "test-resource-1"
					},
					{
						"resourcePrefix": "mellanox.com",
						"resourceName": "test-resource-2"
					}
				]
			}`
			plugin.mockSriovPlugin.Config = &multiResourceConfig

			resources := plugin.GetSrIoVResources()
			Expect(resources).To(HaveLen(2))
			Expect(resources[0].Prefix).To(Equal("nvidia.com"))
			Expect(resources[0].ResourceName).To(Equal("test-resource-1"))
			Expect(resources[1].Prefix).To(Equal("mellanox.com"))
			Expect(resources[1].ResourceName).To(Equal("test-resource-2"))

			// Verify combined resource annotation
			_, annotations, err := plugin.GetConfig(net, hifList)
			Expect(err).NotTo(HaveOccurred())
			Expect(annotations).To(HaveKey(RESOURCE_ANNOTATION))
			Expect(annotations[RESOURCE_ANNOTATION]).To(Equal("nvidia.com/test-resource-1,mellanox.com/test-resource-2"))
		})
	})
})
