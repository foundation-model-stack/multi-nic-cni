/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// RouteHandler handles routes according to CIDR by connecting DaemonConnector
type RouteHandler struct {
	DaemonConnector
	Log logr.Logger
}

// AddRoutes add corresponding routes of CIDR
// success: all routes is properly updated
func (h *RouteHandler) AddRoutes(cidrSpec netcogadvisoriov1.CIDRSpec, entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo, forceDelete bool) (success bool, noConnection bool) {
	success = true
	noConnection = false
	for hostName, daemon := range DaemonCache {
		if _, ok := hostInterfaceInfoMap[hostName]; ok {
			change, connectFail := h.AddRoutesToHost(cidrSpec, hostName, daemon, entries, hostInterfaceInfoMap, forceDelete)
			if !change || connectFail {
				success = false
			}
			if connectFail {
				noConnection = true
			}
		}
	}
	return success, noConnection
}

// AddRoutesToHost add route to a specific host
func (h *RouteHandler) AddRoutesToHost(cidrSpec netcogadvisoriov1.CIDRSpec, hostName string, daemon corev1.Pod, entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo, forceDelete bool) (bool, bool) {
	change := true
	mainSrcHostIP := daemon.Status.HostIP
	routes := []HostRoute{}
	for _, entry := range entries {
		interfaceIndex := entry.InterfaceIndex
		for _, host := range entry.Hosts {
			destHostName := host.HostName
			mainDestHostIP := DaemonCache[destHostName].Status.HostIP
			net := host.PodCIDR
			if mainDestHostIP != mainSrcHostIP {
				if ifaceInfo, exist := hostInterfaceInfoMap[hostName][interfaceIndex]; exist {
					iface := ifaceInfo.InterfaceName
					via := hostInterfaceInfoMap[destHostName][interfaceIndex].HostIP
					route := HostRoute{
						Subnet:        net,
						NextHop:       via,
						InterfaceName: iface,
					}
					routes = append(routes, route)
				}
			}
		}
	}
	podAddress := GetDaemonAddressByPod(daemon)
	res, err := h.DaemonConnector.ApplyL3Config(podAddress, cidrSpec.Config.Name, cidrSpec.Config.Subnet, routes, forceDelete)
	if err != nil {
		h.Log.Info(fmt.Sprintf("fail to apply L3config %s to %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	} else {
		h.Log.Info(fmt.Sprintf("Apply L3config %s to %s: %v", cidrSpec.Config.Name, hostName, res.Success))
	}
	if err != nil || !res.Success {
		change = false
	}
	return change, res.Message == CONNECTION_REFUSED
}

// DeleteRoutes deletes corresponding routes of CIDR
func (h *RouteHandler) DeleteRoutes(cidrSpec netcogadvisoriov1.CIDRSpec) {
	for hostName, daemon := range DaemonCache {
		podAddress := GetDaemonAddressByPod(daemon)
		res, err := h.DaemonConnector.DeleteL3Config(podAddress, cidrSpec.Config.Name, cidrSpec.Config.Subnet)
		h.Log.Info(fmt.Sprintf("Delete L3config %s from %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	}
}
