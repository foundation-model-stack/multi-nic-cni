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
	"strconv"
	"time"

	"github.com/gorilla/mux"

	da "github.com/foundation-model-stack/multi-nic-cni/daemon/allocator"
	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
	di "github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
	dr "github.com/foundation-model-stack/multi-nic-cni/daemon/router"
	ds "github.com/foundation-model-stack/multi-nic-cni/daemon/selector"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type IPAMInfo struct {
	HIFList []di.InterfaceInfoType `json:"hifs"`
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

var DAEMON_PORT int = 11000

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
	interfaces := di.GetInterfaces()
	ipNetMap := make(map[string]string)

	for _, iface := range interfaces {
		ipNetMap[iface.NetAddress] = iface.HostIP
	}

	hifs := ipamInfo.HIFList
	for _, hif := range hifs {
		if myIP, ok := ipNetMap[hif.NetAddress]; ok {
			go Greet(hif.HostIP, myIP)
		}
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
	interfaces := di.GetInterfaces()
	json.NewEncoder(w).Encode(interfaces)
}

func AddRoute(w http.ResponseWriter, r *http.Request) {
	response := dr.AddRoute(r)
	json.NewEncoder(w).Encode(response)
}

func ApplyL3Config(w http.ResponseWriter, r *http.Request) {
	response := dr.ApplyL3Config(r)
	json.NewEncoder(w).Encode(response)
}

func DeleteL3Config(w http.ResponseWriter, r *http.Request) {
	response := dr.DeleteL3Config(r)
	json.NewEncoder(w).Encode(response)
}

func DeleteRoute(w http.ResponseWriter, r *http.Request) {
	response := dr.DeleteRoute(r)
	json.NewEncoder(w).Encode(response)
}

func SelectNic(w http.ResponseWriter, r *http.Request) {
	startSelect := time.Now()
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req ds.NICSelectRequest
	err := json.Unmarshal(reqBody, &req)
	var resp ds.NICSelectResponse
	if err == nil {
		log.Println(fmt.Sprintf("request: %v", req))
		resp = ds.Select(req)
		elapsed := time.Since(startSelect)
		log.Println(fmt.Sprintf("%s SelectNic elapsed: %d us", req.HostName, int64(elapsed/time.Microsecond)))
		log.Println(fmt.Sprintf("return: %v", resp))
	} else {
		log.Println(fmt.Sprintf("select fail: %v", err))
	}
	json.NewEncoder(w).Encode(resp)
}

func Allocate(w http.ResponseWriter, r *http.Request) {
	startAllocate := time.Now()
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req da.IPRequest
	err := json.Unmarshal(reqBody, &req)
	var ipResponses []da.IPResponse
	if err == nil {
		log.Println(fmt.Sprintf("request: %v", req))
		ipResponses = da.AllocateIP(req)
		elapsed := time.Since(startAllocate)
		log.Println(fmt.Sprintf("%s WaitAndAllocate elapsed: %d us", req.HostName, int64(elapsed/time.Microsecond)))
		log.Println(fmt.Sprintf("return: %v", ipResponses))
	} else {
		log.Println(fmt.Sprintf("allocate fail: %v", err))
	}
	json.NewEncoder(w).Encode(ipResponses)
}

func Deallocate(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var req da.IPRequest
	var ipResponses []da.IPResponse
	err := json.Unmarshal(reqBody, &req)
	if err == nil {
		ipResponses = da.DeallocateIP(req)
	}
	json.NewEncoder(w).Encode(ipResponses)
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
	ds.MultinicnetHandler = backend.NewMultiNicNetworkHandler(config)
	ds.NetAttachDefHandler = backend.NewNetAttachDefHandler(config)
	ds.DeviceClassHandler = backend.NewDeviceClassHandler(config)
	da.K8sClientset, _ = kubernetes.NewForConfig(config)
	ds.K8sClientset, _ = kubernetes.NewForConfig(config)
	return config

}

func main() {
	InitClient()
	hostName, err := os.Hostname()
	if err != nil {
		log.Println("failed to get host name")
	}
	setDaemonPort, found := os.LookupEnv("DAEMON_PORT")
	if found && setDaemonPort != "" {
		setDaemonPortInt, err := strconv.Atoi(setDaemonPort)
		if err == nil {
			DAEMON_PORT = setDaemonPortInt
		}
	}
	dr.SetRTTablePath()

	da.CleanHangingAllocation(hostName)
	router := handleRequests()
	daemonAddress := fmt.Sprintf("0.0.0.0:%d", DAEMON_PORT)
	log.Printf("Listening @%s", daemonAddress)
	log.Fatal(http.ListenAndServe(daemonAddress, router))
}
