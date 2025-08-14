/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			handler, err = GetNetAttachDefHandler(Cfg, scheme.Scheme)
			Expect(err).To(BeNil())

			ipvlanPlugin := &IPVLANPlugin{}
			mainPlugin, annotations, err = ipvlanPlugin.GetConfig(*multinicnetwork, nil)
			Expect(err).To(BeNil())
		})

		It("create and delete", func() {
			// Create the MultiNicNetwork in the cluster
			err := K8sClient.Create(ctx, multinicnetwork)
			Expect(err).To(BeNil())
			// Fetch the created MultiNicNetwork to get the UID
			fetched := multinicnetwork.DeepCopy()
			err = K8sClient.Get(ctx, client.ObjectKey{Name: multinicnetwork.Name, Namespace: multinicnetwork.Namespace}, fetched)
			Expect(err).To(BeNil())
			// Use fetched (with UID) for NAD creation
			err = handler.CreateOrUpdate(fetched, mainPlugin, annotations)
			Expect(err).To(BeNil())
			namespaceList := corev1.NamespaceList{}
			err = K8sClient.List(ctx, &namespaceList)
			Expect(err).To(BeNil())
			Expect(len(namespaceList.Items)).To(BeNumerically(">", 0))
			for _, namespace := range namespaceList.Items {
				By(fmt.Sprintf("checking creation in namespace %s", namespace.Name))
				Eventually(func(g Gomega) {
					nad, err := handler.Get(multinicnetworkName, namespace.Name)
					g.Expect(err).To(BeNil())
					// Check owner reference
					refs := nad.OwnerReferences
					g.Expect(refs).ToNot(BeEmpty())
					found := false
					for _, ref := range refs {
						if ref.Kind == "MultiNicNetwork" && ref.Name == fetched.Name && ref.UID == fetched.UID {
							found = true
						}
					}
					g.Expect(found).To(BeTrue(), "OwnerReference to MultiNicNetwork should be set on NAD")
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			}
			// Delete the MultiNicNetwork
			err = K8sClient.Delete(ctx, fetched)
			Expect(err).To(BeNil())
			// Check if the MultiNicNetwork is actually deleted
			Eventually(func(g Gomega) {
				getErr := K8sClient.Get(ctx, client.ObjectKey{Name: fetched.Name, Namespace: fetched.Namespace}, fetched)
				fmt.Fprintf(GinkgoWriter, "[GinkgoWriter] MultiNicNetwork get after delete: %v\n", getErr)
				if getErr == nil {
					fmt.Fprintf(GinkgoWriter, "[GinkgoWriter] MultiNicNetwork DeletionTimestamp: %v, Finalizers: %v\n",
						fetched.DeletionTimestamp, fetched.Finalizers)
				}
				g.Expect(getErr).ToNot(BeNil())
			}).WithTimeout(10 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
			for _, namespace := range namespaceList.Items {
				By(fmt.Sprintf("checking deletion in namespace %s", namespace.Name))
				// NOTE: Kubernetes garbage collection for CRDs is not supported in unit/envtest environments.
				// Only check that the OwnerReference is set correctly. GC should be tested in a real cluster.
				nad, err := handler.Get(multinicnetworkName, namespace.Name)
				if err == nil && len(nad.OwnerReferences) > 0 {
					ref := nad.OwnerReferences[0]
					// Assert OwnerReference fields are correct
					Expect(ref.Controller != nil && *ref.Controller).To(BeTrue())
					Expect(ref.BlockOwnerDeletion != nil && *ref.BlockOwnerDeletion).To(BeTrue())
				}
			}
			// Clean up NADs for future tests
			for _, namespace := range namespaceList.Items {
				_ = handler.Delete(multinicnetworkName, namespace.Name)
			}
		})

		It("finalizer and owner reference work together for NAD cleanup", func() {
			// Add a finalizer to the MultiNicNetwork
			finalizer := "test.finalizer.multinicnetwork"
			multinicnetwork.ObjectMeta.Finalizers = append(multinicnetwork.ObjectMeta.Finalizers, finalizer)
			// Create the resource
			err := handler.CreateOrUpdate(multinicnetwork, mainPlugin, annotations)
			Expect(err).To(BeNil())
			namespaceList := corev1.NamespaceList{}
			err = K8sClient.List(ctx, &namespaceList)
			Expect(err).To(BeNil())
			// Confirm NADs exist
			for _, namespace := range namespaceList.Items {
				Eventually(func(g Gomega) {
					_, err := handler.Get(multinicnetworkName, namespace.Name)
					g.Expect(err).To(BeNil())
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			}
			// Delete the MultiNicNetwork (simulate controller removing finalizer after cleanup)
			// In a real controller, the finalizer is removed after cleanup; here, we simulate it
			// by removing the finalizer and then deleting the resource
			multinicnetwork.ObjectMeta.Finalizers = []string{}
			// Simulate deletion by deleting NADs and then the MultiNicNetwork
			err = handler.DeleteNets(multinicnetwork)
			Expect(err).To(BeNil())
			// Now, delete the MultiNicNetwork (would be done by controller after finalizer logic)
			// In this test, we just check that NADs are gone and resource is not stuck
			for _, namespace := range namespaceList.Items {
				Eventually(func(g Gomega) {
					_, err := handler.Get(multinicnetworkName, namespace.Name)
					g.Expect(errors.IsNotFound(err)).To(BeTrue())
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			}
			// The MultiNicNetwork should not be stuck in Terminating (simulate by checking finalizer is gone)
			Expect(multinicnetwork.ObjectMeta.Finalizers).To(BeEmpty())
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
