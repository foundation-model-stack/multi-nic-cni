/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	da "github.com/foundation-model-stack/multi-nic-cni/daemon/allocator"
	di "github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
	dr "github.com/foundation-model-stack/multi-nic-cni/daemon/router"
	ds "github.com/foundation-model-stack/multi-nic-cni/daemon/selector"
	"github.com/vishvananda/netlink"

	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

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

var MASTER_INTERFACES []string = []string{"test-eth1", "test-eth2"}
var MASTER_IPS = []string{"10.244.0.1/24", "10.244.1.1/24"}
var MASTER_NETADDRESSES = []string{"10.244.0.0/24", "10.244.1.0/24"}
var MASTER_PCIADDRESS = []string{"0000:08:00.0", "0000:0c:00.0"}
var MASTER_VENDORS = []string{"1d0f", ""}
var MASTER_PRODUCTS = []string{"efa1", ""}

var GPU_BUS_MAP map[string]string = map[string]string{
	"GPU-581b17ed-1c48-9b8c-6a9b-e2e6f99500dc": "0000:0c:05.0",
}

var _ = Describe("Join", func() {
	It("empty join", func() {
		requestIpamInfo := IPAMInfo{
			HIFList: []di.InterfaceInfoType{},
		}
		body := MakePutRequest(requestIpamInfo, JOIN_PATH, http.HandlerFunc(Join))
		Expect(strings.TrimSpace(string(body))).To(BeEquivalentTo(`""`))
	})

	It("empty greet ack", func() {
		host := ""
		body := MakePutRequest(host, GREET_PATH, http.HandlerFunc(GreetAck))
		Expect(strings.TrimSpace(string(body))).To(BeEquivalentTo(`""`))
	})
})

var _ = Describe("Test L3Config Add/Delete", func() {
	It("apply/delete l3config", func() {
		body := MakePutRequest(requestL3Config, ADD_L3CONFIG_PATH, http.HandlerFunc(ApplyL3Config))
		var response dr.RouteUpdateResponse
		json.Unmarshal(body, &response)
		log.Printf("TestApplyL3Config: %v", response)
		Expect(response.Success).To(Equal(true))

		body = MakePutRequest(requestL3ConfigForceDelete, ADD_L3CONFIG_PATH, http.HandlerFunc(ApplyL3Config))
		var responseWithForce dr.RouteUpdateResponse
		json.Unmarshal(body, &responseWithForce)
		log.Printf("TestApplyL3ConfigForceDelete: %v", responseWithForce)
		Expect(responseWithForce.Success).To(Equal(true))

		body = MakePutRequest(requestL3Config, DELETE_L3CONFIG_PATH, http.HandlerFunc(DeleteL3Config))
		json.Unmarshal(body, &response)
		log.Printf("TestDeleteL3Config: %v", response)
		Expect(response.Success).To(Equal(true))
	})
})

var _ = Describe("Test Route Add/Delete", func() {
	It("add/delete route", func() {
		// must use valid interface name
		r := dr.HostRoute{
			Subnet:        "192.168.0.100/32",
			NextHop:       "0.0.0.0",
			InterfaceName: getValidIface(),
		}
		By("Adding route")
		body := MakePutRequest(r, ADD_ROUTE_PATH, http.HandlerFunc(AddRoute))
		var response dr.RouteUpdateResponse
		json.Unmarshal(body, &response)
		Expect(response.Success).To(Equal(true))

		By("Deleting route")
		body = MakePutRequest(r, DELETE_ROUTE_PATH, http.HandlerFunc(DeleteRoute))
		json.Unmarshal(body, &response)
		log.Printf("TestDelete: %v", response)
		Expect(response.Success).To(Equal(true))
	})
})

var _ = Describe("Test Get Interfaces", func() {
	It("get interfaces", func() {
		req, _ := http.NewRequest("GET", INTERFACE_PATH, nil)
		res := httptest.NewRecorder()
		handler := http.HandlerFunc(GetInterface)
		handler.ServeHTTP(res, req)
		body, err := io.ReadAll(res.Body)
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
		// allocate
		allocateHandler := http.HandlerFunc(Allocate)
		response := MakeIPRequest(request, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(request, DEALLOCATE_PATH, deallocateHandler, false)
	})

	It("anomaly allocate from begining", func() {
		request := da.IPRequest{
			PodName:          "anormal1_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		// allocate
		allocateHandler := http.HandlerFunc(Allocate)
		response1 := MakeIPRequest(request, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response1)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(request, DEALLOCATE_PATH, deallocateHandler, false)
		//allocate again
		response2 := MakeIPRequest(request, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response2)).To(Equal(len(MASTER_INTERFACES)))
		log.Printf("Response 1: %v\n", response1)
		log.Printf("Response 2: %v\n", response2)
		//response2 should not be equal to response 1 due to anomaly consecutive allocation
		for index, resp1 := range response1 {
			Expect(response2[index].IPAddress).NotTo(Equal(resp1.IPAddress))
		}
		//deallocate
		MakeIPRequest(request, DEALLOCATE_PATH, deallocateHandler, false)
	})

	It("anomaly allocate after some allocations", func() {
		request1 := da.IPRequest{
			PodName:          "normal_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		// allocate some
		allocateHandler := http.HandlerFunc(Allocate)
		response := MakeIPRequest(request1, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response)).To(Equal(len(MASTER_INTERFACES)))

		request2 := da.IPRequest{
			PodName:          "anormal2_" + POD_NAME,
			PodNamespace:     POD_NAMESPACE,
			HostName:         HOST_NAME,
			NetAttachDefName: DEF_NAME,
			InterfaceNames:   MASTER_INTERFACES,
		}
		// allocate
		allocateHandler = http.HandlerFunc(Allocate)
		response1 := MakeIPRequest(request2, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response1)).To(Equal(len(MASTER_INTERFACES)))
		//deallocate
		deallocateHandler := http.HandlerFunc(Deallocate)
		MakeIPRequest(request2, DEALLOCATE_PATH, deallocateHandler, false)
		//allocate again
		response2 := MakeIPRequest(request2, ALLOCATE_PATH, allocateHandler, true)
		Expect(len(response2)).To(Equal(len(MASTER_INTERFACES)))
		log.Printf("Response 1: %v\n", response1)
		log.Printf("Response 2: %v\n", response2)
		//response2 should not be equal to response 1 due to anomaly consecutive allocation
		for index, resp1 := range response1 {
			Expect(response2[index].IPAddress).NotTo(Equal(resp1.IPAddress))
		}
		//deallocate
		MakeIPRequest(request1, DEALLOCATE_PATH, deallocateHandler, false)
		//deallocate
		MakeIPRequest(request2, DEALLOCATE_PATH, deallocateHandler, false)
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
		body := MakePutRequest(request, NIC_SELECT_PATH, http.HandlerFunc(SelectNic))
		err := json.Unmarshal(body, &response)
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
		body := MakePutRequest(request, NIC_SELECT_PATH, http.HandlerFunc(SelectNic))
		err := json.Unmarshal(body, &response)
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
		body := MakePutRequest(request, NIC_SELECT_PATH, http.HandlerFunc(SelectNic))
		err := json.Unmarshal(body, &response)
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
		body := MakePutRequest(request, NIC_SELECT_PATH, http.HandlerFunc(SelectNic))
		err := json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(response.Masters)).To(Equal(1))
		// must select nic in numa 1
		Expect(response.Masters[0]).To(Equal(MASTER_INTERFACES[1]))
	})
})

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

func MakePutRequest(obj any, path string, handler http.HandlerFunc) []byte {
	encoded, err := json.Marshal(obj)
	Expect(err).NotTo(HaveOccurred())
	req, err := http.NewRequest("PUT", path, bytes.NewBuffer(encoded))
	Expect(err).NotTo(HaveOccurred())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	body, err := io.ReadAll(res.Body)
	Expect(err).NotTo(HaveOccurred())
	return body
}

func MakeIPRequest(requestIP da.IPRequest, path string, handler http.HandlerFunc, shouldResponse bool) []da.IPResponse {
	var response []da.IPResponse
	body := MakePutRequest(requestIP, path, handler)
	if shouldResponse {
		err := json.Unmarshal(body, &response)
		Expect(err).NotTo(HaveOccurred())
	}
	return response
}

func getValidIface() string {
	links, err := netlink.LinkList()
	Expect(err).NotTo(HaveOccurred())
	notFound := true
	for _, link := range links {
		devName := link.Attrs().Name
		if link.Type() == "device" {
			return devName
		}
	}
	Expect(notFound).To(BeFalse())
	return ""
}
