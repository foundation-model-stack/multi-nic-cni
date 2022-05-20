/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"fmt"

	"github.com/go-logr/logr"
	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

// RouteHandler handles routes according to CIDR by connecting DaemonConnector
type RouteHandler struct {
	DaemonConnector
	Log logr.Logger
}

// AddRoutes add corresponding routes of CIDR
func (h *RouteHandler) AddRoutes(entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo, suppressWarning bool) (bool, map[string][]string) {
	failMap := make(map[string][]string)
	routeChange := false
	podMap, err := h.DaemonConnector.GetDaemonHostMap()
	if err != nil {
		h.Log.Error(err, "")
	}

	for hostName, daemon := range podMap {
		change, failList := h.AddRoutesToHost(hostName, daemon, podMap, entries, hostInterfaceInfoMap, suppressWarning)
		if change {
			routeChange = true
		}
		failMap[hostName] = failList
	}
	return routeChange, failMap
}

// AddRoutesToHost add route to a specific host
func (h *RouteHandler) AddRoutesToHost(hostName string, daemon corev1.Pod, podMap map[string]corev1.Pod, entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo, suppressWarning bool) (bool, []string) {
	var failList []string
	change := false
	mainSrcHostIP := daemon.Status.HostIP
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
					res, err := h.DaemonConnector.AddRoute(daemon, net, via, iface)
					if err != nil || !res.Success {
						failItem := via + "-" + ifaceInfo.InterfaceName
						failList = append(failList, failItem)
					}
					if res.Success {
						change = true
					}
					if res.Success || !suppressWarning {
						h.Log.Info(fmt.Sprintf("Add route %s %s via %s: %s, %v", hostName, iface, via, res.Message, err))
					}
				}
			} else {
				// delete route to its pod CIDR if exists
				if ifaceInfo, exist := hostInterfaceInfoMap[hostName][interfaceIndex]; exist {
					iface := ifaceInfo.InterfaceName
					via := hostInterfaceInfoMap[destHostName][interfaceIndex].HostIP
					res, err := h.DaemonConnector.DeleteRoute(daemon, net, via, iface)
					if !suppressWarning {
						h.Log.Info(fmt.Sprintf("Delete route %s %s via %s: %s, %v", hostName, iface, via, res.Message, err))
					}
				}
			}
		}
	}
	return change, failList
}

// DeleteRoutes deletes corresponding routes of CIDR
func (h *RouteHandler) DeleteRoutes(entries []netcogadvisoriov1.CIDREntry, hostInterfaceInfoMap map[string]map[int]netcogadvisoriov1.HostInterfaceInfo) map[string][]string {
	failMap := make(map[string][]string)
	podMap, err := h.DaemonConnector.GetDaemonHostMap()
	if err != nil {
		h.Log.Error(err, "")
	}

	for hostName, daemon := range podMap {
		var failList []string
		mainSrcHostIP := daemon.Status.HostIP
		for _, entry := range entries {
			interfaceIndex := entry.InterfaceIndex
			for _, host := range entry.Hosts {
				destHostName := host.HostName
				mainDestHostIP := podMap[destHostName].Status.HostIP
				if mainDestHostIP != mainSrcHostIP {
					if ifaceInfo, exist := hostInterfaceInfoMap[hostName][interfaceIndex]; exist {
						iface := ifaceInfo.InterfaceName
						via := hostInterfaceInfoMap[destHostName][interfaceIndex].HostIP
						net := host.PodCIDR
						res, err := h.DaemonConnector.DeleteRoute(daemon, net, via, iface)
						if err != nil || !res.Success {
							failItem := via + "-" + ifaceInfo.InterfaceName
							failList = append(failList, failItem)
						}
					}
				}
			}
		}
		failMap[hostName] = failList
		h.Log.Info(fmt.Sprintf("Delete routes to %s", hostName))
	}
	return failMap
}

// DeleteRoute deletes specific route on specific host (referred by IPPool)
func (h *RouteHandler) DeleteRoute(daemon corev1.Pod, net string, via string, iface string) (RouteUpdateResponse, error) {
	return h.DaemonConnector.DeleteRoute(daemon, net, via, iface)
}
