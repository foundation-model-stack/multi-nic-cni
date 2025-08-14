/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package allocator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var dyn dynamic.Interface
var dc *discovery.DiscoveryClient
var testEnv *envtest.Environment
var scheme = runtime.NewScheme()

func TestAllocator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Allocator Test Suite")
}

var _ = BeforeSuite(func() {
	os.Setenv("TEST_MODE", "true")
	install.Install(scheme)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "example", "crd")},
		ErrorIfCRDPathMissing: true,
		Scheme:                scheme,
	}

	err := apiextensionsv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	dyn, err = dynamic.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	dc, err = discovery.NewDiscoveryClientForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	K8sClientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	IppoolHandler = backend.NewIPPoolHandler(cfg)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
