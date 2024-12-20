/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"log"
	"net"
	"strings"

	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/utils"
	"github.com/vishvananda/netlink"
)

const (
	MultiConfigIPAMType = "multi-config"
	WhereaboutsIPAMType = "whereabouts"
)

type IPAMExtract struct {
	IPAM map[string]interface{} `json:"ipam"`
}

type MultiIPAMConfig struct {
	Name     string
	Type     string                            `json:"type"`
	IpamType string                            `json:"ipam_type"`
	Args     map[string]map[string]interface{} `json:"args"`
	Routes   []*types.Route                    `json:"routes"`
}

type MultiIPAMExtract struct {
	IPAM MultiIPAMConfig `json:"ipam"`
}

// getPodInfo extracts pod Name and Namespace from cniArgs
func getPodInfo(cniArgs string) (string, string) {
	splits := strings.Split(cniArgs, ";")
	var podName, podNamespace string
	for _, split := range splits {
		if strings.HasPrefix(split, "K8S_POD_NAME=") {
			podName = strings.TrimPrefix(split, "K8S_POD_NAME=")
		}
		if strings.HasPrefix(split, "K8S_POD_NAMESPACE=") {
			podNamespace = strings.TrimPrefix(split, "K8S_POD_NAMESPACE=")
		}
	}
	return podName, podNamespace
}

// getStaticIPs extracts static ip from cniArgs
// reference: https://github.com/k8snetworkplumbingwg/multus-cni/blob/e2e8cfb677e8cf5352f737b5004effed7518d71a/pkg/multus/multus.go#L324
func getStaticIPs(cniArgs string) ([]string, string) {
	splits := strings.Split(cniArgs, ";")
	for index, split := range splits {
		if strings.HasPrefix(split, "IP=") {
			ipsStr := strings.TrimPrefix(split, "IP=")
			newItems := splits[0:index]
			if index < len(split)-1 {
				newItems = append(newItems, splits[index+1:len(splits)]...)
			}
			newCniArgs := strings.Join(newItems, ";")
			return strings.Split(ipsStr, ","), newCniArgs
		}
	}
	return []string{}, cniArgs
}

func getMultiIPAMConfig(multiNicConfBytes []byte) (*MultiIPAMExtract, error) {
	ipamObject := &MultiIPAMExtract{}
	err := json.Unmarshal(multiNicConfBytes, ipamObject)
	return ipamObject, err

}

func getMultiIPAMConfigBytes(multiNicConfBytes []byte) (map[string][]byte, error) {
	multiIPAMConfigBytes := make(map[string][]byte)
	ipamObject, err := getMultiIPAMConfig(multiNicConfBytes)
	if err == nil && ipamObject.IPAM.Args != nil {
		for masterName, singleIPAMObject := range ipamObject.IPAM.Args {
			singleIPAMConfBytes := make(map[string]interface{})
			if singleIPAMObject == nil {
				utils.Logger.Debug(fmt.Sprintf("ipamObject.IPAM.Args not defined on %s", masterName))
				continue
			}
			// set type
			singleIPAMConfBytes["type"] = ipamObject.IPAM.IpamType
			if ipamObject.IPAM.IpamType == WhereaboutsIPAMType {
				singleIPAMConfBytes["network_name"] = masterName
			}
			// set args
			for k, v := range singleIPAMObject {
				singleIPAMConfBytes[k] = v
			}
			ipamBytes, _ := json.Marshal(singleIPAMConfBytes)
			multiIPAMConfigBytes[masterName] = ipamBytes
		}
	}
	return multiIPAMConfigBytes, err
}

// injectIPAM injects ipam bytes to config
func injectMultiNicIPAM(singleNicConfBytes, multiNicConfBytes []byte, ipConfigs []*current.IPConfig, ipIndex int) ([]byte, map[string][]*netlink.NexthopInfo) {
	var ipConfig *current.IPConfig
	if ipIndex < len(ipConfigs) {
		ipConfig = ipConfigs[ipIndex]
	}
	return replaceMultiNicIPAM(singleNicConfBytes, multiNicConfBytes, ipConfig)
}

func injectSingleNicIPAM(singleNicConfBytes []byte, multiNicConfBytes []byte) ([]byte, map[string][]*netlink.NexthopInfo) {
	return replaceSingleNicIPAM(singleNicConfBytes, multiNicConfBytes)
}

func replaceSingleNicIPAM(singleNicConfBytes, multiNicConfBytes []byte) ([]byte, map[string][]*netlink.NexthopInfo) {
	ipamObject := &IPAMExtract{}
	err := json.Unmarshal(multiNicConfBytes, ipamObject)
	if err == nil {
		ipamBytes, _ := json.Marshal(ipamObject.IPAM)
		return replaceSingleNicIPAMWithMultiConfig(singleNicConfBytes, multiNicConfBytes, ipamBytes)
	}
	return singleNicConfBytes, nil
}

func replaceSingleNicIPAMWithMultiConfig(singleNicConfBytes, multiNicConfBytes, ipamBytes []byte) ([]byte, map[string][]*netlink.NexthopInfo) {
	confStr := string(singleNicConfBytes)
	ipamObject := &IPAMExtract{}
	err := json.Unmarshal(multiNicConfBytes, ipamObject)
	if err == nil {
		nonMultiPathRoutes, multiPathRoutes := getRoutesFromIPAM(ipamObject)
		if nonMultiPathRoutes != nil {
			ipamObject.IPAM["routes"] = nonMultiPathRoutes
		}

		singleIPAM := fmt.Sprintf("\"ipam\":%s", string(ipamBytes))
		injectedStr := strings.ReplaceAll(confStr, "\"ipam\":{}", singleIPAM)
		return []byte(injectedStr), multiPathRoutes
	}
	return singleNicConfBytes, nil
}

func replaceMultiNicIPAM(singleNicConfBytes, multiNicConfBytes []byte, ipConfig *current.IPConfig) ([]byte, map[string][]*netlink.NexthopInfo) {
	confStr := string(singleNicConfBytes)
	ipamObject := &IPAMExtract{}
	singleIPAMObject := make(map[string]interface{})
	singleIPAMObject["type"] = "static"
	if ipConfig != nil {
		singleIPAMObject["addresses"] = []map[string]string{
			map[string]string{"address": ipConfig.Address.String()},
		}
	} else {
		singleIPAMObject["addresses"] = []map[string]string{}
	}
	var multiPathRoutes map[string][]*netlink.NexthopInfo
	err := json.Unmarshal(multiNicConfBytes, ipamObject)
	if err == nil {
		var nonMultiPathRoutes []*types.Route
		nonMultiPathRoutes, multiPathRoutes = getRoutesFromIPAM(ipamObject)
		if nonMultiPathRoutes != nil {
			singleIPAMObject["routes"] = nonMultiPathRoutes
		}
	}
	ipamBytes, _ := json.Marshal(singleIPAMObject)
	singleIPAM := fmt.Sprintf("\"ipam\":%s", string(ipamBytes))
	injectedStr := strings.ReplaceAll(confStr, "\"ipam\":{}", singleIPAM)
	log.Printf("conf: %s -> injectedStr: %s", singleNicConfBytes, injectedStr)
	return []byte(injectedStr), multiPathRoutes
}

func getRoutesFromIPAM(ipamObject *IPAMExtract) (nonMultiPathRoutes []*types.Route, multiPathRoutes map[string][]*netlink.NexthopInfo) {
	routes := []*types.Route{}
	if routesInterface, found := ipamObject.IPAM["routes"]; found {
		for _, routeInterface := range routesInterface.([]interface{}) {
			routeMap := routeInterface.(map[string]interface{})
			if dstStr, ok := routeMap["dst"]; ok {
				_, dst, _ := net.ParseCIDR(dstStr.(string))
				if gwStr, ok := routeMap["gw"]; ok {
					gwIP := net.ParseIP(gwStr.(string))
					route := &types.Route{
						Dst: *dst,
						GW:  gwIP,
					}
					routes = append(routes, route)
				}
			}
		}
		multiPathRoutes, nonMultiPathRoutes = separateMultiPathRoutes(routes)
		return
	}
	return
}

// injectMaster replaces original pool with selected master network addresses
func injectMaster(inData []byte, selectedNetAddrs []string, selectedMasters []string, selectedDeviceIDs []string) []byte {
	var obj map[string]interface{}
	json.Unmarshal(inData, &obj)
	obj["masterNets"] = selectedNetAddrs
	obj["masters"] = selectedMasters
	obj["deviceIDs"] = selectedDeviceIDs
	outBytes, _ := json.Marshal(obj)
	return outBytes
}

// getHostIPConfig returns IP of host for a specific devName
func getHostIPConfig(index int, devName string) *current.IPConfig {
	devLink, err := netlink.LinkByName(devName)
	if err != nil {
		log.Printf("cannot find link %s: %v", devName, err)
		return nil
	}
	addrs, err := netlink.AddrList(devLink, netlink.FAMILY_V4)
	if err != nil || len(addrs) == 0 {
		log.Printf("cannot list address on %s: %v", devName, err)
		return nil
	}
	addr := addrs[0].IPNet
	ipConf := &current.IPConfig{
		Address:   *addr,
		Interface: current.Int(index),
	}
	return ipConf
}

// isBuiltInIPAM returns true if ipam is a built-in IPAM (host-device-ipam)
func isBuiltInIPAM(ipamType string) bool {
	return ipamType == HostDeviceIPAMType
}
