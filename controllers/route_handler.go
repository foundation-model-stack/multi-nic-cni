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
func (h *RouteHandler) AddRoutes(cidrSpec netcogadvisoriov1.CIDRSpec, entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo) bool {
	routeChange := false
	podMap, err := h.DaemonConnector.GetDaemonHostMap()
	if err != nil {
		h.Log.Error(err, "")
	}

	for hostName, daemon := range podMap {
		change := h.AddRoutesToHost(cidrSpec, hostName, daemon, podMap, entries, hostInterfaceInfoMap)
		if change {
			routeChange = true
		}
	}
	return routeChange
}

// AddRoutesToHost add route to a specific host
func (h *RouteHandler) AddRoutesToHost(cidrSpec netcogadvisoriov1.CIDRSpec, hostName string, daemon corev1.Pod, podMap map[string]corev1.Pod, entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo) bool {
	change := true
	mainSrcHostIP := daemon.Status.HostIP
	routes := []HostRoute{}
	for _, entry := range entries {
		interfaceIndex := entry.InterfaceIndex
		for _, host := range entry.Hosts {
			destHostName := host.HostName
			mainDestHostIP := podMap[destHostName].Status.HostIP
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
	res, err := h.DaemonConnector.ApplyL3Config(daemon, cidrSpec.Config.Name, cidrSpec.Config.Subnet, routes)
	h.Log.Info(fmt.Sprintf("Apply L3config %s to %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	if err != nil || !res.Success {
		change = false
	}
	return change
}

// DeleteRoutes deletes corresponding routes of CIDR
func (h *RouteHandler) DeleteRoutes(cidrSpec netcogadvisoriov1.CIDRSpec) {
	podMap, err := h.DaemonConnector.GetDaemonHostMap()
	if err != nil {
		h.Log.Error(err, "")
	}

	for hostName, daemon := range podMap {
		res, err := h.DaemonConnector.DeleteL3Config(daemon, cidrSpec.Config.Name, cidrSpec.Config.Subnet)
		h.Log.Info(fmt.Sprintf("Delete L3config %s from %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	}
}
