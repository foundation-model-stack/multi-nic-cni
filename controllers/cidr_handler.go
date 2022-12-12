/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"k8s.io/client-go/kubernetes"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/compute"
	"github.com/foundation-model-stack/multi-nic-cni/plugin"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sort"

	"errors"
	"sync"
)

const (
	BASE_IPAM_TYPE   = "multi-nic-ipam"
	POD_STATUS_FIELD = "status.phase"
	POD_STATUS_VALUE = "Running"
)

// CIDRHandler handles CIDR object
// - general handling: Get, List, Delete
// - compute VLAN CIDR and create CIDR
type CIDRHandler struct {
	client.Client
	*kubernetes.Clientset
	compute.CIDRCompute
	*HostInterfaceHandler
	*IPPoolHandler
	*MultiNicNetworkHandler
	sync.Mutex
	Log logr.Logger
	RouteHandler
	*SafeCache
}

func NewCIDRHandler(client client.Client, config *rest.Config, logger logr.Logger, ippoolLog logr.Logger, networkLog logr.Logger, hostInterfaceHandler *HostInterfaceHandler, daemonCache *DaemonCacheHandler) *CIDRHandler {
	clientset, _ := kubernetes.NewForConfig(config)
	cidrCompute := compute.CIDRCompute{}

	handler := &CIDRHandler{
		Client:               client,
		Clientset:            clientset,
		Log:                  logger,
		CIDRCompute:          cidrCompute,
		HostInterfaceHandler: hostInterfaceHandler,
		IPPoolHandler: &IPPoolHandler{
			Client:    client,
			Log:       ippoolLog,
			SafeCache: InitSafeCache(),
		},
		MultiNicNetworkHandler: &MultiNicNetworkHandler{
			Client: client,
			Log:    networkLog,
		},
		RouteHandler: RouteHandler{
			DaemonConnector: DaemonConnector{
				Clientset: clientset,
			},
			Log:                logger,
			DaemonCacheHandler: daemonCache,
		},
		SafeCache: InitSafeCache(),
	}
	return handler
}

// InitCustomCRCache inits existing list of IPPool and HostInterface to avoid unexpected recomputation of CIDR
func (h *CIDRHandler) InitCustomCRCache() {
	err := InitIppoolCache(h.IPPoolHandler)
	if err != nil {
		h.IPPoolHandler.Log.Info(fmt.Sprintf("Failed to InitIppoolCache: %v", err))
	}
	err = InitHostInterfaceCache(h.Clientset, h.HostInterfaceHandler, h.RouteHandler.DaemonCacheHandler)
	if err != nil {
		h.HostInterfaceHandler.Log.Info(fmt.Sprintf("Failed to InitHostInterfaceCache: %v", err))
	}
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// General handling: Get, List, Delete
//
/////////////////////////////////////////////////////////////////////////////////////////////////////////

// GetCIDR gets CIDR from CIDR name
func (h *CIDRHandler) GetCIDR(name string) (*multinicv1.CIDR, error) {
	instance := &multinicv1.CIDR{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, instance)
	return instance, err
}

// ListCIDR returns a map from CIDR name to instance
func (h *CIDRHandler) ListCIDR() (map[string]multinicv1.CIDR, error) {
	cidrList := &multinicv1.CIDRList{}
	err := h.Client.List(context.TODO(), cidrList)
	cidrSpecMap := make(map[string]multinicv1.CIDR)
	if err == nil {
		for _, cidr := range cidrList.Items {
			cidrName := cidr.GetName()
			cidrSpecMap[cidrName] = cidr
		}
	}
	return cidrSpecMap, err
}

// SyncAllPendingCustomCR sync CIDRs and IPPools corresponding to MultiNICNetwork
// note: CIDR name = NetworkAttachmentDefinition name
func (h *CIDRHandler) SyncAllPendingCustomCR(defHandler *plugin.NetAttachDefHandler) {
	h.InitCustomCRCache()
	cidrMap, err := h.ListCIDR()
	ippoolSnapshot := h.IPPoolHandler.ListCache()
	if err == nil {
		h.Log.Info(fmt.Sprintf("Checking %d cidrs", len(cidrMap)))
		for name, cidr := range cidrMap {
			_, err = h.MultiNicNetworkHandler.GetNetwork(name)
			if err != nil {
				// not found
				h.Log.Info(fmt.Sprintf("%v, delete pending resources (CIDR)", err))
				defHandler.Delete(name, metav1.NamespaceAll)
				h.DeleteCIDR(cidr)
			} else {
				h.CleanPendingIPPools(ippoolSnapshot, name, cidr.Spec)
				h.SetCache(name, cidr.Spec)
				excludes := compute.SortAddress(cidr.Spec.Config.ExcludeCIDRs)
				for _, entry := range cidr.Spec.CIDRs {
					for _, host := range entry.Hosts {
						ippoolName := h.GetIPPoolName(name, host.PodCIDR)
						if _, found := ippoolSnapshot[ippoolName]; !found {
							h.UpdateIPPool(name, host.PodCIDR, entry.VlanCIDR, host.HostName, host.InterfaceName, excludes)
						}
					}
				}
			}
		}
		h.SyncIPPoolWithActivePods(cidrMap, ippoolSnapshot)
	} else {
		h.Log.Info(fmt.Sprintf("Failed to list cidr: %v", err))
	}
}

// DeleteCIDR deletes corresponding routes and IPPools, then deletes CIDR
func (h *CIDRHandler) DeleteCIDR(cidr multinicv1.CIDR) error {
	errorMsg := ""
	// delete corresponding routes
	h.deleteRoutesFromCIDR(cidr.Spec)
	// delete corresponding IPPools
	for _, entry := range cidr.Spec.CIDRs {
		for _, host := range entry.Hosts {
			podCIDR := host.PodCIDR
			err := h.IPPoolHandler.DeleteIPPool(cidr.GetName(), podCIDR)
			if err != nil {
				errorMsg = errorMsg + fmt.Sprintf("%v,", err)
			}
		}
	}
	instance, err := h.GetCIDR(cidr.GetName())
	if err == nil {
		err = h.Client.Delete(context.Background(), instance)
	}
	if err != nil {
		errorMsg = errorMsg + fmt.Sprintf("%v,", err)
	}
	if len(errorMsg) == 0 {
		return nil
	}
	return fmt.Errorf("%s", errorMsg)
}

// deleteRoutesFromCIDR deletes routes from CIDR
func (h *CIDRHandler) deleteRoutesFromCIDR(cidrInfo multinicv1.CIDRSpec) {
	if h.IsL3Mode(cidrInfo.Config) {
		h.RouteHandler.DeleteRoutes(cidrInfo)
	}
}

// GetAllNetAddrs returns all common network address from hiflist
func (h *CIDRHandler) GetAllNetAddrs() []string {
	netAddrSet := []string{}
	netAddressMap := make(map[string]bool)
	snapshot := h.HostInterfaceHandler.ListCache()
	for _, hif := range snapshot {
		for _, iface := range hif.Spec.Interfaces {
			netAddr := iface.NetAddress
			if _, exist := netAddressMap[netAddr]; !exist {
				netAddrSet = append(netAddrSet, netAddr)
				netAddressMap[netAddr] = true
			}
		}
	}
	return netAddrSet
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// Compute VLAN CIDR and create CIDR
//
/////////////////////////////////////////////////////////////////////////////////////////////////////////

// newCIDR returns new CIDR from PluginConfig
func (h *CIDRHandler) newCIDR(def multinicv1.PluginConfig, namespace string) (multinicv1.CIDRSpec, error) {
	entries := []multinicv1.CIDREntry{}
	masterIndex := int(0)
	// maxInterfaceIndex = 2^(interface bits) - 1
	maxInterfaceIndex := int(math.Pow(2, float64(def.InterfaceBlock)) - 1)
	// loop over defined network addresses
	for _, master := range def.MasterNetAddrs {
		vlanCIDR := ""
		// find available VLAN CIDR
		for vlanCIDR == "" {
			vlanInByte, err := h.CIDRCompute.ComputeNet(def.Subnet, masterIndex, def.InterfaceBlock)
			if err != nil {
				// invalid VLAN value (out of range), find next interface index
				masterIndex = masterIndex + 1
				continue
			}
			// check if computed vlan in exclude ranges
			tabu := h.CIDRCompute.CheckIfTabuIndex(def.Subnet, masterIndex, def.InterfaceBlock, def.ExcludeCIDRs)
			if !tabu {
				vlanCIDR = h.CIDRCompute.GetCIDRFromByte(vlanInByte, def.Subnet, def.InterfaceBlock)
				break
			}
			// if tabu, find next interface index
			masterIndex = masterIndex + 1
			if masterIndex > maxInterfaceIndex {
				return multinicv1.CIDRSpec{}, errors.New("wrong request (overflow interface index)")
			}
		}

		entry := multinicv1.CIDREntry{
			NetAddress:     master,
			InterfaceIndex: masterIndex,
			VlanCIDR:       vlanCIDR,
			Hosts:          []multinicv1.HostInterfaceInfo{},
		}
		entries = append(entries, entry)
		masterIndex = masterIndex + 1
	}

	cidrSpec := multinicv1.CIDRSpec{
		Config: def,
		CIDRs:  entries,
	}
	return cidrSpec, nil
}

// updateCIDR modifies existing CIDR from the new HostInterface information
func (h *CIDRHandler) UpdateCIDRs() bool {
	h.Log.Info("UpdateCIDRs")
	routeChange := false
	cidrSnapshot := h.ListCache()
	for _, cidr := range cidrSnapshot {
		change, err := h.UpdateCIDR(cidr, false)
		if err != nil {
			h.Log.Info(fmt.Sprintf("Fail to update CIDR: %v", err))
		} else if change {
			routeChange = true
		}
	}
	return routeChange
}

// NewCIDRWithNewConfig creates new CIDR by computing interface indexes from master networks
func (h *CIDRHandler) NewCIDRWithNewConfig(def multinicv1.PluginConfig, namespace string) (bool, error) {
	h.Log.Info("NewCIDRWithNewConfig")
	cidrSpec, err := h.newCIDR(def, namespace)
	if err != nil {
		return false, err
	}
	return h.UpdateCIDR(cidrSpec, true)
}

// updateEntries updates CIDR entry from current HostInterfaceCache
func (h *CIDRHandler) updateEntries(cidrSpec multinicv1.CIDRSpec, excludes []compute.IPValue, changed bool) (map[string]multinicv1.CIDREntry, bool) {
	hostInterfaceSnapshot := h.HostInterfaceHandler.ListCache()
	entries := cidrSpec.CIDRs
	entriesMap := make(map[string]multinicv1.CIDREntry)
	diedHost := make(map[string]interface{})
	for _, entry := range entries {
		var newHostList []multinicv1.HostInterfaceInfo
		for _, host := range entry.Hosts {
			if hif, exists := hostInterfaceSnapshot[host.HostName]; exists {
				// to not include hanging entry
				for _, iface := range hif.Spec.Interfaces {
					if iface.NetAddress == entry.NetAddress {
						newHostList = append(newHostList, host)
						break
					}
				}
			} else {
				if _, died := diedHost[host.HostName]; died {
					changed = true
				} else {
					if _, foundErr := h.Clientset.CoreV1().Nodes().Get(context.TODO(), host.HostName, metav1.GetOptions{}); foundErr != nil {
						// host died
						h.Log.Info(fmt.Sprintf("Host %s no longer exist, delete from entry of CIDR %s", host.HostName, cidrSpec.Config.Name))
						// host not exist anymore
						diedHost[host.HostName] = nil // applied to other vlan check
					} else {
						// use the previous value
						newHostList = append(newHostList, host)
					}
				}
			}
		}
		if len(entry.Hosts) != len(newHostList) || len(diedHost) > 0 {
			changed = true
		}
		if len(entry.Hosts) == 0 || len(newHostList) > 0 {
			// new or has some in  newHostList
			entry.Hosts = newHostList
			entriesMap[entry.NetAddress] = entry
		}
	}

	def := cidrSpec.Config

	// sort and convert exclude object to string
	excludesInStr := []string{}
	for _, exclude := range excludes {
		excludesInStr = append(excludesInStr, exclude.Address)
	}

	// maxHostIndex = 2^(host bits) - 1
	maxHostIndex := int(math.Pow(2, float64(def.HostBlock)) - 1)

	// compute host indexes over host interface list
	for _, hif := range hostInterfaceSnapshot {
		hostName := hif.Spec.HostName
		ifaces := hif.Spec.Interfaces

		// assign interface index to each host
		for _, iface := range ifaces {
			interfaceNetAddress := iface.NetAddress
			interfaceName := iface.InterfaceName
			hostIP := iface.HostIP
			success, entry := h.getInterfaceEntry(def, entriesMap, interfaceNetAddress)
			if !success {
				continue
			}
			vlanCIDR := entry.VlanCIDR
			existingHosts := entry.Hosts

			// check if host index computed before
			itemIndex := h.getHostIndex(existingHosts, hostName)
			if itemIndex == -1 {
				// compute new host index
				entry, changed = h.tryAddNewHost(existingHosts, entry, maxHostIndex, def, hostName, interfaceName, hostIP)
			} else {
				// refer to previous host index
				host := existingHosts[itemIndex]
				nodeIndex := existingHosts[itemIndex].HostIndex
				nodeBlock := def.HostBlock
				podInByte, err := h.CIDRCompute.ComputeNet(entry.VlanCIDR, host.HostIndex, def.HostBlock)
				if err != nil {
					// invalid pod VLAN
					// remove from existing list
					entry.Hosts = append(entry.Hosts[0:itemIndex], entry.Hosts[itemIndex+1:]...)
					// recompute host index
					entry, changed = h.tryAddNewHost(existingHosts, entry, maxHostIndex, def, hostName, interfaceName, hostIP)
				} else {
					// recheck is computed pod VLAN tabu
					tabu := h.CIDRCompute.CheckIfTabuIndex(vlanCIDR, nodeIndex, nodeBlock, excludesInStr)
					if !tabu {
						podCIDR := h.CIDRCompute.GetCIDRFromByte(podInByte, vlanCIDR, nodeBlock)
						// check if recomputed pod VLAN equal to the computed pod VLAN in  CIDR resource
						if podCIDR != host.PodCIDR {
							entry.Hosts[itemIndex].PodCIDR = podCIDR
							changed = true
						}
						if interfaceName != host.InterfaceName {
							entry.Hosts[itemIndex].InterfaceName = interfaceName
							changed = true
						}
						if hostIP != host.HostIP {
							entry.Hosts[itemIndex].HostIP = hostIP
							changed = true
						}
					} else {
						// tabu, recompute host index
						entry.Hosts = append(entry.Hosts[0:itemIndex], entry.Hosts[itemIndex+1:]...)
						entry, changed = h.tryAddNewHost(existingHosts, entry, maxHostIndex, def, hostName, interfaceName, hostIP)
					}
				}
			}
			entriesMap[interfaceNetAddress] = entry
		}
	}

	if len(diedHost) > 0 {
		h.Log.Info(fmt.Sprintf("Delete died hosts: %v", diedHost))
		// delete died host
		for interfaceNetAddress, entry := range entriesMap {
			aliveHosts := []multinicv1.HostInterfaceInfo{}
			for _, host := range entry.Hosts {
				if _, died := diedHost[host.HostName]; !died {
					aliveHosts = append(aliveHosts, host)
				}
			}
			entry.Hosts = aliveHosts
			entriesMap[interfaceNetAddress] = entry
		}
	}

	return entriesMap, changed
}

// UpdateCIDR computes host indexes and coresponding pod VLAN from host interface list
func (h *CIDRHandler) UpdateCIDR(cidrSpec multinicv1.CIDRSpec, new bool) (bool, error) {
	if !ConfigReady {
		h.Log.Info("Config is not ready yet, skip CIDR update")
		return false, nil
	}
	h.Mutex.Lock()
	def := cidrSpec.Config
	excludes := compute.SortAddress(def.ExcludeCIDRs)
	h.Log.Info(fmt.Sprintf("Update CIDR %s", def.Name))
	entriesMap, changed := h.updateEntries(cidrSpec, excludes, new)
	// if pod CIDR changes, update CIDR and create corresponding IPPools and routes
	if changed {
		newEntries := []multinicv1.CIDREntry{}
		for _, entry := range entriesMap {
			newEntries = append(newEntries, entry)
		}

		spec := multinicv1.CIDRSpec{
			Config: def,
			CIDRs:  newEntries,
		}
		mapObj := &multinicv1.CIDR{
			ObjectMeta: metav1.ObjectMeta{
				Name: def.Name,
			},
			Spec: spec,
		}
		ippoolSnapshot := h.IPPoolHandler.ListCache()
		// create/update CIDR
		existCIDR, err := h.GetCIDR(def.Name)
		if err == nil {
			updateCIDR := existCIDR.DeepCopy()
			updateCIDR.Spec = spec
			h.CleanPendingIPPools(ippoolSnapshot, def.Name, updateCIDR.Spec)
			err = h.Client.Update(context.TODO(), updateCIDR)
		} else {
			err = h.Client.Create(context.TODO(), mapObj)
			h.CleanPendingIPPools(ippoolSnapshot, def.Name, mapObj.Spec)
		}

		if err != nil {
			h.Log.Info(fmt.Sprintf("Cannot create or update CIDR %s: error=%v", def.Name, err))
			h.Mutex.Unlock()
			return false, err
		}

		if h.IsL3Mode(def) {
			// initialize the MultiNicNetwork status
			h.MultiNicNetworkHandler.UpdateStatus(*mapObj, multinicv1.ApplyingRoute)
		}

		// update IPPools
		h.IPPoolHandler.UpdateIPPools(def.Name, newEntries, excludes)
		h.Log.Info(fmt.Sprintf("CIDR %s changed", def.Name))
	}
	h.Mutex.Unlock()
	return changed, nil
}

// SyncCIDRRoute try adding routes by CIDR
func (h *CIDRHandler) SyncCIDRRoute(cidrSpec multinicv1.CIDRSpec, forceDelete bool) (status multinicv1.RouteStatus) {
	def := cidrSpec.Config
	// try re-adding routes
	if h.IsL3Mode(def) {
		h.Mutex.Lock()
		entries := cidrSpec.CIDRs
		hostInterfaceInfoMap := h.GetHostInterfaceIndexMap(entries)
		h.Log.Info(fmt.Sprintf("Sync routes from CIDR (force delete: %v)", forceDelete))
		success, noConnection := h.RouteHandler.AddRoutes(cidrSpec, entries, hostInterfaceInfoMap, forceDelete)
		h.Mutex.Unlock()
		if noConnection {
			return multinicv1.RouteUnknown
		}
		if forceDelete && !success {
			return multinicv1.SomeRouteFailed
		}
		return multinicv1.AllRouteApplied
	} else {
		return multinicv1.RouteNoApplied
	}
}

func (h *CIDRHandler) SyncCIDRRouteToHost(daemon DaemonPod) {
	snapshot := h.ListCache()
	for name, cidrSpec := range snapshot {
		def := cidrSpec.Config
		if h.IsL3Mode(def) {
			h.Mutex.Lock()
			entries := cidrSpec.CIDRs
			hostInterfaceInfoMap := h.GetHostInterfaceIndexMap(entries)
			hostName := daemon.NodeName
			if _, ok := hostInterfaceInfoMap[hostName]; ok {
				change, connectFail := h.AddRoutesToHost(cidrSpec, hostName, daemon, entries, hostInterfaceInfoMap, false)
				h.Log.Info(fmt.Sprintf("Add route to host %s change:%v, connectionFail: %v)", hostName, change, connectFail))
				if connectFail {
					routeStatus := multinicv1.RouteUnknown
					err := h.MultiNicNetworkHandler.SyncStatus(name, cidrSpec, routeStatus)
					if err != nil {
						h.Log.Info(fmt.Sprintf("failed to update route status of %s: %v", name, err))
					}
				}
			}
			h.Mutex.Unlock()
		}
	}
}

// DeleteOldRoutes forcefully deletes old routes from CIDR
func (h *CIDRHandler) DeleteOldRoutes(cidrSpec multinicv1.CIDRSpec) {
	def := cidrSpec.Config
	if h.IsL3Mode(def) {
		h.RouteHandler.DeleteRoutes(cidrSpec)
	}
}

// CleanPendingIPPools clean ippools in case that cidr is updated with new subnet entry
func (h *CIDRHandler) CleanPendingIPPools(snapshot map[string]multinicv1.IPPoolSpec, defName string, newCIDR multinicv1.CIDRSpec) map[string]multinicv1.IPPoolSpec {
	newPoolMap := make(map[string]multinicv1.CIDREntry)
	for _, entry := range newCIDR.CIDRs {
		for _, host := range entry.Hosts {
			newPoolMap[host.PodCIDR] = entry
		}
	}
	// delete IPPool that not in valid list
	for ippoolName, ippool := range snapshot {
		if ippool.NetAttachDefName == defName {
			if _, exist := newPoolMap[ippool.PodCIDR]; !exist {
				h.Log.Info(fmt.Sprintf("Delete IPPool: %s", ippoolName))
				h.DeleteIPPool(ippool.NetAttachDefName, ippool.PodCIDR)
			}
		}
	}
	return snapshot
}

// SyncIPPoolWithActivePods adds assigned IP to the IPPool and reported unsync IPs
func (h *CIDRHandler) SyncIPPoolWithActivePods(cidrMap map[string]multinicv1.CIDR, ippoolSnapshot map[string]multinicv1.IPPoolSpec) {
	h.Log.Info("SyncIPPoolWithActivePods")
	syncedMap := make(map[string]int)
	validIPPoolCache := make(map[string]multinicv1.IPPoolSpec)
	// get syncedMap mapping Pod ID with the number of IPs found in valid IPPool based on CIDR
	for ippoolName, ippool := range ippoolSnapshot {
		defName := ippool.NetAttachDefName
		if cidr, found := cidrMap[defName]; !found {
			h.Log.Info(fmt.Sprintf("Delete IPPool: %s (no corresponding CIDR)", ippoolName))
			h.DeleteIPPool(ippool.NetAttachDefName, ippool.PodCIDR)
		} else {
			for _, entry := range cidr.Spec.CIDRs {
				for _, host := range entry.Hosts {
					ippoolName := h.IPPoolHandler.GetIPPoolName(defName, host.PodCIDR)
					if ippool, exist := ippoolSnapshot[ippoolName]; exist {
						for _, allocation := range ippool.Allocations {
							syncID := getPodIPsSyncedMapID(defName, allocation.Pod, allocation.Namespace)
							syncedMap[syncID] += 1
						}
						validIPPoolCache[ippoolName] = ippool
					}
				}
			}
		}
	}

	unsyncedIPMap, err := h.getUnsyncPodIPs(cidrMap, syncedMap)
	if err != nil {
		h.Log.Info(fmt.Sprintf("Cannot getUnsyncPodIPs: %v", err))
		return
	}
	// check all valid ippool cache
	for ippoolName, ippool := range validIPPoolCache {
		defName := ippool.NetAttachDefName
		if allocationMap, defFound := unsyncedIPMap[defName]; !defFound || len(allocationMap) == 0 {
			// no unsynced
			continue
		} else {
			newAllocations := make([]multinicv1.Allocation, 0)
			for unsyncedIp, unsyncedAllocation := range allocationMap {
				contains, index := h.CIDRCompute.GetIndexInRange(ippool.PodCIDR, unsyncedIp)
				if contains {
					unsyncedAllocation.Index = index
					newAllocations = append(newAllocations, unsyncedAllocation)
					delete(unsyncedIPMap[defName], unsyncedIp)
				}
			}
			if len(newAllocations) > 0 {
				err = h.IPPoolHandler.AppendIPPoolAllocations(ippoolName, newAllocations)
				if err != nil {
					h.Log.Info(fmt.Sprintf("Cannot AppendIPPoolAllocations %v to %s: %v", newAllocations, ippoolName, err))
				} else {
					h.Log.Info(fmt.Sprintf("Patch IPPool %s, add allocations: %v", ippoolName, newAllocations))
				}
			}
		}
	}
	for defName, ipMap := range unsyncedIPMap {
		if len(ipMap) > 0 {
			h.Log.Info(fmt.Sprintf("List of IPs still unsync for %s: %v", defName, ipMap))
		}
	}
}

// getUnsyncPodIPs returns mapping of deName->allocations of that is unsync from syncedMap defName/podName/podNamespace
func (h *CIDRHandler) getUnsyncPodIPs(cidrMap map[string]multinicv1.CIDR, syncedMap map[string]int) (map[string]map[string]multinicv1.Allocation, error) {
	selectors := fmt.Sprintf("%s=%s", POD_STATUS_FIELD, POD_STATUS_VALUE)
	listOptions := metav1.ListOptions{
		FieldSelector: selectors,
	}
	pods, err := h.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), listOptions)
	unsyncedIPMap := make(map[string]map[string]multinicv1.Allocation)
	if err == nil {
		for _, pod := range pods.Items {
			// for unsynced pods
			podName := pod.GetName()
			podNamespace := pod.GetNamespace()
			networksStatus := make([]plugin.NetworkStatus, 0)
			if networkStatusStr, valid := pod.Annotations[plugin.StatusesKey]; valid {
				err := json.Unmarshal([]byte(networkStatusStr), &networksStatus)
				if err != nil {
					h.Log.Info(fmt.Sprintf("Cannot unmarshal NetworkStatus: %s", networkStatusStr))
					continue
				}
				for _, status := range networksStatus {
					nameSplit := strings.Split(status.Name, "/")
					var defName string
					if len(nameSplit) > 1 {
						defName = nameSplit[len(nameSplit)-1]
					} else {
						defName = nameSplit[0]
					}
					if _, cidrFound := cidrMap[defName]; !cidrFound {
						// irrelevant status
						continue
					}
					syncID := getPodIPsSyncedMapID(defName, podName, podNamespace)
					if count, synced := syncedMap[syncID]; synced && count == len(status.IPs) {
						// already synced
						continue
					}
					h.Log.Info(fmt.Sprintf("%s unsynced", syncID))
					_, found := unsyncedIPMap[defName]
					if !found {
						unsyncedIPMap[defName] = make(map[string]multinicv1.Allocation)
					}
					for _, ip := range status.IPs {
						allocation := multinicv1.Allocation{
							Pod:       pod.GetName(),
							Namespace: pod.GetNamespace(),
							Address:   ip,
						}
						unsyncedIPMap[defName][ip] = allocation
					}
				}
			}
		}
	}
	return unsyncedIPMap, err
}

// addNewHost finds new available host index
func (h *CIDRHandler) addNewHost(hosts []multinicv1.HostInterfaceInfo, maxHostIndex int, vlanCIDR string, nodeBlock int, excludes []string) (string, int, error) {
	nodeIndex := 0
	// excludedIndexes = previously-assigned host indexes
	excludedIndexes := []int{}
	for _, host := range hosts {
		excludedIndexes = append(excludedIndexes, host.HostIndex)
	}
	// find new available host index
	for {
		if len(excludedIndexes) > 0 {
			sort.Ints(excludedIndexes)
			// set nodeIndex to the next number from the last assigned index
			nodeIndex = excludedIndexes[len(excludedIndexes)-1] + 1
			if nodeIndex > maxHostIndex {
				// next number is too large, find unassigned index
				nodeIndex = h.CIDRCompute.FindAvailableIndex(excludedIndexes, 0, 0)
				if nodeIndex == -1 {
					// no index available, return error
					return "", -1, errors.New("wrong request (no available host index)")
				}
			}
		}
		vlanInByte, err := h.CIDRCompute.ComputeNet(vlanCIDR, nodeIndex, nodeBlock)
		if err == nil {
			// valid VLAN, check tabu ranges in definition
			tabu := h.CIDRCompute.CheckIfTabuIndex(vlanCIDR, nodeIndex, nodeBlock, excludes)
			if !tabu {
				// not tabu, return valid pod CIDR
				podCIDR := h.CIDRCompute.GetCIDRFromByte(vlanInByte, vlanCIDR, nodeBlock)
				return podCIDR, nodeIndex, nil
			}
		} else {
			// invalid VLAN
			h.Log.Info(fmt.Sprintf("Cannot assign nodeIndex %d, %v", nodeIndex, err))
		}
		// VLAN in tabu ranges or invalid, try next
		excludedIndexes = append(excludedIndexes, nodeIndex)
	}
}

// getInterfaceEntry get entry from interfaceaddress if exists, otherwise create new
func (h *CIDRHandler) getInterfaceEntry(def multinicv1.PluginConfig, entriesMap map[string]multinicv1.CIDREntry, newNetAdress string) (bool, multinicv1.CIDREntry) {
	if entry, found := entriesMap[newNetAdress]; found {
		// already exists
		return true, entry
	}
	var excludedIndexes []int
	for _, entry := range entriesMap {
		index := entry.InterfaceIndex
		excludedIndexes = append(excludedIndexes, index)
	}
	masterIndex := int(-1)
	maxInterfaceIndex := int(math.Pow(2, float64(def.InterfaceBlock)) - 1)
	vlanCIDR := ""
	for vlanCIDR == "" {
		masterIndex = h.CIDRCompute.FindAvailableIndex(excludedIndexes, -1, 0)
		fmt.Printf("excludedIndexes: %v\n", excludedIndexes)
		if masterIndex < 0 {
			masterIndex = len(excludedIndexes)
		}
		tabu := h.CIDRCompute.CheckIfTabuIndex(def.Subnet, masterIndex, def.InterfaceBlock, def.ExcludeCIDRs)
		if tabu {
			excludedIndexes = append(excludedIndexes, masterIndex)
		} else {
			vlanInByte, err := h.CIDRCompute.ComputeNet(def.Subnet, masterIndex, def.InterfaceBlock)
			if err != nil {
				excludedIndexes = append(excludedIndexes, masterIndex)
				continue
			}
			vlanCIDR = h.CIDRCompute.GetCIDRFromByte(vlanInByte, def.Subnet, def.InterfaceBlock)
			break
		}
		if masterIndex > maxInterfaceIndex {
			h.Log.Info("cannot add new interface (no available index)")
			return false, multinicv1.CIDREntry{}
		}
	}
	return true, multinicv1.CIDREntry{
		NetAddress:     newNetAdress,
		InterfaceIndex: masterIndex,
		VlanCIDR:       vlanCIDR,
		Hosts:          []multinicv1.HostInterfaceInfo{},
	}
}

// tryAddNewHost creates new entry of HostInterfaceInfo in CIDR and computes corresponding pod VLAN
func (h *CIDRHandler) tryAddNewHost(existingHosts []multinicv1.HostInterfaceInfo, entry multinicv1.CIDREntry, maxHostIndex int, def multinicv1.PluginConfig, hostName, interfaceName, hostIP string) (multinicv1.CIDREntry, bool) {
	h.Log.Info(fmt.Sprintf("TryAddNewHost %s:, LastIndex:%d, InterfaceName: %s, HostIP: %s", hostName, maxHostIndex, interfaceName, hostIP))
	podCIDR, hostIndex, err := h.addNewHost(existingHosts, maxHostIndex, entry.VlanCIDR, def.HostBlock, def.ExcludeCIDRs)
	if err == nil {
		// successfully compute pod VLAN, create and append new entry of HostInterfaceInfo orderly
		newHost := multinicv1.HostInterfaceInfo{
			HostIndex:     hostIndex,
			HostName:      hostName,
			InterfaceName: interfaceName,
			HostIP:        hostIP,
			PodCIDR:       podCIDR,
		}
		hosts := append(existingHosts, newHost)
		sort.SliceStable(hosts, func(i, j int) bool {
			return hosts[i].HostIndex < hosts[j].HostIndex
		})
		entry.Hosts = hosts
		return entry, true
	} else {
		h.Log.Info(fmt.Sprintf("Cannot add new host %s, %s: %v", hostName, interfaceName, err))
		return entry, false
	}
}

// IsL3Mode checkes L3 VLAN mode (to add/delete L3 routes automatically)
func (h *CIDRHandler) IsL3Mode(def multinicv1.PluginConfig) bool {
	net, err := h.MultiNicNetworkHandler.GetNetwork(def.Name)
	if err != nil {
		// not corresponding definition
		return false
	}
	if !net.Spec.IsMultiNICIPAM {
		// not managed by multi-nic IPAM
		return false
	}
	mode := def.VlanMode
	switch mode {
	case "", "l2":
		return false
	case "l3":
		return true
	case "l3s":
		return true
	default:
		return false
	}
}

// getHostIndex finds assigned host index from the HostInterfaceInfo list
func (h *CIDRHandler) getHostIndex(hosts []multinicv1.HostInterfaceInfo, hostName string) int {
	for index, host := range hosts {
		if hostName == host.HostName {
			return index
		}
	}
	return -1
}

// GetHostInterfaceIndexMap finds a map from (host name, interface index) to HostInterfaceInfo of CIDR
func (h *CIDRHandler) GetHostInterfaceIndexMap(entries []multinicv1.CIDREntry) map[string]map[int]multinicv1.HostInterfaceInfo {
	hostInterfaceIndexMap := make(map[string]map[int]multinicv1.HostInterfaceInfo)
	for _, entry := range entries {
		ifaceIndex := entry.InterfaceIndex
		for _, host := range entry.Hosts {
			hostName := host.HostName
			if _, exists := hostInterfaceIndexMap[hostName]; !exists {
				hostInterfaceIndexMap[hostName] = make(map[int]multinicv1.HostInterfaceInfo)
			}
			hostInterfaceIndexMap[hostName][ifaceIndex] = host
		}
	}
	return hostInterfaceIndexMap
}

// handling CIDRCache
func (h *CIDRHandler) SetCache(key string, value multinicv1.CIDRSpec) {
	h.SafeCache.SetCache(key, value)
}

func (h *CIDRHandler) GetCache(key string) (multinicv1.CIDRSpec, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return multinicv1.CIDRSpec{}, fmt.Errorf("Not Found")
	}
	return value.(multinicv1.CIDRSpec), nil
}

func (h *CIDRHandler) ListCache() map[string]multinicv1.CIDRSpec {
	snapshot := make(map[string]multinicv1.CIDRSpec)
	h.SafeCache.Lock()
	for key, value := range h.cache {
		snapshot[key] = value.(multinicv1.CIDRSpec)
	}
	h.SafeCache.Unlock()
	return snapshot
}

// getPodIPsSyncedMapID returns combination of defName, podName, podNamespace
func getPodIPsSyncedMapID(defName string, podName string, podNamespace string) string {
	return fmt.Sprintf("%s/%s/%s", defName, podName, podNamespace)
}
