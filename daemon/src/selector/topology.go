/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package selector

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	gonvml "github.com/NVIDIA/gpu-monitoring-tools/bindings/go/nvml"
	"github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
)

var (
	GPUResourceName         = "nvidia.com/gpu"
	defaultTopologyFilePath = "/var/run/nvidia-topologyd/virtualTopology.xml"
	deviceDir               = "/sys/devices"
)

type NcclTopolgy struct {
	XMLName xml.Name `xml:"system"`
	CPUs    []CPUTag `xml:"cpu"`
}

type CPUTag struct {
	XMLName xml.Name `xml:"cpu"`
	NumaId  string   `xml:"numaid,attr"`
	PCIs    []PCITag `xml:"pci"`
}

type PCITag struct {
	XMLName   xml.Name `xml:"pci"`
	BusId     string   `xml:"busid,attr"`
	Class     string   `xml:"class,attr"`
	Vendor    string   `xml:"vendor,attr"`
	Device    string   `xml:"device,attr"`
	SubVendor string   `xml:"subsystem_vendor,attr"`
	SubDevice string   `xml:"subsystem_device,attr"`
	LinkSpeed string   `xml:"link_speed,attr"`
	LinkWidth string   `xml:"link_width,attr"`
	PCIs      []PCITag `xml:"pci"`
}

type NumaAwareSelector struct {
	NcclTopolgy
	gpuIDBusMap map[string]string
	NumaMap     map[string]string
}

func InitNumaAwareSelector(topologyFilePath string, gpuIdBusIdMap map[string]string) *NumaAwareSelector {
	xmlFile, err := os.Open(topologyFilePath)
	var topology NcclTopolgy
	numaMap := make(map[string]string)
	if err == nil {
		defer xmlFile.Close()
		byteValue, err := ioutil.ReadAll(xmlFile)
		if err == nil {
			err = xml.Unmarshal(byteValue, &topology)
		}
	}
	numaMap = getNumaMap(topology)
	log.Printf("InitNumaAwareSelector with %d numa nodes\n", len(numaMap))
	return &NumaAwareSelector{
		NcclTopolgy: topology,
		gpuIDBusMap: gpuIdBusIdMap,
		NumaMap:     numaMap,
	}
}

func GetGPUIDMap() (gpuIdBusIdMap map[string]string) {
	gpuIdBusIdMap = make(map[string]string)
	err := gonvml.Init()
	if err != nil {
		return
	}
	defer gonvml.Shutdown()
	count, err := gonvml.GetDeviceCount()
	if err != nil {
		return
	}
	for i := 0; i < int(count); i++ {
		device, err := gonvml.NewDevice(uint(i))
		if err != nil {
			continue
		}
		gpuIdBusIdMap[device.UUID] = device.PCI.BusID
	}
	log.Printf("GetGPUIDMap: %v\n", gpuIdBusIdMap)
	return
}

func getNumaMap(topology NcclTopolgy) map[string]string {
	numaMap := make(map[string]string)
	if len(topology.CPUs) == 0 {
		return getNumaMapFromSysfs()
	}
	for _, cpu := range topology.CPUs {
		numaId := cpu.NumaId
		for _, pci := range cpu.PCIs {
			numaMap[pci.BusId] = numaId
			for _, subpci := range pci.PCIs {
				numaMap[subpci.BusId] = numaId
			}
		}
	}
	return numaMap
}

// getNumaMapFromSysfs get map from pci_address from /sys/devices/pci*/<pci_address>/numa_node
func getNumaMapFromSysfs() (numaMap map[string]string) {
	numaMap = make(map[string]string)
	err := filepath.Walk(deviceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, "/numa_node") {
			numaNodePath := filepath.Dir(path)
			directoryName := filepath.Base(filepath.Dir(numaNodePath))
			numaNodeData, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			numaNode := strings.TrimSpace(string(numaNodeData))
			numaMap[directoryName] = numaNode
		}
		return nil
	})
	if err != nil {
		log.Printf("failed to getNumaMapFromSysfs: %v\n", err)
	}
	return
}

// NumaAwareSelector selects considering Numa of allocatedPCI Address
func (s *NumaAwareSelector) Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string, resourceMap map[string][]string) []string {
	selectedMaster := []string{}
	maxSize := req.NicSet.NumOfInterfaces
	// use defined master network addresses
	for _, netAddress := range req.MasterNetAddrs {
		var ok bool
		if _, ok = interfaceNameMap[netAddress]; ok {
			selectedMaster = append(selectedMaster, netAddress)
		}
		log.Printf("select by net %s: valid=%v", netAddress, ok)
	}
	if len(selectedMaster) == 0 {
		// apply all network addresses
		for netAddress, _ := range interfaceNameMap {
			log.Printf("select %s", netAddress)
			selectedMaster = append(selectedMaster, netAddress)
		}
	}
	if maxSize > 0 && maxSize < len(selectedMaster) {
		maxSize = int(math.Min(float64(len(selectedMaster)), float64(maxSize)))
		// must be sorted and selected
		sortedSelectedMaster := s.SortByNumaAware(selectedMaster, interfaceNameMap, resourceMap)
		if len(sortedSelectedMaster) != len(selectedMaster) {
			log.Printf("sorted value is not equal to origin: %d != %d, use origin", len(sortedSelectedMaster), len(selectedMaster))
		} else {
			selectedMaster = sortedSelectedMaster
		}
	} else {
		maxSize = len(selectedMaster)
	}
	return selectedMaster[0:maxSize]
}

func (s *NumaAwareSelector) SortByNumaAware(selectedMaster []string, interfaceNameMap map[string]string, resourceMap map[string][]string) []string {
	if gpuIds, ok := resourceMap[GPUResourceName]; !ok || len(s.NumaMap) == 0 {
		// cannot sort value
		log.Printf("cannot sort by numa node: resourceMap=%v, NumaMap length=%d", ok, len(s.NumaMap))
		return selectedMaster
	} else {
		numaPriority := make(map[string]int)
		for _, gpuId := range gpuIds {
			// map gpuId to pciAddress
			pciAddress := s.gpuIDBusMap[gpuId]
			numaId := s.NumaMap[pciAddress]
			if priority, ok := numaPriority[numaId]; ok {
				numaPriority[numaId] = priority + 1
			} else {
				numaPriority[numaId] = 1
			}
		}
		nicPriority := make(map[string]int)
		interfaceMap := iface.GetInterfaceInfoCache()
		for _, masterNetAddr := range selectedMaster {
			if devName, ok := interfaceNameMap[masterNetAddr]; ok {
				if info, ok := interfaceMap[devName]; ok {
					log.Printf("info: %v", info)
					nicPciAddress := info.PciAddress
					if numaId, ok := s.NumaMap[nicPciAddress]; ok {
						if priority, ok := numaPriority[numaId]; ok {
							nicPriority[masterNetAddr] = priority
						} else {
							nicPriority[masterNetAddr] = 0
						}
					}
				}
			}
		}
		return getSortedKeyByMap(nicPriority)
	}
}

func (s *NumaAwareSelector) GetCopy() *NumaAwareSelector {
	return &NumaAwareSelector{
		NcclTopolgy: s.NcclTopolgy,
		gpuIDBusMap: s.gpuIDBusMap,
		NumaMap:     s.NumaMap,
	}
}
