/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/vishvananda/netlink"
)

// SRIOVNetConfig defines sriov net config
// references:
// - https://github.com/openshift/sriov-cni/blob/dfbc68063bb549910a5440d7c80e45a2519d12cc/pkg/config/config.go
// - https://github.com/k8snetworkplumbingwg/sriov-cni/blob/v2.1.0/cmd/sriov/main.go
type SRIOVNetConfig struct {
	types.NetConf
	MainPlugin *SriovNetConf `json:"plugin"`
}

// reference: github.com/k8snetworkplumbingwg/sriov-cni/pkg/types
// NetConf extends types.NetConf for sriov-cni
type SriovNetConf struct {
	types.NetConf
	DPDKMode    bool
	Master      string
	Vlan        int    `json:"vlan"`
	DeviceID    string `json:"deviceID"` // PCI address of a VF in valid sysfs format
	VFID        int
	HostIFNames string // VF netdevice name(s)
	ContIFNames string // VF names after in the container; used during deletion
}

// loadSRIOVConf unmarshal to SRIOVNetConfig and returns list of SR-IOV configs
func loadSRIOVConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) (confBytesArray [][]byte, multiPathRoutes map[string][]*netlink.NexthopInfo, loadError error) {
	confBytesArray = [][]byte{}

	configInSRIOV := SRIOVNetConfig{}
	if err := json.Unmarshal(bytes, &configInSRIOV); err != nil {
		loadError = err
		return
	}

	// interfaces are orderly assigned from interface set
	for index, deviceID := range n.DeviceIDs {
		if deviceID == "" {
			continue
		}
		// add config
		singleConfig, err := copySRIOVconfig(configInSRIOV.MainPlugin)
		if err != nil {
			loadError = err
			return
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.DeviceID = deviceID
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			loadError = err
			return
		}
		if n.IsMultiNICIPAM {
			// multi-NIC IPAM config
			confBytes, multiPathRoutes = injectMultiNicIPAM(confBytes, bytes, ipConfigs, index)
		} else {
			confBytes, multiPathRoutes = injectSingleNicIPAM(confBytes, bytes)
		}
		confBytesArray = append(confBytesArray, confBytes)
	}
	return
}

// copySRIOVconfig makes a copy of base SR-IOV config
func copySRIOVconfig(original *SriovNetConf) (*SriovNetConf, error) {
	copiedObject := &SriovNetConf{}
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
