/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"strconv"

	"github.com/gorilla/mux"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	da "github.com/foundation-model-stack/multi-nic-cni/e2e-test/daemon-stub/allocator"
	"github.com/foundation-model-stack/multi-nic-cni/e2e-test/daemon-stub/backend"
)

type InterfaceInfoType struct {
	InterfaceName string `json:"interfaceName"`
	NetAddress    string `json:"netAddress"`
	HostIP        string `json:"hostIP"`
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	PciAddress    string `json:"pciAddress"`
}

type RouteUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"msg"`
}

type NICSelectResponse struct {
	DeviceIDs []string `json:"deviceIDs"`
	Masters   []string `json:"masters"`
}

type IPResponse struct {
	InterfaceName string `json:"interface"`
	IPAddress     string `json:"ip"`
	VLANBlockSize string `json:"block"`
}

type IPAMInfo struct {
	HIFList []InterfaceInfoType `json:"hifs"`
}

const (
	JOIN_PATH  = "/join"
	GREET_PATH = "/greet"

	INTERFACE_PATH = "/interface"

	ADD_ROUTE_PATH    = "/addroute"
	DELETE_ROUTE_PATH = "/deleteroute"

	ADD_L3CONFIG_PATH    = "/addl3"
	DELETE_L3CONFIG_PATH = "/deletel3"

	ALLOCATE_PATH   = "/allocate"
	DEALLOCATE_PATH = "/deallocate"

	NIC_SELECT_PATH = "/select"
)

var (
	interfaceNames []string = []string{"eth0", "eth1", "eth2"}
	netAddresses   []string = []string{"", "10.0.0.0/16", "10.1.0.0/16"}
	pciAddresses   []string = []string{"0000:00:03.0", "0000:00:04.0", "0000:00:05.0"}
	vendor                  = "1af4"
	product                 = "1000"

	DAEMON_PORT = 11000
	HostIP      string
)

func handleRequests() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(JOIN_PATH, Join).Methods("POST")
	router.HandleFunc(GREET_PATH, GreetAck).Methods("POST")
	router.HandleFunc(INTERFACE_PATH, GetInterface)
	router.HandleFunc(ADD_ROUTE_PATH, AddRoute).Methods("POST")
	router.HandleFunc(DELETE_ROUTE_PATH, DeleteRoute).Methods("POST")
	router.HandleFunc(ADD_L3CONFIG_PATH, ApplyL3Config).Methods("POST")
	router.HandleFunc(DELETE_L3CONFIG_PATH, DeleteL3Config).Methods("POST")
	router.HandleFunc(NIC_SELECT_PATH, SelectNic).Methods("POST")
	router.HandleFunc(ALLOCATE_PATH, Allocate).Methods("POST")
	router.HandleFunc(DEALLOCATE_PATH, Deallocate).Methods("POST")
	return router
}

func getSecondaryHostIPFromMainIP() []string {
	hostIPs := []string{}
	for _, netAddress := range netAddresses {
		netSplits := strings.Split(netAddress, ".")
		ipSplits := strings.Split(HostIP, ".")
		ip := fmt.Sprintf("%s.%s.%s.%s", netSplits[0], netSplits[1], ipSplits[2], ipSplits[3])
		hostIPs = append(hostIPs, ip)
	}
	return hostIPs
}

func getInterfaces() []InterfaceInfoType {
	hostIPs := getSecondaryHostIPFromMainIP()
	dummyInterfaces := make([]InterfaceInfoType, len(interfaceNames))
	for index, interfaceName := range interfaceNames {
		dummyInterfaces[index] = InterfaceInfoType{
			InterfaceName: interfaceName,
			NetAddress:    netAddresses[index],
			HostIP:        hostIPs[index],
			Vendor:        vendor,
			Product:       product,
			PciAddress:    pciAddresses[index],
		}
	}
	return dummyInterfaces
}

func routeResponse() RouteUpdateResponse {
	response := RouteUpdateResponse{Success: true, Message: ""}
	return response
}

func Join(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Fail to read: %v", err)
	}
	var ipamInfo IPAMInfo
	err = json.Unmarshal(reqBody, &ipamInfo)
	if err != nil {
		log.Printf("Fail to unmarshal ipam info: %v", err)
	}
	json.NewEncoder(w).Encode("")
}

func Greet(targetHost string, myIP string) {
	if targetHost == myIP {
		return
	}
	address := fmt.Sprintf("http://%s:%d", targetHost, DAEMON_PORT) + GREET_PATH
	jsonReq, err := json.Marshal(myIP)

	if err != nil {
		log.Printf("Fail to marshal: %v", err)
		return
	} else {
		res, err := http.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			log.Printf("Fail to post: %v", err)
			return
		}
		if res.StatusCode != http.StatusOK {
			log.Printf("Status: %v", res.StatusCode)
			return
		}
		ioutil.ReadAll(res.Body)
	}
	if myIP != "" {
		log.Printf("Greeting %s from %s", targetHost, myIP)
	}
}

func GreetAck(w http.ResponseWriter, r *http.Request) {
	var host string
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Fail to read: %v", err)
	}
	err = json.Unmarshal(reqBody, &host)

	if host != "" {
		log.Printf("Acknowledge greeting from %s", host)
		go Greet(host, "")
	}
	json.NewEncoder(w).Encode("")
}

func GetInterface(w http.ResponseWriter, r *http.Request) {
	interfaces := getInterfaces()
	json.NewEncoder(w).Encode(interfaces)
}

func AddRoute(w http.ResponseWriter, r *http.Request) {
	response := routeResponse()
	json.NewEncoder(w).Encode(response)
}

func ApplyL3Config(w http.ResponseWriter, r *http.Request) {
	response := routeResponse()
	json.NewEncoder(w).Encode(response)
}

func DeleteL3Config(w http.ResponseWriter, r *http.Request) {
	response := routeResponse()
	json.NewEncoder(w).Encode(response)
}

func DeleteRoute(w http.ResponseWriter, r *http.Request) {
	response := routeResponse()
	json.NewEncoder(w).Encode(response)
}

func SelectNic(w http.ResponseWriter, r *http.Request) {
	resp := NICSelectResponse{
		DeviceIDs: []string{},
		Masters:   interfaceNames,
	}
	json.NewEncoder(w).Encode(resp)
}

func Allocate(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req da.IPRequest
	err := json.Unmarshal(reqBody, &req)
	var ipResponses []da.IPResponse
	if err == nil {
		log.Println(fmt.Sprintf("request: %v", req))
		ipResponses = da.AllocateIP(req)
		log.Println(fmt.Sprintf("return: %v (%s)", ipResponses, req.HostName))
	} else {
		log.Println(fmt.Sprintf("allocate fail: %v", err))
	}
	json.NewEncoder(w).Encode(ipResponses)
}

func Deallocate(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req da.IPRequest
	err := json.Unmarshal(reqBody, &req)
	if err == nil {
		da.DeallocateIP(req)
	}
	json.NewEncoder(w).Encode("")
}

func InitClient() *rest.Config {
	var config *rest.Config
	var err error
	presentKube, ok := os.LookupEnv("KUBECONFIG_FILE")
	if !ok && presentKube != "" {
		log.Println("InCluster Config")
		config, err = rest.InClusterConfig()
	} else {
		log.Printf("Config %s", presentKube)
		config, err = clientcmd.BuildConfigFromFlags("", presentKube)
	}
	if err != nil {
		log.Printf("Config Error: %v", err)
	}

	da.IppoolHandler = backend.NewIPPoolHandler(config)
	da.K8sClientset, _ = kubernetes.NewForConfig(config)
	return config

	return config
}

func main() {
	InitClient()
	var found bool
	HostIP, found = os.LookupEnv("HOST_IP")
	if !found {
		log.Fatal("No HOST_IP set")
	}
	setDaemonPort, found := os.LookupEnv("DAEMON_PORT")
	if found && setDaemonPort != "" {
		setDaemonPortInt, err := strconv.Atoi(setDaemonPort)
		if err == nil {
			DAEMON_PORT = setDaemonPortInt
		}
	}
	ipSplits := strings.Split(HostIP, ".")
	netAddresses[0] = fmt.Sprintf("%s.%s.0.0/16", ipSplits[0], ipSplits[1])
	router := handleRequests()
	daemonAddress := fmt.Sprintf("0.0.0.0:%d", DAEMON_PORT)
	log.Printf("Listening @%s", daemonAddress)
	log.Fatal(http.ListenAndServe(daemonAddress, router))
}