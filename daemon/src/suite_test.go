/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ds "github.com/foundation-model-stack/multi-nic-cni/daemon/selector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var dyn dynamic.Interface
var dc *discovery.DiscoveryClient
var testEnv *envtest.Environment
var scheme = runtime.NewScheme()
var targetPod *v1.Pod

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-NIC Daemon Suite")
}

var _ = BeforeSuite(func() {
	os.Setenv("TEST_MODE", "true")

	// this env should be set by config.multinic when creating the daemonset
	initHostName() // before setting NODENAME_ENV
	os.Setenv(NODENAME_ENV, FULL_HOST_NAME)
	initHostName() // after setting
	Expect(hostName).To(Equal(FULL_HOST_NAME))
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	install.Install(scheme)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "example", "crd")},
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

	initHandlers(cfg)
	Expect(hostName).NotTo(BeEmpty())

	deployExamples(EXAMPLE_RESOURCE_FOLDER, false)
	addMasterInterfaces()
	replacePodUID(ds.K8sClientset)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	deleteExamples(EXAMPLE_RESOURCE_FOLDER, true)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	deleteMasterInterfaces()
})

func deployExamples(folder string, ignoreErr bool) {
	files, err := os.ReadDir(folder)
	Expect(err).NotTo(HaveOccurred())
	ctx := context.Background()

	for _, file := range files {
		fileLocation := folder + "/" + file.Name()
		obj, dr := getDR(fileLocation, ignoreErr)
		if dr == nil {
			fmt.Println("No DR, deploy")
			continue
		}
		_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
		fmt.Printf("Deploy %s (%v): %v\n", fileLocation, ignoreErr, err)
		if !ignoreErr {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func addMasterInterfaces() {
	// Add master
	for index, master := range MASTER_INTERFACES {
		linkAttrs := netlink.LinkAttrs{
			Name: master,
		}
		err := netlink.LinkAdd(&netlink.Dummy{
			linkAttrs,
		})
		Expect(err).NotTo(HaveOccurred())
		masterLink, err := netlink.LinkByName(master)
		Expect(err).NotTo(HaveOccurred())

		addr, _ := netlink.ParseAddr(MASTER_IPS[index])
		netlink.AddrAdd(masterLink, addr)
		Expect(err).NotTo(HaveOccurred())
		err = netlink.LinkSetUp(masterLink)
		Expect(err).NotTo(HaveOccurred())
	}
}

func replacePodUID(clientset *kubernetes.Clientset) {
	var err error
	// Get PodUID
	targetPod, err = clientset.CoreV1().Pods(POD_NAMESPACE).Get(context.TODO(), POD_NAME, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	// Replace checkpoint file

	// Read the file content
	content, err := os.ReadFile(EXAMPLE_CHECKPOINT)
	Expect(err).NotTo(HaveOccurred())

	// Perform text replacements
	modifiedContent := strings.ReplaceAll(string(content), TO_REPLACE_POD_UID, string(targetPod.UID))

	// Write the modified content back to the file
	err = os.WriteFile(EXAMPLE_CHECKPOINT, []byte(modifiedContent), 0644)
	Expect(err).NotTo(HaveOccurred())
}

func deleteExamples(folder string, ignoreErr bool) {
	files, err := os.ReadDir(folder)
	Expect(err).NotTo(HaveOccurred())

	for _, file := range files {
		fileLocation := folder + "/" + file.Name()
		fmt.Printf("Delete %s (%v)\n", fileLocation, ignoreErr)
		obj, dr := getDR(fileLocation, ignoreErr)
		if dr == nil {
			fmt.Println("No DR, delete")
			continue
		}
		err = dr.Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{})
		if !ignoreErr {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

func deleteMasterInterfaces() {
	// Add master
	for _, master := range MASTER_INTERFACES {
		masterLink, err := netlink.LinkByName(master)
		Expect(err).NotTo(HaveOccurred())
		netlink.LinkSetDown(masterLink)
		netlink.LinkDel(masterLink)
	}
}

func getDR(fileName string, ignoreErr bool) (*unstructured.Unstructured, dynamic.ResourceInterface) {
	bodyBytes, err := ioutil.ReadFile(fileName)
	if ignoreErr && err != nil {
		return nil, nil
	}
	Expect(err).NotTo(HaveOccurred())
	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode(bodyBytes, nil, obj)
	if ignoreErr && err != nil {
		return nil, nil
	}
	Expect(err).NotTo(HaveOccurred())
	return obj, getResourceInterface(gvk, obj.GetNamespace(), ignoreErr)
}

func getResourceInterface(gvk *schema.GroupVersionKind, ns string, ignoreErr bool) dynamic.ResourceInterface {
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if !ignoreErr {
		Expect(err).NotTo(HaveOccurred())
	}
	if err != nil {
		return nil
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = dyn.Resource(mapping.Resource)
	}
	return dr
}
