/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"bytes"
	daemon "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon"
	backend "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon/backend"
	da "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon/allocator"
	di "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon/iface"
	dr "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon/router"
	ds "github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon/selector"
	"k8s.io/client-go/kubernetes"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"context"
	"github.com/vishvananda/netlink"
	"path/filepath"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"log"
	//+kubebuilder:scaffold:imports
)

var dyn dynamic.Interface
var dc *discovery.DiscoveryClient
var requestRoute = dr.HostRoute {
	Subnet: "172.23.0.64/26",
	NextHop: "10.244.1.6",
	InterfaceName: "ens10",
}

const (
	HOST_NAME = "master0"
	DEF_NAME = "multi-nic-sample"
	DEVCLASS_DEF_NAME = "multi-nic-dev"
	POD_NAME = "sample-pod"
	POD_NAMESPACE = "default"

	KUBECONFIG_FILE="../../ipam/hpcg-kubeconfig"

	EXAMPLE_CRD_FOLDER = "../example/crd"
	EXAMPLE_RESOURCE_FOLDER = "../example/resource"

	EXAMPLE_CHECKPOINT = "../example/example-checkpoint"

	REQUEST_NUMBER = 2
)

var testEnv *envtest.Environment
var scheme = runtime.NewScheme()

var MASTER_INTERFACES []string = []string{"test-eth1", "test-eth2"}
var MASTER_IPS = []string{"10.244.0.1/24", "10.244.1.1/24"}
var MASTER_NETADDRESSES = []string{"10.244.0.0/24", "10.244.1.0/24"}
var MASTER_VENDORS = []string{"1d0f",""}
var MASTER_PRODUCTS = []string{"efa1",""}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultiNic CNI Suite")
}


func setTestLatestInterfaces() {
	latestInterfaceMap := map[string]di.InterfaceInfoType{}
	for index, master := range MASTER_INTERFACES {
		netAddress := MASTER_NETADDRESSES[index]
		vendor := MASTER_VENDORS[index]
		product := MASTER_PRODUCTS[index]
		iface := di.InterfaceInfoType{
			InterfaceName: master,
			NetAddress: netAddress,
			Vendor: vendor,
			Product: product,
		}
		latestInterfaceMap[master] = iface
	}
	di.LastestInterfaceMap = latestInterfaceMap
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	install.Install(scheme)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "example", "crd")},
		ErrorIfCRDPathMissing: true,
		Scheme: scheme,
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

	da.IppoolHandler = backend.NewIPPoolHandler(cfg)
	ds.MultinicnetHandler = backend.NewMultiNicNetworkHandler(cfg)
	ds.NetAttachDefHandler = backend.NewNetAttachDefHandler(cfg)
	ds.DeviceClassHandler = backend.NewDeviceClassHandler(cfg)
	da.K8sClientset, _ = kubernetes.NewForConfig(cfg)
	ds.K8sClientset, _ = kubernetes.NewForConfig(cfg)

	deployExamples(EXAMPLE_RESOURCE_FOLDER, false)
	addMasterInterfaces()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	deleteExamples(EXAMPLE_RESOURCE_FOLDER, true)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	deleteMasterInterfaces()
})

var _ = Describe("Test Route Add/Delete", func() {
	It("add route", func() {
		routeJson, err := json.Marshal(requestRoute)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", "/addroute", bytes.NewBuffer(routeJson))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.AddRoute)
		handler.ServeHTTP(res, req)
		body, _ := ioutil.ReadAll(res.Body)
		var response dr.RouteUpdateResponse
		json.Unmarshal(body, &response)
		log.Printf("TestAddRoute: %v", response)
		Expect(response.Success).To(Equal(true))
	})
	It("delete route", func() {
		routeJson, err := json.Marshal(requestRoute)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", "/deleteroute", bytes.NewBuffer(routeJson))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.DeleteRoute)
		handler.ServeHTTP(res, req)
		body, _ := ioutil.ReadAll(res.Body)
		var response dr.RouteUpdateResponse
		json.Unmarshal(body, &response)
		log.Printf("TestDeleteRoute: %v", response)
		Expect(response.Success).To(Equal(true))
	})
})

var _ = Describe("Test Get Interfaces", func() {
	It("get interfaces", func() {
		req, _ := http.NewRequest("GET", "/interface", nil)
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.GetInterface)
		handler.ServeHTTP(res, req)
		body, _ := ioutil.ReadAll(res.Body)
		var response []di.InterfaceInfoType
		json.Unmarshal(body, &response)
		log.Printf("TestUpdateInterface: %v", response)
		Expect(len(response)).Should(BeNumerically(">", 0))
	})
})

var _ = Describe("Test Allocation", func() {
	It("allocate" , func() {
		var response []da.IPResponse
	
		request := da.IPRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
	
		ipJson, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", daemon.ALLOCATE_PATH, bytes.NewBuffer(ipJson))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.Allocate)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response)).To(Equal(len(MASTER_INTERFACES)))
	})
	initIndexes := []int{1,2,3,8,13,18}
	allocations := genAllocation(initIndexes)

	It("find simple next available index" , func() {
		indexes := []int{1,2,3,8,13,18}
		nextIndex := da.FindAvailableIndex(indexes,0)
		Expect(nextIndex).To(Equal(4))
	})

	It("find next available index with exclude range over consecutive order" , func() {
		excludes := []da.ExcludeRange {
			da.ExcludeRange {
				MinIndex: 4,
				MaxIndex: 6,
			},
		}
		indexes := da.GenerateAllocateIndexes(allocations,20,excludes)
		Expect(indexes).To(Equal([]int{1,2,3,4,5,6,8,13,18}))
		nextIndex := da.FindAvailableIndex(indexes,0)
		Expect(nextIndex).To(Equal(7))
	})
	It("find next available index with exclude range over non-consecutive order" , func() {
		excludes := []da.ExcludeRange {
			da.ExcludeRange {
				MinIndex: 4,
				MaxIndex: 7,
			},
		}
	
		indexes := da.GenerateAllocateIndexes(allocations,20,excludes)
		Expect(indexes).To(Equal([]int{1,2,3,4,5,6,7,8,13,18}))
		nextIndex := da.FindAvailableIndex(indexes,0)
		Expect(nextIndex).To(Equal(9))
	})

	It("find next available index with exclude range over non-consecutive and then consecutive order" , func() {
		excludes := []da.ExcludeRange {
			da.ExcludeRange {
				MinIndex: 4,
				MaxIndex: 7,
			},
			da.ExcludeRange {
				MinIndex: 9,
				MaxIndex: 12,
			},
		}
	
		indexes := da.GenerateAllocateIndexes(allocations,20,excludes)
		Expect(indexes).To(Equal([]int{1,2,3,4,5,6,7,8,9,10,11,12,13,18}))
		nextIndex := da.FindAvailableIndex(indexes,0)
		Expect(nextIndex).To(Equal(14))
	})
})

var _ = Describe("Test NIC Select", func() {
	It("select all nic" , func() {
		setTestLatestInterfaces()
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		var response ds.NICSelectResponse
	
		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			MasterNetAddrs:   []string{},
		}
	
		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", daemon.NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(len(MASTER_NETADDRESSES)))
	})
	It("select one nic" , func() {
		setTestLatestInterfaces()
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		var response ds.NICSelectResponse
	
		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			NicSet:           ds.NicArgs{
				NumOfInterfaces: 1,
			},
		}
	
		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", daemon.NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
	})
	It("select nic by dev class" , func() {
		setTestLatestInterfaces()
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		var response ds.NICSelectResponse
	
		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEVCLASS_DEF_NAME,
			NicSet:           ds.NicArgs{
				DevClass: "highspeed",
			},
		}
	
		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", daemon.NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(daemon.SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
		Expect(response.Masters[0]).To(Equal(MASTER_INTERFACES[0]))
	})
})

func genAllocation(indexes []int) []backend.Allocation {
	var allocations []backend.Allocation
	for _, index := range indexes {
		allocations = append(allocations, backend.Allocation{Index: index})
	}
	return allocations
}


func deployExamples(folder string, ignoreErr bool) {
	files, err := ioutil.ReadDir(folder)
	Expect(err).NotTo(HaveOccurred())

	for _, file := range files {
		fileLocation := folder + "/" + file.Name()
		obj, dr := getDR(fileLocation, ignoreErr)
		if dr == nil {
			fmt.Println("No DR, deploy")
			continue
		}
		_, err  = dr.Create(context.TODO(), obj, metav1.CreateOptions{})
		fmt.Printf("Deploy %s (%v): %v\n", fileLocation, ignoreErr, err)
		if !ignoreErr {
			Expect(err).NotTo(HaveOccurred())
		}	
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

func deleteExamples(folder string, ignoreErr bool) {
	files, err := ioutil.ReadDir(folder)
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

func deleteMasterInterfaces() {
	// Add master
	for _, master := range MASTER_INTERFACES {
		masterLink, err := netlink.LinkByName(master)
		Expect(err).NotTo(HaveOccurred())
		netlink.LinkSetDown(masterLink)
		netlink.LinkDel(masterLink)
	}
}