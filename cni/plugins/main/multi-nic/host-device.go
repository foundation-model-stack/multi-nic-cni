/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/utils"
)

const (
	HostDeviceIPAMType = "host-device-ipam"
	sysBusPCI          = "/sys/bus/pci/devices"
)

type HostDeviceTypeNetConf struct {
	types.NetConf
	MainPlugin *HostDeviceNetConf `json:"plugin"`
}

type HostDeviceRuntimeConfig struct {
	DeviceID string `json:"deviceID,omitempty"`
}

// https://github.com/containernetworking/plugins/blob/283f200489b5ef8f0b6aadc09f751ab0c5145497/plugins/main/host-device/host-device.go#L45C1-L56C2
type HostDeviceNetConf struct {
	types.NetConf
	Device        string `json:"device"` // Device-Name, something like eth0 or can0 etc.
	HWAddr        string `json:"hwaddr"` // MAC Address of target network interface
	DPDKMode      bool
	KernelPath    string                  `json:"kernelpath"` // Kernelpath of the device
	PCIAddr       string                  `json:"pciBusID"`   // PCI Address of target network device
	RuntimeConfig HostDeviceRuntimeConfig `json:"runtimeConfig,omitempty"`
}

// loadHostDeviceConf unmarshal to HostDeviceNetConf and returns list of SR-IOV configs
func loadHostDeviceConf(bytes []byte, ifName string, n *NetConf, ipConfigs []*current.IPConfig) ([][]byte, error) {
	confBytesArray := [][]byte{}

	configInHostDevice := HostDeviceTypeNetConf{}
	if err := json.Unmarshal(bytes, &configInHostDevice); err != nil {
		return confBytesArray, err
	}

	// interfaces are orderly assigned from interface set
	for index, deviceID := range n.DeviceIDs {
		if deviceID == "" {
			utils.Logger.Debug(fmt.Sprintf("skip %d: no device ID", index))
			continue
		}
		// add config
		singleConfig, err := copyHostDeviceConfig(configInHostDevice.MainPlugin)
		if err != nil {
			return confBytesArray, err
		}
		if singleConfig.CNIVersion == "" {
			singleConfig.CNIVersion = n.CNIVersion
		}
		singleConfig.Name = fmt.Sprintf("%s-%d", ifName, index)
		singleConfig.RuntimeConfig = HostDeviceRuntimeConfig{
			DeviceID: deviceID,
		}
		confBytes, err := json.Marshal(singleConfig)
		if err != nil {
			return confBytesArray, err
		}
		if n.IPAM.Type == HostDeviceIPAMType {
			ipConfig := getHostIPConfig(index, n.Masters[index], deviceID)
			if ipConfig == nil {
				utils.Logger.Debug(fmt.Sprintf("skip %d: no host IP", index))
				confBytes = replaceEmptyIPAM(confBytes)
			}
			confBytes = replaceMultiNicIPAM(confBytes, ipConfig)
		} else if n.IsMultiNICIPAM {
			// multi-NIC IPAM config
			confBytes = injectMultiNicIPAM(confBytes, ipConfigs, index)
		} else {
			confBytes = injectSingleNicIPAM(confBytes, bytes)
		}
		confBytesArray = append(confBytesArray, confBytes)
	}
	return confBytesArray, nil
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

func getLinkNameFromPciAddress(pciaddr string) (string, error) {
	netDir := filepath.Join(sysBusPCI, pciaddr, "net")
	if _, err := os.Lstat(netDir); err != nil {
		virtioNetDir := filepath.Join(sysBusPCI, pciaddr, "virtio*", "net")
		matches, err := filepath.Glob(virtioNetDir)
		if matches == nil || err != nil {
			return "", fmt.Errorf("no net directory under pci device %s", pciaddr)
		}
		netDir = matches[0]
	}
	return linkNameFromPath(netDir)
}

// linkNameFromPath is modified from linkFromPath in HostDevice plugin CNI
// https://github.com/containernetworking/plugins/blob/main/plugins/main/host-device/host-device.go#L499
func linkNameFromPath(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %q", path, err)
	}
	if len(entries) > 0 {
		// grab the first net device
		return entries[0].Name(), nil
	}
	return "", fmt.Errorf("failed to find network device in path %s", path)
}
