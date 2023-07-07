/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package iface

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"
	v1 "k8s.io/api/core/v1"
)

const (
	SysBusPci = "/sys/bus/pci/devices"
	netClass  = 0x02
)

var CheckPointfile string = "/var/lib/kubelet/device-plugins/kubelet_internal_checkpoint"

// modify from https://github.com/k8snetworkplumbingwg/multus-cni/blob/9b45d4b211728aa0db44a1624aac8e61843390cf/pkg/checkpoint/checkpoint.go#L72
// DeviceIDs can map[string]string or []string
type PodDevicesEntry struct {
	PodUID        string
	ContainerName string
	ResourceName  string
	DeviceIDs     interface{}
	AllocResp     []byte
}

type checkpointData struct {
	PodDeviceEntries  []PodDevicesEntry
	RegisteredDevices map[string][]string
}

type checkpointFileData struct {
	Data     checkpointData
	Checksum uint64
}

type NetDeviceInfo struct {
	Name       string
	Vendor     string
	Product    string
	PciAddress string
}

func getCheckpointData() (checkpointData, error) {
	cpd := &checkpointFileData{}
	rawBytes, err := ioutil.ReadFile(CheckPointfile)
	if err != nil {
		return checkpointData{}, fmt.Errorf("getPodEntries: error reading file %s\n%v\n", CheckPointfile, err)
	}

	if err = json.Unmarshal(rawBytes, cpd); err != nil {
		return checkpointData{}, fmt.Errorf("getPodEntries: error unmarshalling raw bytes %v", err)
	}
	return cpd.Data, nil
}

func convertDeviceIDs(deviceIDs interface{}) []string {
	deviceIDsArray := []string{}
	switch v := deviceIDs.(type) {
	case []interface{}:
		for _, val := range v {
			deviceIDsArray = append(deviceIDsArray, val.(string))
		}
	case map[string]interface{}:
		for _, val := range v {
			for _, idVal := range val.([]interface{}) {
				deviceIDsArray = append(deviceIDsArray, idVal.(string))
			}
		}
	}
	return deviceIDsArray
}

func GetPodResourceMap(pod *v1.Pod) (map[string][]string, error) {
	podID := string(pod.UID)
	log.Printf("GetPodDeviceIDs: %s (%s)\n", pod.GetName(), podID)

	resourceMap := make(map[string][]string)
	if podID == "" {
		return resourceMap, fmt.Errorf("GetPodResourceMap: invalid Pod cannot be empty")
	}
	cpd, err := getCheckpointData()
	if err != nil {
		return resourceMap, err
	}
	for _, pod := range cpd.PodDeviceEntries {
		if pod.PodUID == podID {
			entry, ok := resourceMap[pod.ResourceName]
			deviceIDs := convertDeviceIDs(pod.DeviceIDs)
			if ok {
				// already exists; append to it
				entry = append(entry, deviceIDs...)
			} else {
				// new entry
				resourceMap[pod.ResourceName] = deviceIDs
			}
		}
	}
	return resourceMap, nil
}

// GetDeviceMap returns a map from network address to NIC device
func GetDeviceMap(resourceMap map[string][]string, resourceName string) map[string]string {
	deviceMap := make(map[string]string)
	log.Printf("GetDeviceMap of %s\n", resourceName)
	nameNetMap := GetNameNetMap()
	log.Printf("resource map: %v\n", resourceMap)
	log.Printf("nameNetMap map: %v\n", nameNetMap)

	if deviceIDs, exist := resourceMap[resourceName]; exist {
		for _, deviceID := range deviceIDs {
			masterName, err := GetPfName(deviceID)
			if err == nil {
				if netAddress, exist := nameNetMap[masterName]; exist {
					deviceMap[netAddress] = deviceID
				}
			}
		}
	}
	return deviceMap
}

// reference: github.com/k8snetworkplumbingwg/sriov-cni/pkg/utils
// GetPfName returns PF net device name of a given VF pci address
func GetPfName(vf string) (string, error) {
	pfSymLink := filepath.Join(SysBusPci, vf, "physfn", "net")
	_, err := os.Lstat(pfSymLink)
	if err != nil {
		return "", err
	}

	files, err := ioutil.ReadDir(pfSymLink)
	if err != nil {
		return "", err
	}

	if len(files) < 1 {
		return "", fmt.Errorf("PF network device not found")
	}

	return strings.TrimSpace(files[0].Name()), nil
}

// GetTargetNetworks returns considering network information (existing PCI address)
func GetTargetNetworks() []NetDeviceInfo {
	netDevices := []NetDeviceInfo{}
	pci, err := ghw.PCI()
	if err != nil {
		log.Printf("cannot get PCI info: %v", err)
	}
	devices := pci.ListDevices()
	for _, device := range devices {
		devClass, err := strconv.ParseInt(device.Class.ID, 16, 64)
		if err != nil {
			// cannnot convert
			continue
		}
		if devClass != netClass {
			// not network device
			continue
		}
		pciAddress := device.Address
		devNames, err := getNetNames(pciAddress)
		if err != nil {
			// cannot get device name
			continue
		}
		for _, devName := range devNames {
			vendorID := device.Vendor.ID
			productID := device.Product.ID

			netDevice := NetDeviceInfo{
				Name:       devName,
				Vendor:     vendorID,
				Product:    productID,
				PciAddress: pciAddress,
			}
			netDevices = append(netDevices, netDevice)
		}
	}
	return netDevices
}

// getVirtioNetNames returns list of net name in virtio folder
func getVirtioNetNames(topDir string) ([]string, error) {
	names := []string{}
	fileList, err := ioutil.ReadDir(topDir)
	if err != nil {
		return names, fmt.Errorf("failed to read directory %s: %q", topDir, err)
	}
	for _, f := range fileList {
		fileName := f.Name()
		if strings.Contains(fileName, "virtio") {
			virtioDir := filepath.Join(topDir, fileName)
			netDir := filepath.Join(virtioDir, "net")
			if _, err := os.Lstat(netDir); err != nil {
				// net folder not exist
				continue
			}
			fileList, err := ioutil.ReadDir(netDir)
			if err != nil {
				continue
			}
			for _, f := range fileList {
				names = append(names, f.Name())
			}
		}
	}
	if len(names) > 0 {
		return names, nil
	}
	return names, fmt.Errorf("no net or virtio folder from %s", topDir)
}

// getNetNames returns list of net name from pci address
func getNetNames(pciAddr string) ([]string, error) {
	netDir := filepath.Join(SysBusPci, pciAddr, "net")
	names := []string{}
	if _, err := os.Lstat(netDir); err != nil {
		topDir := filepath.Join(SysBusPci, pciAddr)
		return getVirtioNetNames(topDir)
	}
	fileList, err := ioutil.ReadDir(netDir)
	if err != nil {
		return names, fmt.Errorf("failed to read net directory %s: %q", netDir, err)
	}
	for _, f := range fileList {
		names = append(names, f.Name())
	}
	if len(names) > 0 {
		return names, nil
	}
	return names, fmt.Errorf("no net in %s", netDir)
}
