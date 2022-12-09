/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bytes"
	"errors"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CONNECTION_REFUSED = "connection refused"
	MAX_CONNECT_TRY    = 10
)

var DAEMON_NAMESPACE, DAEMON_PORT, INTERFACE_PATH, ADD_ROUTE_PATH, DELETE_ROUTE_PATH, REGISTER_IPAM_PATH string

// SetDaemon sets daemon environments
func SetDaemon(daemonSpec multinicv1.ConfigSpec) {
	DAEMON_PORT = fmt.Sprintf("%d", daemonSpec.Daemon.DaemonPort)
	DAEMON_NAMESPACE = OPERATOR_NAMESPACE
	INTERFACE_PATH = daemonSpec.InterfacePath
	ADD_ROUTE_PATH = daemonSpec.AddRoutePath
	DELETE_ROUTE_PATH = daemonSpec.DeleteRoutePath
	REGISTER_IPAM_PATH = daemonSpec.JoinPath
}

// GetDaemonAddressByPod returns daemon IP address (pod IP:daemon port)
func GetDaemonAddressByPod(daemon DaemonPod) string {
	return fmt.Sprintf("http://%s:%s", daemon.HostIP, DAEMON_PORT)
}

type DaemonConnector struct {
	*kubernetes.Clientset
}

// L3 Configuration defines request of l3 route configuration
type L3ConfigRequest struct {
	Name   string      `json:"name"`
	Subnet string      `json:"subnet"`
	Routes []HostRoute `json:"routes"`
	Force  bool        `json:"force"`
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
	HIFList []multinicv1.InterfaceInfoType `json:"hifs"`
}

// GetInterfaces returns HostInterface of specific host
func (dc DaemonConnector) GetInterfaces(podAddress string) ([]multinicv1.InterfaceInfoType, error) {
	var interfaces []multinicv1.InterfaceInfoType
	address := podAddress + INTERFACE_PATH
	// try connect and get interface from daemon pod
	res, err := http.Get(address)
	if err != nil {
		return []multinicv1.InterfaceInfoType{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []multinicv1.InterfaceInfoType{}, err
	}
	json.Unmarshal(body, &interfaces)
	return interfaces, nil
}

// Join notifies new daemon to get knowing the existing daemons on the other hosts
func (dc DaemonConnector) Join(podAddress string, hifs []multinicv1.InterfaceInfoType) error {
	address := podAddress + REGISTER_IPAM_PATH

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
func (dc DaemonConnector) ApplyL3Config(podAddress string, cidrName string, subnet string, routes []HostRoute, forceDelete bool) (RouteUpdateResponse, error) {
	return dc.putRouteRequest(podAddress, ADD_ROUTE_PATH, cidrName, subnet, routes, forceDelete)
}

// DeleteRoute sends a request to delete the route from specific host
func (dc DaemonConnector) DeleteL3Config(podAddress string, cidrName string, subnet string) (RouteUpdateResponse, error) {
	return dc.putRouteRequest(podAddress, DELETE_ROUTE_PATH, cidrName, subnet, []HostRoute{}, false)
}

// putRouteRequest sends a route adding/deleting request to specific host
func (dc DaemonConnector) putRouteRequest(podAddress string, path string, cidrName string, subnet string, routes []HostRoute, forceDelete bool) (RouteUpdateResponse, error) {
	address := podAddress + path
	var response RouteUpdateResponse

	requestL3Config := L3ConfigRequest{
		Name:   cidrName,
		Subnet: subnet,
		Routes: routes,
		Force:  forceDelete,
	}

	jsonReq, err := json.Marshal(requestL3Config)

	if err != nil {
		return response, err
	} else {
		res, err := http.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			response.Message = CONNECTION_REFUSED
			return response, err
		}
		if res.StatusCode != http.StatusOK {
			response.Message = CONNECTION_REFUSED
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
