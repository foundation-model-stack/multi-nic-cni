/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

// note:
// no route to host error: check if veth interface (172.168.17.2) is properly removed

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/containernetworking/cni/pkg/types/040"
	"github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"

	"github.com/containernetworking/plugins/pkg/ip"

	"github.com/gorilla/mux"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type IPRequest struct {
	PodName          string   `json:"pod"`
	PodNamespace     string   `json:"namespace"`
	HostName         string   `json:"host"`
	NetAttachDefName string   `json:"def"`
	InterfaceNames   []string `json:"masters"`
}

type IPResponse struct {
	InterfaceName string `json:"interface"`
	IPAddress     string `json:"ip"`
	VLANBlockSize string `json:"block"`
}

// type NICSelectRequest struct {
// 	PodName          string   `json:"pod"`
// 	PodNamespace     string   `json:"namespace"`
// 	HostName         string   `json:"host"`
// 	NetAttachDefName string   `json:"def"`
// 	NicSet           NicArgs  `json:"args"`
// }

// type NicArgs struct {
// 	NumOfInterfaces int `json:"nics,omitempty"`
// 	InterfaceNames  []string `json:"masters,omitempty"`
// }

//	type NICSelectResponse struct {
//		DeviceIDs   []string `json:"deviceIDs"`
//	}
var POOL_MASTER_NAMES = []string{"eth0", "eth1", "eth2"}
var POOL_NETWORK_ADDRESSES = []string{"10.244.0.0/24", "10.244.1.0/24", "10.244.2.0/24"}
var POOL_IP_ADDRESSES = []string{"10.244.0.120/24", "10.244.1.5/24", "10.244.2.1/24"}

var MASTER_NAMES = []string{"eth0", "eth1"}
var NETWORK_ADDRESSES = []string{"10.244.0.0/24", "10.244.1.0/24"}
var IP_ADDRESSES = []string{"10.244.0.120/24", "10.244.1.5/24"}
var NEXT_ADDRESSES = []string{"192.168.0.65", "192.168.1.66"}
var BRIDGE_CONTAINER_IP = "172.168.17.1"
var BRIDGE_HOST_IP = "172.168.17.2"
var daemonPort int

const (
	ALLOCATE_PATH   = "allocate"
	DEALLOCATE_PATH = "deallocate"
)

func buildOneConfig(cniVersion string, orig *NetConf, prevResult types.Result) (*NetConf, error) {
	confBytes, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})
	err = json.Unmarshal(confBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal existing network bytes: %s", err)
	}

	inject := map[string]interface{}{
		"name":       orig.Name,
		"cniVersion": orig.CNIVersion,
	}
	// Add previous plugin result
	if prevResult != nil && testutils.SpecVersionHasChaining(cniVersion) {
		inject["prevResult"] = prevResult
	}

	for key, value := range inject {
		config[key] = value
	}

	newBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	conf := &NetConf{}
	if err := json.Unmarshal(newBytes, &conf); err != nil {
		return nil, fmt.Errorf("error parsing configuration: %s", err)
	}

	return conf, nil

}

func multinicAddCheckDelTest(conf, masterName string, originalNS, targetNS ns.NetNS) {
	log.Printf("multinicAddCheckDelTest")
	log.Printf("%s", conf)
	// Unmarshal to pull out CNI spec version
	rawConfig := make(map[string]interface{})
	err := json.Unmarshal([]byte(conf), &rawConfig)
	Expect(err).NotTo(HaveOccurred())
	cniVersion := rawConfig["cniVersion"].(string)

	args := &skel.CmdArgs{
		ContainerID: "dummy",
		Netns:       targetNS.Path(),
		IfName:      "net1",
		StdinData:   []byte(conf),
	}

	var result types.Result
	var macAddress string
	err = originalNS.Do(func(ns.NetNS) error {
		defer GinkgoRecover()
		result, _, err = testutils.CmdAddWithArgs(args, func() error {
			return cmdAdd(args)
		})
		fmt.Println(result)
		Expect(err).NotTo(HaveOccurred())
		t := newTesterByVersion(cniVersion)
		macAddress = t.verifyResult(result, args.IfName)
		return nil
	})
	Expect(err).NotTo(HaveOccurred())

	// Make sure link exists in the target namespace
	err = targetNS.Do(func(ns.NetNS) error {
		defer GinkgoRecover()

		link, err := netlink.LinkByName(args.IfName + "-0")
		Expect(err).NotTo(HaveOccurred())
		Expect(link.Attrs().Name).To(Equal(args.IfName + "-0"))

		if macAddress != "" {
			hwaddr, err := net.ParseMAC(macAddress)
			Expect(err).NotTo(HaveOccurred())
			Expect(link.Attrs().HardwareAddr).To(Equal(hwaddr))
		}

		addrs, err := netlink.AddrList(link, syscall.AF_INET)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(addrs)).To(Equal(1))
		return nil
	})
	Expect(err).NotTo(HaveOccurred())

	n := &NetConf{}
	err = json.Unmarshal([]byte(conf), &n)
	Expect(err).NotTo(HaveOccurred())

	// build chained/cached config for DEL
	newConf, err := buildOneConfig(cniVersion, n, result)
	Expect(err).NotTo(HaveOccurred())
	confBytes, err := json.Marshal(newConf)
	Expect(err).NotTo(HaveOccurred())

	args.StdinData = confBytes
	GinkgoT().Logf(string(confBytes))

	if testutils.SpecVersionHasCHECK(cniVersion) {
		// CNI Check in the target namespace
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			return testutils.CmdCheckWithArgs(args, func() error {
				return cmdCheck(args)
			})
		})
		Expect(err).NotTo(HaveOccurred())
	}

	err = originalNS.Do(func(ns.NetNS) error {
		defer GinkgoRecover()

		err = testutils.CmdDelWithArgs(args, func() error {
			return cmdDel(args)
		})
		Expect(err).NotTo(HaveOccurred())
		return nil
	})
	Expect(err).NotTo(HaveOccurred())

	// Make sure ipvlan link has been deleted
	err = targetNS.Do(func(ns.NetNS) error {
		defer GinkgoRecover()

		link, err := netlink.LinkByName(args.IfName)
		Expect(err).To(HaveOccurred())
		Expect(link).To(BeNil())
		return nil
	})
	Expect(err).NotTo(HaveOccurred())
}

type tester interface {
	// verifyResult minimally verifies the Result and returns the interface's MAC address
	verifyResult(result types.Result, name string) string
}

type testerBase struct{}

type testerV10x testerBase
type testerV04x testerBase
type testerV02x testerBase

func newTesterByVersion(version string) tester {
	switch {
	case strings.HasPrefix(version, "1.0."):
		return &testerV10x{}
	case strings.HasPrefix(version, "0.4.") || strings.HasPrefix(version, "0.3."):
		return &testerV04x{}
	case strings.HasPrefix(version, "0.1.") || strings.HasPrefix(version, "0.2."):
		return &testerV02x{}
	}
	Fail(fmt.Sprintf("unsupported config version %s", version))
	return nil
}

// verifyResult minimally verifies the Result and returns the interface's MAC address
func (t *testerV10x) verifyResult(result types.Result, name string) string {
	r, err := types100.GetResult(result)
	Expect(err).NotTo(HaveOccurred())

	Expect(len(r.Interfaces)).To(Equal(len(MASTER_NAMES)))
	Expect(r.Interfaces[0].Name).To(Equal(name))
	Expect(len(r.IPs)).To(Equal(len(MASTER_NAMES)))

	return r.Interfaces[0].Mac
}

// verifyResult minimally verifies the Result and returns the interface's MAC address
func (t *testerV04x) verifyResult(result types.Result, name string) string {
	r, err := types040.GetResult(result)
	Expect(err).NotTo(HaveOccurred())

	Expect(len(r.Interfaces)).To(Equal(len(MASTER_NAMES)))
	Expect(r.Interfaces[0].Name).To(Equal(name + "-0"))
	Expect(len(r.IPs)).To(Equal(len(MASTER_NAMES)))

	return r.Interfaces[0].Mac
}

// verifyResult minimally verifies the Result and returns the interface's MAC address
func (t *testerV02x) verifyResult(result types.Result, name string) string {
	r, err := types020.GetResult(result)
	Expect(err).NotTo(HaveOccurred())

	Expect(r.IP4.IP).NotTo(BeNil())
	Expect(r.IP4.IP.IP).NotTo(BeNil())
	Expect(r.IP6).To(BeNil())

	// 0.2 and earlier don't return MAC address
	return ""
}

var _ = Describe("Operations", func() {
	var originalNS, targetNS ns.NetNS
	var dataDir string

	httpServerExitDone := &sync.WaitGroup{}
	var srv *http.Server
	BeforeEach(func() {
		// Create a new original NetNS so we don't modify the host
		var err error

		hostNS, err := ns.GetCurrentNS()
		Expect(err).NotTo(HaveOccurred())

		originalNS, err = testutils.NewNS()
		Expect(err).NotTo(HaveOccurred())

		targetNS, err = testutils.NewNS()
		Expect(err).NotTo(HaveOccurred())

		log.Printf("Namespaces: %s, %s, %s", hostNS.Path(), originalNS.Path(), targetNS.Path())

		dataDir, err = ioutil.TempDir("", "multi-nic_test")
		Expect(err).NotTo(HaveOccurred())

		var hostVethName string
		// Setup host network namespace for testing
		err = originalNS.Do(func(ns.NetNS) error {
			defer GinkgoRecover()

			// Add master
			for index, master := range POOL_MASTER_NAMES {
				linkAttrs := netlink.LinkAttrs{
					Name: master,
				}
				err = netlink.LinkAdd(&netlink.Dummy{
					linkAttrs,
				})
				masterLink, err := netlink.LinkByName(master)
				Expect(err).NotTo(HaveOccurred())

				addr, _ := netlink.ParseAddr(POOL_IP_ADDRESSES[index])
				netlink.AddrAdd(masterLink, addr)
				Expect(err).NotTo(HaveOccurred())
			}
			// set lo dev up
			localLink, err := netlink.LinkByName("lo")
			Expect(err).NotTo(HaveOccurred())
			err = netlink.LinkSetUp(localLink)
			Expect(err).NotTo(HaveOccurred())

			// Link testing namespace to base namespace
			hostVeth, containerVeth, err := ip.SetupVeth("tmpBridge", 1500, "", hostNS)
			Expect(err).NotTo(HaveOccurred())
			hostVethName = hostVeth.Name

			log.Printf("Create veth pair %s, %s", hostVeth.Name, containerVeth.Name)

			containerVethLink, err := netlink.LinkByName(containerVeth.Name)
			Expect(err).NotTo(HaveOccurred())

			bridgeAddress, err := netlink.ParseAddr(BRIDGE_CONTAINER_IP + "/24")
			netlink.AddrAdd(containerVethLink, bridgeAddress)
			Expect(err).NotTo(HaveOccurred())

			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		hostVethLink, err := netlink.LinkByName(hostVethName)
		bridgeAddress, err := netlink.ParseAddr(BRIDGE_HOST_IP + "/24")
		netlink.AddrAdd(hostVethLink, bridgeAddress)
		Expect(err).NotTo(HaveOccurred())

		// Start Daemon server
		httpServerExitDone.Add(1)
		srv = startDaemonServer(httpServerExitDone)
	})

	AfterEach(func() {
		closeServer(srv, httpServerExitDone)

		Expect(os.RemoveAll(dataDir)).To(Succeed())

		Expect(originalNS.Close()).To(Succeed())
		Expect(testutils.UnmountNS(originalNS)).To(Succeed())

		Expect(targetNS.Close()).To(Succeed())
		Expect(testutils.UnmountNS(targetNS)).To(Succeed())

	})

	//	for _, ver := range testutils.AllSpecVersions {
	// Redefine ver inside for scope so real value is picked up by each dynamically defined It()
	// See Gingkgo's "Patterns for dynamically generating tests" documentation.
	for _, ver := range []string{"0.3.0"} {
		ver := ver
		masterNetsBytes, _ := json.Marshal(POOL_NETWORK_ADDRESSES)
		masterNets := string(masterNetsBytes)
		singleNICIPAM := `"ipam": {
			"type": "static",
			"addresses": [{"address": "192.168.1.1/24"}]
		},
		"multiNICIPAM": false,
		`
		It(fmt.Sprintf("[%s] configures and deconfigures link with ADD/DEL (multi-nic IPAM)", ver), func() {
			multiNICIPAM := fmt.Sprintf(`"ipam": {
				"type": "multi-nic-ipam",
				"hostBlock": 8,
				"interfaceBlock": 2,
				"daemonIP": "%s",
				"daemonPort": %d
				},
			"multiNICIPAM": true,
			`, BRIDGE_HOST_IP, daemonPort)
			conf := getConfig(ver, multiNICIPAM, masterNets)
			multinicAddCheckDelTest(conf, "", originalNS, targetNS)
		})

		It(fmt.Sprintf("[%s] configures and deconfigures link with ADD/DEL (single-nic IPAM)", ver), func() {
			conf := getConfig(ver, singleNICIPAM, masterNets)
			multinicAddCheckDelTest(conf, "", originalNS, targetNS)
		})

		It(fmt.Sprintf("[%s] check config load", ver), func() {
			conf, n := getAwsIpvlanConfig(ver, masterNets)
			podIP := "192.168.0.1/24"
			ipVal, ipnet, err := net.ParseCIDR(podIP)
			ipnet.IP = ipVal
			Expect(err).NotTo(HaveOccurred())
			nodeIP := getHostIP("eth0")
			log.Printf("Host IP: %s", nodeIP.String())
			podIPConfig := &types100.IPConfig{Address: *ipnet}
			confBytesArray, err := loadAWSCNIConf(conf, "net1", n, []*types100.IPConfig{podIPConfig})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(confBytesArray)).NotTo(Equal(0))
			log.Printf("%s", string(confBytesArray[0]))
			confObj := &AWSIPVLANNetConf{}
			err = json.Unmarshal(confBytesArray[0], confObj)
			Expect(err).NotTo(HaveOccurred())
			Expect(confObj.PodIP).To(Equal(ipVal.String()))
		})
	}
})

func getConfig(ver, ipamValue, masterNets string) string {
	return fmt.Sprintf(`{ 
		"cniVersion": "0.3.0", 
		"name": "multi-nic-sample",
		"type": "multi-nic",
		"plugin": {
			"cniVersion": "0.3.0",
			"type": "ipvlan",
			"mode": "l3"
		},
		"vlanMode": "l3",
		%s
		"daemonIP": "%s",
		"daemonPort": %d,
		"subnet": "192.168.0.0/16",
		"masterNets": %s
		}`, ipamValue, BRIDGE_HOST_IP, daemonPort, masterNets)
}

func getAwsIpvlanConfig(ver, masterNets string) ([]byte, *NetConf) {
	confStr := fmt.Sprintf(`{ 
		"cniVersion": "%s", 
		"name": "multi-nic-sample",
		"type": "multi-nic",
		"plugin": {
			"cniVersion": "0.3.0",
			"type": "aws-ipvlan",
			"mode": "l3"
		},
		"vlanMode": "l3",
		"ipam": {},
		"multiNICIPAM": true,
		"daemonIP": "%s",
		"daemonPort": %d,
		"subnet": "192.168.0.0/16",
		"masterNets": %s
		}`, ver, BRIDGE_HOST_IP, daemonPort, masterNets)
	log.Printf("%s", confStr)
	conf := []byte(confStr)
	n := &NetConf{}
	err := json.Unmarshal(conf, n)
	Expect(err).NotTo(HaveOccurred())
	n.DeviceIDs = POOL_MASTER_NAMES[0:1]
	n.Masters = POOL_MASTER_NAMES[0:1]
	return conf, n
}

func closeServer(srv *http.Server, httpServerExitDone *sync.WaitGroup) {
	if err := srv.Shutdown(context.TODO()); err != nil {
		panic(err)
	}
	httpServerExitDone.Wait()
	log.Printf("Connection closed")
}

func startDaemonServer(httpServerExitDone *sync.WaitGroup) *http.Server {
	log.Printf("startDaemonServer")
	var srv *http.Server
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/"+ALLOCATE_PATH,
		func(w http.ResponseWriter, r *http.Request) {
			ioutil.ReadAll(r.Body)
			var ipResponses []IPResponse
			ipResponses = []IPResponse{}
			for index, master := range MASTER_NAMES {
				response := IPResponse{
					InterfaceName: master,
					IPAddress:     NEXT_ADDRESSES[index],
					VLANBlockSize: "24",
				}
				ipResponses = append(ipResponses, response)
			}
			log.Printf("return responses: %v", ipResponses)
			json.NewEncoder(w).Encode(ipResponses)
		},
	).Methods("POST")
	router.HandleFunc("/"+DEALLOCATE_PATH,
		func(w http.ResponseWriter, r *http.Request) {
			ioutil.ReadAll(r.Body)
			json.NewEncoder(w).Encode("")
		},
	).Methods("POST")
	router.HandleFunc("/"+NIC_SELECT_PATH,
		func(w http.ResponseWriter, r *http.Request) {
			ioutil.ReadAll(r.Body)
			selectResponse := NICSelectResponse{
				DeviceIDs: []string{},
				Masters:   MASTER_NAMES,
			}
			log.Printf("return responses: %v", selectResponse)
			json.NewEncoder(w).Encode(selectResponse)
		},
	)

	// use next available portt
	srv = &http.Server{Addr: fmt.Sprintf("%s:0", BRIDGE_HOST_IP), Handler: router}

	go func() {
		defer httpServerExitDone.Done() // let main know we are done cleaning up
		log.Printf("Server Listening")
		// ListenAndServe
		addr := srv.Addr
		if addr == "" {
			addr = ":http"
		}
		ln, err := net.Listen("tcp", addr)
		Expect(err).NotTo(HaveOccurred())
		daemonPort = ln.Addr().(*net.TCPAddr).Port

		// always returns error. ErrServerClosed on graceful close
		if err := srv.Serve(ln); err != http.ErrServerClosed {
			// unexpected error. port in use?
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	time.Sleep(2 * time.Second)

	return srv
}
