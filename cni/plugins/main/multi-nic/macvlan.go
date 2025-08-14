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
)

// MACVLANNetConfig defines macvlan net config
type MACVLANNetConfig struct {
	types.NetConf
	MainPlugin MACVLANTypeNetConf `json:"plugin"`
}

type MACVLANTypeNetConf struct {
	types.NetConf
	Master string `json:"master"`
	Mode   string `json:"mode"`
	MTU    int    `json:"mtu"`
}

// loadMACVLANConf unmarshals to MACVLANNetConfig and returns a list of MACVLAN configs
func loadMACVLANConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) ([][]byte, error) {
	confBytesArray := [][]byte{}

	configInMACVLAN := &MACVLANNetConfig{}
	if err := json.Unmarshal(bytes, configInMACVLAN); err != nil {
		return confBytesArray, err
	}

	// Interfaces are orderly assigned from the interface set
	for index, masterName := range n.Masters {
		if masterName == "" {
			continue
		}

		// Add config
		singleConfig, err := copyMACVLANConfig(configInMACVLAN.MainPlugin)
		if err != nil {
			return confBytesArray, err
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.Master = masterName
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			return confBytesArray, err
		}

		if n.IsMultiNICIPAM {
			// Multi-NIC IPAM config
			confBytes = injectMultiNicIPAM(confBytes, ipConfigs, index)
		} else {
			// Single-NIC IPAM config
			confBytes = injectSingleNicIPAM(confBytes, bytes)
		}
		confBytesArray = append(confBytesArray, confBytes)
	}
	return confBytesArray, nil
}

// copyMACVLANConfig makes a copy of the base MACVLAN config
func copyMACVLANConfig(original MACVLANTypeNetConf) (*MACVLANTypeNetConf, error) {
	copiedObject := &MACVLANTypeNetConf{}
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
