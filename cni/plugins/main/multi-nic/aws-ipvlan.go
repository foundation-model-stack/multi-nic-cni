/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/utils"
	"github.com/vishvananda/netlink"
)

func getHostIP(devName string) net.IP {
	devLink, err := netlink.LinkByName(devName)
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("cannot find link %s: %v", devName, err))
		return nil
	}
	addrs, err := netlink.AddrList(devLink, netlink.FAMILY_V4)
	if err != nil || len(addrs) == 0 {
		utils.Logger.Debug(fmt.Sprintf("cannot list address on %s: %v", devName, err))
		return nil
	}
	return addrs[0].IPNet.IP.To4()
}

type AWSIPVLANNetConfig struct {
	types.NetConf
	MainPlugin AWSIPVLANNetConf `json:"plugin"`
}

type AWSIPVLANNetConf struct {
	types.NetConf
	PrimaryIP string `json:"primaryIP"`
	PodIP     string `json:"podIP"`
	Master    string `json:"master"`
	Mode      string `json:"mode"`
	MTU       int    `json:"mtu"`
}

// loadAWSCNIConf unmarshal to AWSCNINetConfig and returns list of AWSCNI configs
func loadAWSCNIConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) (confBytesArray [][]byte, multiPathRoutes map[string][]*netlink.NexthopInfo, loadError error) {
	confBytesArray = [][]byte{}

	configInAWSIPVLAN := &AWSIPVLANNetConfig{}
	if err := json.Unmarshal(bytes, configInAWSIPVLAN); err != nil {
		loadError = fmt.Errorf("unmarshal AWSIPVLANNetConfig: %v", err)
		return
	}

	// interfaces are orderly assigned from interface set
	for index, masterName := range n.Masters {
		// add config
		singleConfig, err := copyAWSIPVLANConfig(configInAWSIPVLAN.MainPlugin)
		if err != nil {
			loadError = fmt.Errorf("copyAWSIPVLANConfig: %v", err)
			return
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.Master = masterName
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			loadError = fmt.Errorf("Marshal confBytes: %v", err)
			return
		}
		// add primary IP value
		nodeIP := getHostIP(masterName)
		primaryIP := nodeIP.String()
		confBytes = injectPrimaryIP(confBytes, primaryIP)
		if n.IsMultiNICIPAM {
			// multi-NIC IPAM config
			if index < len(ipConfigs) {
				confBytes, multiPathRoutes = injectMultiNicIPAM(confBytes, bytes, ipConfigs, index)
				podIP := ipConfigs[index].Address.IP.String()
				// add pod IP value
				confBytes = injectPodIP(confBytes, podIP)
				confBytesArray = append(confBytesArray, confBytes)
			} else {
				utils.Logger.Debug(fmt.Sprintf("index not match config %d, %v", index, ipConfigs))
			}
		} else {
			confBytes, multiPathRoutes = injectSingleNicIPAM(confBytes, bytes)
			confBytesArray = append(confBytesArray, confBytes)
			// TO-DO: get IP and inject podIP
		}
	}
	return
}

func injectPodIP(confBytes []byte, podIP string) []byte {
	confStr := string(confBytes)
	replaceValue := fmt.Sprintf("\"podIP\":\"%s\"", podIP)
	injectedStr := strings.ReplaceAll(confStr, "\"podIP\":\"\"", replaceValue)
	return []byte(injectedStr)
}

func injectPrimaryIP(confBytes []byte, primaryIP string) []byte {
	confStr := string(confBytes)
	replaceValue := fmt.Sprintf("\"primaryIP\":\"%s\"", primaryIP)
	injectedStr := strings.ReplaceAll(confStr, "\"primaryIP\":\"\"", replaceValue)
	return []byte(injectedStr)
}

// copyAWSIPVLANConfig makes a copy of AWS IPVLAN config
func copyAWSIPVLANConfig(original AWSIPVLANNetConf) (*AWSIPVLANNetConf, error) {
	copiedObject := &AWSIPVLANNetConf{}
	byteObject, err := json.Marshal(original)
	if err != nil {
		return copiedObject, err
	}
	err = json.Unmarshal(byteObject, copiedObject)
	if err != nil {
		return copiedObject, err
	}
	return copiedObject, nil
}
