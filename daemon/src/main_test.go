/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	da "github.com/foundation-model-stack/multi-nic-cni/daemon/allocator"
	backend "github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
	di "github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
	dr "github.com/foundation-model-stack/multi-nic-cni/daemon/router"
	ds "github.com/foundation-model-stack/multi-nic-cni/daemon/selector"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"context"
	"github.com/vishvananda/netlink"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strings"

	"os"
	//+kubebuilder:scaffold:imports
)

var dyn dynamic.Interface
var dc *discovery.DiscoveryClient
var requestRoute = dr.HostRoute{
	Subnet:        "172.23.0.64/26",
	NextHop:       "10.244.1.6",
	InterfaceName: "ens10",
}

var requestL3Config = dr.L3ConfigRequest{
	Name:   "l3net",
	Subnet: "192.168.0.0/16",
	Routes: []dr.HostRoute{
		requestRoute,
	},
	Force: false,
}

var requestL3ConfigForceDelete = dr.L3ConfigRequest{
	Name:   "l3net",
	Subnet: "192.168.0.0/16",
	Routes: []dr.HostRoute{
		requestRoute,
	},
	Force: true,
}

const (
	HOST_NAME          = "master0"
	FULL_HOST_NAME     = HOST_NAME + ".local"
	DEF_NAME           = "multi-nic-sample"
	DEVCLASS_DEF_NAME  = "multi-nic-dev"
	TOPOLOGY_DEF_NAME  = "multi-nic-topology"
	POD_NAME           = "sample-pod"
	POD_NAMESPACE      = "default"
	TO_REPLACE_POD_UID = "18134649-65e3-4bf8-ba89-e3d133fe9e53"

	KUBECONFIG_FILE = "../../ipam/hpcg-kubeconfig"

	EXAMPLE_CRD_FOLDER      = "../example/crd"
	EXAMPLE_RESOURCE_FOLDER = "../example/resource"

	EXAMPLE_CHECKPOINT = "../example/example-checkpoint"
	EXAMPLE_TOPOLOGY   = "../example/example-topology.xml"

	REQUEST_NUMBER = 2
)

var testEnv *envtest.Environment
var scheme = runtime.NewScheme()
var targetPod *v1.Pod

var MASTER_INTERFACES []string = []string{"test-eth1", "test-eth2"}
var MASTER_IPS = []string{"10.244.0.1/24", "10.244.1.1/24"}
var MASTER_NETADDRESSES = []string{"10.244.0.0/24", "10.244.1.0/24"}
var MASTER_PCIADDRESS = []string{"0000:08:00.0", "0000:0c:00.0"}
var MASTER_VENDORS = []string{"1d0f", ""}
var MASTER_PRODUCTS = []string{"efa1", ""}

var GPU_BUS_MAP map[string]string = map[string]string{
	"GPU-581b17ed-1c48-9b8c-6a9b-e2e6f99500dc": "0000:0c:05.0",
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultiNic CNI Suite")
}

func setTestLatestInterfaces() {
	for index, master := range MASTER_INTERFACES {
		netAddress := MASTER_NETADDRESSES[index]
		vendor := MASTER_VENDORS[index]
		product := MASTER_PRODUCTS[index]
		pciAddress := MASTER_PCIADDRESS[index]
		iface := di.InterfaceInfoType{
			InterfaceName: master,
			NetAddress:    netAddress,
			Vendor:        vendor,
			Product:       product,
			PciAddress:    pciAddress,
		}
		di.SetInterfaceInfoCache(master, iface)
	}
	interfaceMap := di.GetInterfaceInfoCache()
	Expect(len(interfaceMap)).To(Equal(2))
}

func replacePodUID(clientset *kubernetes.Clientset) {
	var err error
	// Get PodUID
	targetPod, err = clientset.CoreV1().Pods(POD_NAMESPACE).Get(context.TODO(), POD_NAME, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	// Replace checkpoint file

	// Read the file content
	content, err := ioutil.ReadFile(EXAMPLE_CHECKPOINT)
	Expect(err).NotTo(HaveOccurred())

	// Perform text replacements
	modifiedContent := strings.ReplaceAll(string(content), TO_REPLACE_POD_UID, string(targetPod.UID))

	// Write the modified content back to the file
	err = ioutil.WriteFile(EXAMPLE_CHECKPOINT, []byte(modifiedContent), 0644)
	Expect(err).NotTo(HaveOccurred())
}

var _ = BeforeSuite(func() {

	// this env should be set by config.multinic when creating the daemonset
	os.Setenv(NODENAME_ENV, FULL_HOST_NAME)
	initHostName()
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

	da.IppoolHandler = backend.NewIPPoolHandler(cfg)
	ds.MultinicnetHandler = backend.NewMultiNicNetworkHandler(cfg)
	ds.NetAttachDefHandler = backend.NewNetAttachDefHandler(cfg)
	ds.DeviceClassHandler = backend.NewDeviceClassHandler(cfg)
	da.K8sClientset, _ = kubernetes.NewForConfig(cfg)
	ds.K8sClientset, _ = kubernetes.NewForConfig(cfg)

	deployExamples(EXAMPLE_RESOURCE_FOLDER, false)
	addMasterInterfaces()
	replacePodUID(ds.K8sClientset)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	deleteExamples(EXAMPLE_RESOURCE_FOLDER, true)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	deleteMasterInterfaces()
})

var _ = Describe("Test L3Config Add/Delete", func() {
	It("apply/delete l3config", func() {
		l3config, err := json.Marshal(requestL3Config)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", ADD_L3CONFIG_PATH, bytes.NewBuffer(l3config))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(ApplyL3Config)
		handler.ServeHTTP(res, req)
		body, _ := ioutil.ReadAll(res.Body)
		var response dr.RouteUpdateResponse
		json.Unmarshal(body, &response)
		log.Printf("TestApplyL3Config: %v", response)
		Expect(response.Success).To(Equal(true))

		l3config, err = json.Marshal(requestL3ConfigForceDelete)
		Expect(err).NotTo(HaveOccurred())
		req, err = http.NewRequest("PUT", ADD_L3CONFIG_PATH, bytes.NewBuffer(l3config))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res = httptest.NewRecorder()
		handler = http.HandlerFunc(ApplyL3Config)
		handler.ServeHTTP(res, req)
		body, _ = ioutil.ReadAll(res.Body)
		var responseWithForce dr.RouteUpdateResponse
		json.Unmarshal(body, &responseWithForce)
		log.Printf("TestApplyL3ConfigForceDelete: %v", responseWithForce)
		Expect(responseWithForce.Success).To(Equal(true))

		l3config, err = json.Marshal(requestL3Config)
		Expect(err).NotTo(HaveOccurred())
		req, err = http.NewRequest("PUT", DELETE_L3CONFIG_PATH, bytes.NewBuffer(l3config))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res = httptest.NewRecorder()
		handler = http.HandlerFunc(DeleteL3Config)
		handler.ServeHTTP(res, req)
		body, _ = ioutil.ReadAll(res.Body)
		json.Unmarshal(body, &response)
		log.Printf("TestDeleteL3Config: %v", response)
		Expect(response.Success).To(Equal(true))
	})
})

var _ = Describe("Test Get Interfaces", func() {
	It("get interfaces", func() {
		req, _ := http.NewRequest("GET", INTERFACE_PATH, nil)
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(GetInterface)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		var response []di.InterfaceInfoType
		json.Unmarshal(body, &response)
		log.Printf("TestUpdateInterface: %v", response)
	})
})

var _ = Describe("Test Allocation", func() {
	It("normaly allocate", func() {
		request := da.IPRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		ipJson, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		// allocate
		allocateHandler := http.HandlerFunc(Allocate)
		response := MakeIPRequest(ipJson, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(ipJson, DEALLOCATE_PATH, deallocateHandler, false)
	})

	It("anomaly allocate from begining", func() {
		request := da.IPRequest{
			PodName:          "anormal1_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		ipJson, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		// allocate
		allocateHandler := http.HandlerFunc(Allocate)
		response1 := MakeIPRequest(ipJson, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response1)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(ipJson, DEALLOCATE_PATH, deallocateHandler, false)
		//allocate again
		response2 := MakeIPRequest(ipJson, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response2)).To(Equal(len(MASTER_INTERFACES)))
		log.Printf("Response 1: %v\n", response1)
		log.Printf("Response 2: %v\n", response2)
		//response2 should not be equal to response 1 due to anomaly consecutive allocation
		for index, resp1 := range response1 {
			Expect(response2[index].IPAddress).NotTo(Equal(resp1.IPAddress))
		}
		//deallocate
		MakeIPRequest(ipJson, DEALLOCATE_PATH, deallocateHandler, false)
	})

	It("anomaly allocate after some allocations", func() {
		request1 := da.IPRequest{
			PodName:          "normal_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		ipJson1, err := json.Marshal(request1)
		Expect(err).NotTo(HaveOccurred())
		// allocate some
		allocateHandler := http.HandlerFunc(Allocate)
		response := MakeIPRequest(ipJson1, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response)).To(Equal(len(MASTER_INTERFACES)))

		request2 := da.IPRequest{
			PodName:          "anormal2_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		ipJson2, err := json.Marshal(request2)
		Expect(err).NotTo(HaveOccurred())
		// allocate
		allocateHandler = http.HandlerFunc(Allocate)
		response1 := MakeIPRequest(ipJson2, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response1)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(ipJson2, DEALLOCATE_PATH, deallocateHandler, false)
		//allocate again
		response2 := MakeIPRequest(ipJson2, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response2)).To(Equal(len(MASTER_INTERFACES)))
		log.Printf("Response 1: %v\n", response1)
		log.Printf("Response 2: %v\n", response2)
		//response2 should not be equal to response 1 due to anomaly consecutive allocation
		for index, resp1 := range response1 {
			Expect(response2[index].IPAddress).NotTo(Equal(resp1.IPAddress))
		}
		//deallocate
		MakeIPRequest(ipJson1, DEALLOCATE_PATH, deallocateHandler, false)
		//deallocate
		MakeIPRequest(ipJson2, DEALLOCATE_PATH, deallocateHandler, false)
	})

})

var _ = Describe("Test NIC Select", func() {
	It("select all nic", func() {
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
		req, err := http.NewRequest("PUT", NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(len(MASTER_NETADDRESSES)))
	})
	It("select one nic", func() {
		setTestLatestInterfaces()
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		var response ds.NICSelectResponse

		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			NicSet: ds.NicArgs{
				NumOfInterfaces: 1,
			},
		}

		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
	})
	It("select nic by dev class", func() {
		setTestLatestInterfaces()
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		var response ds.NICSelectResponse

		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEVCLASS_DEF_NAME,
			NicSet: ds.NicArgs{
				DevClass: "highspeed",
			},
		}

		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
		Expect(response.Masters[0]).To(Equal(MASTER_INTERFACES[0]))
	})
	It("Init NumaAwareSelector from topology file", func() {
		selector := ds.InitNumaAwareSelector(EXAMPLE_TOPOLOGY, map[string]string{})
		Expect(len(selector.NcclTopolgy.CPUs)).To(Equal(2))
		Expect(len(selector.NumaMap)).To(BeNumerically(">", 0))
	})
	It("Init NumaAwareSelector from sysfs", func() {
		selector := ds.InitNumaAwareSelector("", map[string]string{})
		Expect(len(selector.NcclTopolgy.CPUs)).To(Equal(0))
		Expect(len(selector.NumaMap)).To(BeNumerically(">", 0))
	})
	It("Get resource map", func() {
		resourceMap, err := di.GetPodResourceMap(targetPod)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(resourceMap)).To(BeNumerically(">", 0))
		_, ok := resourceMap[ds.GPUResourceName]
		Expect(ok).To(Equal(true))
	})
	It("select nic by NumaAwareSelector (topology)", func() {
		setTestLatestInterfaces()
		ds.TopologyFilePath = EXAMPLE_TOPOLOGY
		ds.GPUIdBusIdMap = GPU_BUS_MAP
		di.CheckPointfile = EXAMPLE_CHECKPOINT
		ds.NumaAwareSelectorInstance = ds.InitNumaAwareSelector(ds.TopologyFilePath, ds.GPUIdBusIdMap)
		var response ds.NICSelectResponse

		request := ds.NICSelectRequest{
			PodName:          POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: TOPOLOGY_DEF_NAME,
			MasterNetAddrs:   MASTER_NETADDRESSES, // fixed order before sort to avoid random pass
			NicSet: ds.NicArgs{
				NumOfInterfaces: 1,
			},
		}
		jsonObj, err := json.Marshal(request)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest("PUT", NIC_SELECT_PATH, bytes.NewBuffer(jsonObj))
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(SelectNic)
		handler.ServeHTTP(res, req)
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
		// must select nic in numa 1
		Expect(response.Masters[0]).To(Equal(MASTER_INTERFACES[1]))
	})
})

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
		_, err = dr.Create(context.TODO(), obj, metav1.CreateOptions{})
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

func MakeIPRequest(ipJson []byte, path string, handler http.HandlerFunc, shouldResponse bool) []da.IPResponse {
	var response []da.IPResponse
	req, err := http.NewRequest("PUT", path, bytes.NewBuffer(ipJson))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	body, err := ioutil.ReadAll(res.Body)
	Expect(err).NotTo(HaveOccurred())
	if shouldResponse {
		err = json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
	}
	return response
}
