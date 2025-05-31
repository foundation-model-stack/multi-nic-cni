/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("NetAttachDef test", func() {
	var handler *NetAttachDefHandler
	var mainPlugin string
	var annotations map[string]string

	var (
		cniVersion = "0.3.0"
		cniType    = IPVLAN_TYPE
		cniArgs    = map[string]string{
			"mode": "l2",
			"mtu":  "1500",
		}
		multinicnetworkName = "test-nad"
		multinicnetwork     = getMultiNicCNINetwork(multinicnetworkName, cniVersion, cniType, cniArgs)
	)

	Context("handler", Ordered, func() {
		ctx := context.TODO()

		BeforeAll(func() {
			var err error
			handler, err = GetNetAttachDefHandler(Cfg)
			Expect(err).To(BeNil())

			ipvlanPlugin := &IPVLANPlugin{}
			mainPlugin, annotations, err = ipvlanPlugin.GetConfig(*multinicnetwork, nil)
			Expect(err).To(BeNil())
		})

		It("create and delete", func() {
			err := handler.CreateOrUpdate(multinicnetwork, mainPlugin, annotations)
			Expect(err).To(BeNil())
			namespaceList := corev1.NamespaceList{}
			err = K8sClient.List(ctx, &namespaceList)
			Expect(err).To(BeNil())
			Expect(len(namespaceList.Items)).To(BeNumerically(">", 0))
			for _, namespace := range namespaceList.Items {
				By(fmt.Sprintf("checking creation in namespace %s", namespace.Name))
				Eventually(func(g Gomega) {
					_, err := handler.Get(multinicnetworkName, namespace.Name)
					g.Expect(err).To(BeNil())
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			}
			err = handler.DeleteNets(multinicnetwork)
			Expect(err).To(BeNil())
			for _, namespace := range namespaceList.Items {
				By(fmt.Sprintf("checking deletion in namespace %s", namespace.Name))
				Eventually(func(g Gomega) {
					_, err := handler.Get(multinicnetworkName, namespace.Name)
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			}
		})

		Context("CheckDefChanged", func() {
			type testCase struct {
				description string
				def1        *NetworkAttachmentDefinition
				def2        *NetworkAttachmentDefinition
				expected    bool
			}

			DescribeTable("comparing network attachment definitions",
				func(tc testCase) {
					changed := CheckDefChanged(tc.def1, tc.def2)
					Expect(changed).To(Equal(tc.expected))
				},
				Entry("identical definitions", testCase{
					description: "should not detect changes in identical definitions",
					def1: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: multinicnetworkName,
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					def2: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: multinicnetworkName,
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					expected: false,
				}),
				Entry("different configurations", testCase{
					description: "should detect changes in configuration",
					def1: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: multinicnetworkName,
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					def2: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: multinicnetworkName,
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l3","mtu":"1500"}`,
						},
					},
					expected: true,
				}),
				Entry("different annotation values", testCase{
					description: "should detect changes in annotation values",
					def1: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name:        multinicnetworkName,
							Annotations: map[string]string{"key1": "value1"},
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					def2: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name:        multinicnetworkName,
							Annotations: map[string]string{"key1": "value2"},
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					expected: true,
				}),
				Entry("different annotation count", testCase{
					description: "should detect changes in annotation count",
					def1: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name:        multinicnetworkName,
							Annotations: map[string]string{"key1": "value1"},
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					def2: &NetworkAttachmentDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: multinicnetworkName,
							Annotations: map[string]string{
								"key1": "value1",
								"key2": "value2",
							},
						},
						Spec: NetworkAttachmentDefinitionSpec{
							Config: `{"cniVersion":"0.3.0","type":"ipvlan","mode":"l2","mtu":"1500"}`,
						},
					},
					expected: true,
				}),
			)
		})
	})
})
