/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"encoding/json"
	"log"
	"strings"

	"fmt"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/vishvananda/netlink"
)

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

// injectIPAM injects ipam bytes to config

func injectMultiNicIPAM(singleNicConfBytes []byte, ipConfigs []*current.IPConfig, ipIndex int) []byte {
	var ipConfig *current.IPConfig
	if ipIndex < len(ipConfigs) {
		ipConfig = ipConfigs[ipIndex]
	}
	return replaceMultiNicIPAM(singleNicConfBytes, ipConfig)
}
func injectSingleNicIPAM(singleNicConfBytes []byte, multiNicConfBytes []byte) []byte {
	return replaceSingleNicIPAM(singleNicConfBytes, multiNicConfBytes)
}

type IPAMExtract struct {
	IPAM map[string]interface{} `json:"ipam"`
}

func replaceSingleNicIPAM(singleNicConfBytes []byte, multiNicConfBytes []byte) []byte {
	confStr := string(singleNicConfBytes)
	ipamObject := &IPAMExtract{}
	json.Unmarshal(multiNicConfBytes, ipamObject)
	ipamBytes, _ := json.Marshal(ipamObject.IPAM)
	singleIPAM := fmt.Sprintf("\"ipam\":%s", string(ipamBytes))
	injectedStr := strings.ReplaceAll(confStr, "\"ipam\":{}", singleIPAM)
	return []byte(injectedStr)
}

func replaceMultiNicIPAM(singleNicConfBytes []byte, ipConfig *current.IPConfig) []byte {
	confStr := string(singleNicConfBytes)
	singleIPAM := "\"ipam\":{\"type\":\"static\",\"addresses\":[]}" // empty ipam
	if ipConfig != nil {
		singleIPAM = fmt.Sprintf("\"ipam\":{\"type\":\"static\",\"addresses\":[{\"address\":\"%s\"}]}", ipConfig.Address.String())
	}
	injectedStr := strings.ReplaceAll(confStr, "\"ipam\":{}", singleIPAM)
	return []byte(injectedStr)
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
