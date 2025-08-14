/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"fmt"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

// RouteHandler handles routes according to CIDR by connecting DaemonConnector
type RouteHandler struct {
	DaemonConnector
	*DaemonCacheHandler
}

// AddRoutes add corresponding routes of CIDR
// success: all routes is properly updated
func (h *RouteHandler) AddRoutes(cidrSpec multinicv1.CIDRSpec, entries []multinicv1.CIDREntry, hostInterfaceInfoMap map[string]map[int]multinicv1.HostInterfaceInfo, forceDelete bool) (success bool, noConnection bool) {
	success = true
	noConnection = false
	daemonCache := h.DaemonCacheHandler.ListCache()
	for hostName, daemon := range daemonCache {
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
func (h *RouteHandler) AddRoutesToHost(cidrSpec multinicv1.CIDRSpec, hostName string, daemon DaemonPod, entries []multinicv1.CIDREntry, hostInterfaceInfoMap map[string]map[int]multinicv1.HostInterfaceInfo, forceDelete bool) (bool, bool) {
	_, err := h.DaemonCacheHandler.GetCache(hostName)
	if err != nil {
		vars.CIDRLog.V(6).Info(fmt.Sprintf("fail to apply L3config %s to %s: %v", cidrSpec.Config.Name, hostName, err))
		// no change, connecion failed
		return false, true
	}
	change := true
	mainSrcHostIP := daemon.HostIP
	routes := []HostRoute{}
	for _, entry := range entries {
		interfaceIndex := entry.InterfaceIndex
		for _, host := range entry.Hosts {
			destHostName := host.HostName
			destDaemon, err := h.DaemonCacheHandler.GetCache(destHostName)
			if err != nil {
				vars.CIDRLog.V(6).Info(fmt.Sprintf("AddRoutesToHost %s failed: %v", destHostName, err))
				continue
			}
			mainDestHostIP := destDaemon.HostIP
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
		vars.CIDRLog.V(6).Info(fmt.Sprintf("fail to apply L3config %s to %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	} else {
		vars.CIDRLog.V(6).Info(fmt.Sprintf("Apply L3config %s to %s: %v", cidrSpec.Config.Name, hostName, res.Success))
	}
	if err != nil || !res.Success {
		change = false
	}
	return change, res.Message == vars.ConnectionRefusedError
}

// DeleteRoutes deletes corresponding routes of CIDR
func (h *RouteHandler) DeleteRoutes(cidrSpec multinicv1.CIDRSpec) {
	daemonCache := h.DaemonCacheHandler.ListCache()
	for hostName, daemon := range daemonCache {
		podAddress := GetDaemonAddressByPod(daemon)
		res, err := h.DaemonConnector.DeleteL3Config(podAddress, cidrSpec.Config.Name, cidrSpec.Config.Subnet)
		vars.CIDRLog.V(6).Info(fmt.Sprintf("Delete L3config %s from %s: %v (%v)", cidrSpec.Config.Name, hostName, res, err))
	}
}
