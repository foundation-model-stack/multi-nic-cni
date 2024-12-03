/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package selector

import (
	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
	"github.com/foundation-model-stack/multi-nic-cni/daemon/iface"

	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// For NIC Selection
type NICSelectRequest struct {
	PodName          string   `json:"pod"`
	PodNamespace     string   `json:"namespace"`
	HostName         string   `json:"host"`
	NetAttachDefName string   `json:"def"`
	MasterNetAddrs   []string `json:"masterNets"`
	NicSet           NicArgs  `json:"args"`
}

// NicArgs defines additional specification in pod annotation
type NicArgs struct {
	NumOfInterfaces int      `json:"nics,omitempty"`
	InterfaceNames  []string `json:"masters,omitempty"`
	Target          string   `json:"target,omitempty"`
	DevClass        string   `json:"class,omitempty"`
}

type NICSelectResponse struct {
	DeviceIDs []string `json:"deviceIDs"`
	Masters   []string `json:"masters"`
}

type Selector interface {
	// return list of network addresses
	Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string, resourceMap map[string][]string) []string
}

var MultinicnetHandler *backend.MultiNicNetworkHandler
var NetAttachDefHandler *backend.NetAttachDefHandler
var K8sClientset *kubernetes.Clientset
var DeviceClassHandler *backend.DeviceClassHandler
var GPUIdBusIdMap = GetGPUIDMap()
var TopologyFilePath = defaultTopologyFilePath
var NumaAwareSelectorInstance = InitNumaAwareSelector(TopologyFilePath, GPUIdBusIdMap)

func isEmptyDeviceIDs(selectedDeviceIDs []string) bool {
	for _, deviceID := range selectedDeviceIDs {
		if deviceID != "" {
			return false
		}
	}
	return true
}

func getDefaultResponse(req NICSelectRequest, masterNameMap map[string]string, nameNetMap map[string]string, deviceMap map[string]string, resourceMap map[string][]string, podDeviceIDs []string) NICSelectResponse {
	selector := DefaultSelector{}
	selectedMasterNetAddrs := selector.Select(req, masterNameMap, nameNetMap, resourceMap)
	selectedMasters := []string{}
	selectedDeviceIDs := []string{}
	for _, netAddress := range selectedMasterNetAddrs {
		deviceID := deviceMap[netAddress]
		master := masterNameMap[netAddress]
		selectedDeviceIDs = append(selectedDeviceIDs, deviceID)
		selectedMasters = append(selectedMasters, master)
	}
	// allow empty master name for the devices that has been assigned to the pod
	if len(podDeviceIDs) > 0 && isEmptyDeviceIDs(selectedDeviceIDs) {
		selectedMasters = []string{}
		selectedDeviceIDs = []string{}
		for _, deviceID := range podDeviceIDs {
			selectedDeviceIDs = append(selectedDeviceIDs, deviceID)
			selectedMasters = append(selectedMasters, "")
		}
	}

	return NICSelectResponse{
		DeviceIDs: selectedDeviceIDs,
		Masters:   selectedMasters,
	}
}

func Select(req NICSelectRequest) NICSelectResponse {
	deviceMap := make(map[string]string)
	resourceMap := make(map[string][]string)
	podDeviceIDs := []string{}

	pod, err := K8sClientset.CoreV1().Pods(req.PodNamespace).Get(context.TODO(), req.PodName, metav1.GetOptions{})
	if err == nil {
		resourceMap, err = iface.GetPodResourceMap(pod)
		if err == nil {
			log.Printf("resourceMap of %s: %v\n", pod.UID, resourceMap)
			resourceNames := NetAttachDefHandler.GetResourceNames(req.NetAttachDefName, req.PodNamespace)
			if len(resourceNames) > 0 {
				log.Printf("resource map: %v\n", resourceMap)
				for _, resourceName := range resourceNames {
					if deviceIDs, exist := resourceMap[resourceName]; exist {
						podDeviceIDs = append(podDeviceIDs, deviceIDs...)
						deviceMapPerResource := iface.GetDeviceMap(deviceIDs)
						log.Printf("deviceMap of %s (%v): %v\n", resourceName, deviceIDs, deviceMapPerResource)
						for netAddress, deviceID := range deviceMapPerResource {
							deviceMap[netAddress] = deviceID
						}
					}
				}
			}
		} else {
			log.Printf("Cannot get pod resource map: %v\n", err)
		}
	} else {
		log.Printf("Cannot get pod: %v\n", err)
	}

	masterNameMap := iface.GetInterfaceNameMap()
	log.Printf("master name map: %v\n", masterNameMap)
	nameNetMap := iface.GetNameNetMap()
	netSpec, err := MultinicnetHandler.Get(req.NetAttachDefName, req.PodNamespace)
	if err != nil {
		// FIXME: failed to get network spec (use default policy): the server could not find the requested resource
		log.Printf("failed to get network spec (use default policy): %v\n", err)
		defaultMasterNameMap := make(map[string]string)
		if len(deviceMap) > 0 {
			// filter only existing deviceID
			for netAddress, deviceID := range deviceMap {
				if master, exist := masterNameMap[netAddress][deviceID]; exist {
					defaultMasterNameMap[netAddress] = master
				}
			}
		} else {
			for netAddress, masterDeviceMap := range masterNameMap {
				for _, master := range masterDeviceMap {
					defaultMasterNameMap[netAddress] = master
				}
			}
		}
		return getDefaultResponse(req, defaultMasterNameMap, nameNetMap, deviceMap, resourceMap, podDeviceIDs)
	}
	policy := netSpec.Policy

	filteredMasterNameMap := make(map[string]string)
	if len(deviceMap) > 0 {
		// filter only existing deviceID
		for netAddress, deviceID := range deviceMap {
			if master, exist := masterNameMap[netAddress][deviceID]; exist {
				filteredMasterNameMap[netAddress] = master
			}
		}
	} else {
		for netAddress, masterDeviceMap := range masterNameMap {
			for _, master := range masterDeviceMap {
				filteredMasterNameMap[netAddress] = master
			}
		}
	}

	var selector Selector
	strategy := Strategy(policy.Strategy)
	switch strategy {
	case None:
		selector = DefaultSelector{}
	case CostOpt:
		selector = CostOptSelector{}
	case PerfOpt:
		selector = PerfOptSelector{}
	case DevClass:
		selector = DevClassSelector{}
	case Topology:
		selector = NumaAwareSelectorInstance.GetCopy()
	default:
		selector = DefaultSelector{}
	}
	selectedMasterNetAddrs := selector.Select(req, filteredMasterNameMap, nameNetMap, resourceMap)
	selectedMasters := []string{}
	selectedDeviceIDs := []string{}
	log.Printf("masterNets %v, %v, %v\n", selectedMasterNetAddrs, filteredMasterNameMap, nameNetMap)
	for _, netAddress := range selectedMasterNetAddrs {
		deviceID := deviceMap[netAddress]
		if master, ok := filteredMasterNameMap[netAddress]; ok && master != "" {
			log.Printf("masterNets %s,%s\n", deviceID, master)
			selectedDeviceIDs = append(selectedDeviceIDs, deviceID)
			selectedMasters = append(selectedMasters, master)
		}
	}
	// allow empty master name for the devices that has been assigned to the pod
	if len(podDeviceIDs) > 0 && isEmptyDeviceIDs(selectedDeviceIDs) {
		selectedMasters = []string{}
		selectedDeviceIDs = []string{}
		for _, deviceID := range podDeviceIDs {
			selectedDeviceIDs = append(selectedDeviceIDs, deviceID)
			selectedMasters = append(selectedMasters, "")
		}
	}

	return NICSelectResponse{
		DeviceIDs: selectedDeviceIDs,
		Masters:   selectedMasters,
	}
}
