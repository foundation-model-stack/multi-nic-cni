/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bytes"
	"errors"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CONNECTION_REFUSED = "connection refused"
	MAX_CONNECT_TRY    = 10
)

var DAEMON_NAMESPACE, DAEMON_PORT, INTERFACE_PATH, ADD_ROUTE_PATH, DELETE_ROUTE_PATH, REGISTER_IPAM_PATH string

// SetDaemon sets daemon environments
func SetDaemon(daemonSpec netcogadvisoriov1.ConfigSpec) {
	DAEMON_PORT = fmt.Sprintf("%d", daemonSpec.Daemon.DaemonPort)
	DAEMON_NAMESPACE = OPERATOR_NAMESPACE
	INTERFACE_PATH = daemonSpec.InterfacePath
	ADD_ROUTE_PATH = daemonSpec.AddRoutePath
	DELETE_ROUTE_PATH = daemonSpec.DeleteRoutePath
	REGISTER_IPAM_PATH = daemonSpec.JoinPath
}

// GetDaemonAddressByPod returns daemon IP address (pod IP:daemon port)
func GetDaemonAddressByPod(daemon v1.Pod) string {
	return fmt.Sprintf("http://%s:%s", daemon.Status.PodIP, DAEMON_PORT)
}

type DaemonConnector struct {
	*kubernetes.Clientset
}

// HostRoute defines a route
type HostRoute struct {
	Subnet        string `json:"net"`
	NextHop       string `json:"via"`
	InterfaceName string `json:"iface"`
}

// RouteUpdateResponse defines response from adding/deleting routes
type RouteUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"msg"`
}

// IPAMInfo defines information about HostInterface sent to daemon for greeting
type IPAMInfo struct {
	HIFList []netcogadvisoriov1.InterfaceInfoType `json:"hifs"`
}

// GetInterfaces returns HostInterface of specific host
func (dc DaemonConnector) GetInterfaces(daemon v1.Pod) ([]netcogadvisoriov1.InterfaceInfoType, error) {
	var interfaces []netcogadvisoriov1.InterfaceInfoType

	address := GetDaemonAddressByPod(daemon)
	// try connect and get interface from daemon pod
	res, err := http.Get(address + INTERFACE_PATH)
	if err != nil {
		return []netcogadvisoriov1.InterfaceInfoType{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []netcogadvisoriov1.InterfaceInfoType{}, err
	}
	json.Unmarshal(body, &interfaces)
	return interfaces, nil
}

// Join notifies new daemon to get knowing the existing daemons on the other hosts
func (dc DaemonConnector) Join(daemon v1.Pod, hifs []netcogadvisoriov1.InterfaceInfoType) error {
	address := GetDaemonAddressByPod(daemon) + REGISTER_IPAM_PATH

	ipamInfo := IPAMInfo{
		HIFList: hifs,
	}

	jsonReq, err := json.Marshal(ipamInfo)

	if err != nil {
		return err
	} else {
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		res, err := client.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errors.New(res.Status)
		}

		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return nil
	}
}

// AddRoute sends a request to add a new route to specific host
func (dc DaemonConnector) AddRoute(daemon v1.Pod, net string, via string, iface string) (RouteUpdateResponse, error) {
	return dc.putRouteRequest(daemon, ADD_ROUTE_PATH, net, via, iface)
}

// DeleteRoute sends a request to delete the route from specific host
func (dc DaemonConnector) DeleteRoute(daemon v1.Pod, net string, via string, iface string) (RouteUpdateResponse, error) {
	return dc.putRouteRequest(daemon, DELETE_ROUTE_PATH, net, via, iface)
}

// putRouteRequest sends a route adding/deleting request to specific host
func (dc DaemonConnector) putRouteRequest(daemon v1.Pod, path string, net string, via string, iface string) (RouteUpdateResponse, error) {
	address := GetDaemonAddressByPod(daemon) + path
	var response RouteUpdateResponse

	hostRoute := HostRoute{
		Subnet:        net,
		NextHop:       via,
		InterfaceName: iface,
	}

	jsonReq, err := json.Marshal(hostRoute)

	if err != nil {
		return response, err
	} else {
		res, err := http.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			return response, err
		}
		if res.StatusCode != http.StatusOK {
			return response, errors.New(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return response, err
		}
		err = json.Unmarshal(body, &response)
		return response, err
	}
}

// GetDaemonHostMap finds a map from host name to daemon pod
//                  uses clientset to get pods in DAEMON_NAMESPACE
//                       that are labeled with DAEMON_LABEL_NAME=DAEMON_LABEL_VALUE
func (dc DaemonConnector) GetDaemonHostMap() (map[string]v1.Pod, error) {
	podMap := make(map[string]v1.Pod)
	// get host IP -> host name
	nodeIPMap, err := dc.getHostNameIPMap()
	if err != nil {
		return podMap, err
	}

	// list daemon pods labeled with DAEMON_LABEL_NAME=DAEMON_LABEL_VALUE
	labels := fmt.Sprintf("%s=%s", DAEMON_LABEL_NAME, DAEMON_LABEL_VALUE)
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	daemonList, err := dc.Clientset.CoreV1().Pods(DAEMON_NAMESPACE).List(context.TODO(), listOptions)
	if err != nil {
		return podMap, err
	}

	for _, daemon := range daemonList.Items {
		hostIP := daemon.Status.HostIP
		// map from host IP to host name
		if hostName, exists := nodeIPMap[hostIP]; exists {
			// put map of host name to daemon pod
			podMap[hostName] = daemon
		} else {
			return podMap, fmt.Errorf("NodeNotFound")
		}
	}
	return podMap, nil
}

// getHostNameIPMap finds a map from NodeInternalIP of each host to node name
func (dc DaemonConnector) getHostNameIPMap() (map[string]string, error) {
	nodeIPMap := make(map[string]string)
	nodes, err := dc.Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nodeIPMap, err
	}
	for _, node := range nodes.Items {
		addrs := node.Status.Addresses
		ip := ""

		for _, addr := range addrs {
			if addr.Type == v1.NodeInternalIP {
				ip = addr.Address
				nodeIPMap[ip] = node.ObjectMeta.Name
				break
			}
		}
		if ip == "" {
			return nodeIPMap, fmt.Errorf("IPNotFound")
		}
	}
	return nodeIPMap, nil
}
