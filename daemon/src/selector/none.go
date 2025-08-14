/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package selector

import (
	"log"
	"math"
	"sort"

	gnet "github.com/jaypipes/ghw/pkg/net"
)

type DefaultSelector struct{}

func getMasterNames(ifaceNameMap map[string]string, req NICSelectRequest) []string {
	masters := []string{}
	if req.MasterNetAddrs == nil || len(req.MasterNetAddrs) == 0 {
		for _, master := range ifaceNameMap {
			masters = append(masters, master)
		}
	} else {
		for _, netAddress := range req.MasterNetAddrs {
			master := ifaceNameMap[netAddress]
			masters = append(masters, master)
		}
	}
	return masters
}

func getDeviceIDs(deviceMap map[string]*gnet.NIC, req NICSelectRequest) []string {
	deviceIDs := []string{}
	if req.MasterNetAddrs == nil || len(req.MasterNetAddrs) == 0 {
		for _, nic := range deviceMap {
			deviceIDs = append(deviceIDs, *nic.PCIAddress)
		}
	} else {
		for _, netAddress := range req.MasterNetAddrs {
			nic := deviceMap[netAddress]
			deviceIDs = append(deviceIDs, *nic.PCIAddress)
		}
	}
	return deviceIDs
}

// DefaultSelector simply selects interface in order
func (DefaultSelector) Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string, resourceMap map[string][]string) []string {
	selectedMaster := []string{}
	maxSize := req.NicSet.NumOfInterfaces
	fixedSet := req.NicSet.InterfaceNames
	if len(fixedSet) > 0 {
		// use defined fixset
		for _, devName := range fixedSet {
			if netAddress, exists := nameNetMap[devName]; exists {
				selectedMaster = append(selectedMaster, netAddress)
			}
		}
	} else {
		// use defined master network addresses
		for _, netAddress := range req.MasterNetAddrs {
			log.Printf("select by net %s", netAddress)
			selectedMaster = append(selectedMaster, netAddress)
		}
	}
	if len(selectedMaster) == 0 {
		// apply all network addresses
		for netAddress := range interfaceNameMap {
			log.Printf("select %s", netAddress)
			selectedMaster = append(selectedMaster, netAddress)
		}
		sort.Strings(selectedMaster)
	}
	if maxSize > 0 {
		maxSize = int(math.Min(float64(len(selectedMaster)), float64(maxSize)))
	} else {
		maxSize = len(selectedMaster)
	}
	return selectedMaster[0:maxSize]
}
