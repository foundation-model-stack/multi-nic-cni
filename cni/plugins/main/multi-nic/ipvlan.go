/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/utils"
	"github.com/vishvananda/netlink"
)

// IPVLANNetConfig defines ipvlan net config
// Master string `json:"master"`
// Mode   string `json:"mode"`
// MTU    int    `json:"mtu"`
type IPVLANNetConfig struct {
	types.NetConf
	MainPlugin IPVLANTypeNetConf `json:"plugin"`
}

type IPVLANTypeNetConf struct {
	types.NetConf
	Master string `json:"master"`
	Mode   string `json:"mode"`
	MTU    int    `json:"mtu"`
}

// loadIPVANConf unmarshal to IPVLANNetConfig and returns list of IPVLAN configs
func loadIPVANConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) (confBytesArray [][]byte, multiPathRoutes map[string][]*netlink.NexthopInfo, loadError error) {
	confBytesArray = [][]byte{}

	configInIPVLAN := &IPVLANNetConfig{}
	if err := json.Unmarshal(bytes, configInIPVLAN); err != nil {
		loadError = err
		return
	}

	var ipamMap map[string][]byte

	if n.IPAM.Type == MultiConfigIPAMType {
		var err error
		ipamMap, err = getMultiIPAMConfigBytes(bytes)
		if err != nil {
			ipamMap = nil
			utils.Logger.Debug(fmt.Sprintf("getMultiIPAMConfigBytes failed: %v", err))
		}
	}

	// interfaces are orderly assigned from interface set
	for index, masterName := range n.Masters {
		if masterName == "" {
			continue
		}
		// add config
		singleConfig, err := copyIPVLANConfig(configInIPVLAN.MainPlugin)
		if err != nil {
			loadError = err
			return
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.Master = masterName
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			loadError = err
			return
		}

		if len(ipConfigs) > 0 {
			// no need to call ipam due to static ip
			confBytes, multiPathRoutes = injectMultiNicIPAM(confBytes, bytes, ipConfigs, index)
		} else if ipamMap != nil {
			if ipamBytes, found := ipamMap[masterName]; found {
				confBytes, multiPathRoutes = replaceSingleNicIPAMWithMultiConfig(confBytes, bytes, ipamBytes)
			} else {
				utils.Logger.Debug(fmt.Sprintf("Multi-config IPAM has no definition of %s", masterName))
				continue
			}
		} else {
			confBytes, multiPathRoutes = injectSingleNicIPAM(confBytes, bytes)
		}
		confBytesArray = append(confBytesArray, confBytes)
	}
	return
}

// copyIPVLANConfig makes a copy of base IPVLAN config
func copyIPVLANConfig(original IPVLANTypeNetConf) (*IPVLANTypeNetConf, error) {
	copiedObject := &IPVLANTypeNetConf{}
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
