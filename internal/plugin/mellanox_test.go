/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin_test

import (
	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

var _ = Describe("Mellanox Plugin", func() {
	var (
		plugin     *MellanoxPlugin
		testConfig *rest.Config
	)

	BeforeEach(func() {
		plugin = &MellanoxPlugin{}
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
})
