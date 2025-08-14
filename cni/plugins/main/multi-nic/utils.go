/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"log"
	"net"
	"os/exec"
	"strings"

	"fmt"

	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/utils"
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

func replaceEmptyIPAM(singleNicConfBytes []byte) []byte {
	confStr := string(singleNicConfBytes)
	injectedStr := strings.ReplaceAll(confStr, "\"ipam\":{}", "\"ipam\":{\"type\":\"static\",\"addresses\":[]}")
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

// getHostIPConfig returns IP of host for a specific device ID and correct devName if needed
func getHostIPConfig(index int, devName, deviceID string) *current.IPConfig {
	presentName, err := getLinkNameFromPciAddress(deviceID)
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("failed to get link name from device ID %s: %v", deviceID, err))
		return nil
	}
	devLink, err := netlink.LinkByName(presentName)
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("cannot find link %s: %v", presentName, err))
		return nil
	}
	addrs, err := netlink.AddrList(devLink, netlink.FAMILY_V4)
	if err != nil || len(addrs) == 0 {
		utils.Logger.Debug(fmt.Sprintf("cannot list address on %s: %v", devName, err))
		return nil
	}
	addr := addrs[0].IPNet
	ipConf := &current.IPConfig{
		Address:   *addr,
		Interface: current.Int(index),
	}
	if devName != "" && presentName != devName {
		if devLink.Attrs().Flags&net.FlagUp == net.FlagUp {
			if err = netlink.LinkSetDown(devLink); err != nil {
				log.Printf("WARNING: cannot set link down: %v", err)
			}
			defer func() {
				_ = netlink.LinkSetUp(devLink)
			}()
		}
		if err = netlink.LinkSetAlias(devLink, ""); err != nil {
			log.Printf("WARNING: cannot reset alias: %v", err)
		}

		if err = delAltName(presentName, devName); err != nil {
			log.Printf("WARNING: cannot del altname: %v", err)
		} else {
			log.Printf("successfully delete altname %s (%s)", devName, presentName)
		}

		// correct the device to the expected name
		if err = netlink.LinkSetName(devLink, devName); err != nil {
			utils.Logger.Debug(fmt.Sprintf("failed to rename host device %s to %s: %v", presentName, devName, err))
		} else {
			utils.Logger.Debug(fmt.Sprintf("successfully rename host device %s to %s", presentName, devName))
		}
	}
	return ipConf
}

// delAltName
// temporary solution, need upgrade to netlink 1.3.1 for using LinkDelAltName
func delAltName(presentName, devName string) error {
	cmd := exec.Command("ip", "link", "property", "del", "dev", presentName, "altname", devName)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
