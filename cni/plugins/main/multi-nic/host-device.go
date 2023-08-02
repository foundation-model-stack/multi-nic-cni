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

const (
	HostDeviceIPAMType = "host-device-ipam"
)

type HostDeviceTypeNetConf struct {
	types.NetConf
	MainPlugin *HostDeviceNetConf `json:"plugin"`
}

type HostDeviceRuntimConfig struct {
	DeviceID string `json:"deviceID,omitempty"`
}

// https://github.com/containernetworking/plugins/blob/283f200489b5ef8f0b6aadc09f751ab0c5145497/plugins/main/host-device/host-device.go#L45C1-L56C2
type HostDeviceNetConf struct {
	types.NetConf
	Device        string `json:"device"` // Device-Name, something like eth0 or can0 etc.
	HWAddr        string `json:"hwaddr"` // MAC Address of target network interface
	DPDKMode      bool
	KernelPath    string                 `json:"kernelpath"` // Kernelpath of the device
	PCIAddr       string                 `json:"pciBusID"`   // PCI Address of target network device
	RuntimeConfig HostDeviceRuntimConfig `json:"runtimeConfig,omitempty"`
}

// loadHostDeviceConf unmarshal to HostDeviceNetConf and returns list of SR-IOV configs
func loadHostDeviceConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) (confBytesArray [][]byte, multiPathRoutes map[string][]*netlink.NexthopInfo, loadError error) {
	confBytesArray = [][]byte{}

	configInHostDevice := HostDeviceTypeNetConf{}
	if err := json.Unmarshal(bytes, &configInHostDevice); err != nil {
		loadError = err
		return
	}

	// interfaces are orderly assigned from interface set
	for index, deviceID := range n.DeviceIDs {
		if deviceID == "" {
			continue
		}
		// add config
		singleConfig, err := copyHostDeviceConfig(configInHostDevice.MainPlugin)
		if err != nil {
			loadError = err
			return
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.RuntimeConfig = HostDeviceRuntimConfig{
			DeviceID: deviceID,
		}
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			loadError = err
			return
		}
		if n.IPAM.Type == HostDeviceIPAMType {
			ipConfig := getHostIPConfig(index, n.Masters[index])
			if ipConfig == nil {
				continue
			}
			confBytes, multiPathRoutes = replaceMultiNicIPAM(confBytes, bytes, ipConfig)
		} else if n.IsMultiNICIPAM {
			// multi-NIC IPAM config
			confBytes, multiPathRoutes = injectMultiNicIPAM(confBytes, bytes, ipConfigs, index)
		} else {
			confBytes, multiPathRoutes = injectSingleNicIPAM(confBytes, bytes)
		}
		confBytesArray = append(confBytesArray, confBytes)
	}
	return
}

// copyHostDeviceConfig makes a copy of base host-device config
func copyHostDeviceConfig(original *HostDeviceNetConf) (*HostDeviceNetConf, error) {
	copiedObject := &HostDeviceNetConf{}
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
