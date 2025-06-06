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

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"sort"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	"github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"errors"
	"sync"
)

const ()

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
	RouteHandler
	*SafeCache
	UpdateRequestQueue chan struct{}
	Quit               chan struct{}
	requestQueueClosed bool
}

func NewCIDRHandler(client client.Client, config *rest.Config, hostInterfaceHandler *HostInterfaceHandler, daemonCache *DaemonCacheHandler, quit chan struct{}) *CIDRHandler {
	clientset, _ := kubernetes.NewForConfig(config)
	cidrCompute := compute.CIDRCompute{}
	updateReq := make(chan struct{}, vars.MaxQueueSize)
	handler := &CIDRHandler{
		Client:               client,
		Clientset:            clientset,
		CIDRCompute:          cidrCompute,
		HostInterfaceHandler: hostInterfaceHandler,
		IPPoolHandler: &IPPoolHandler{
			Client:    client,
			SafeCache: InitSafeCache(),
		},
		MultiNicNetworkHandler: &MultiNicNetworkHandler{
			Client:    client,
			SafeCache: InitSafeCache(),
		},
		RouteHandler: RouteHandler{
			DaemonConnector: DaemonConnector{
				Clientset: clientset,
			},
			DaemonCacheHandler: daemonCache,
		},
		SafeCache:          InitSafeCache(),
		UpdateRequestQueue: updateReq,
		Quit:               quit,
	}
	return handler
}

// InitCustomCRCache inits existing list of IPPool and HostInterface to avoid unexpected recomputation of CIDR
func (h *CIDRHandler) InitCustomCRCache() {
	err := InitIppoolCache(h.IPPoolHandler)
	if err != nil {
		vars.IPPoolLog.V(5).Info(fmt.Sprintf("Failed to InitIppoolCache: %v", err))
	}
	err = InitHostInterfaceCache(h.Clientset, h.HostInterfaceHandler, h.RouteHandler.DaemonCacheHandler)
	if err != nil {
		vars.HifLog.V(4).Info(fmt.Sprintf("Failed to InitHostInterfaceCache: %v", err))
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
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.Get(ctx, namespacedName, instance)
	return instance, err
}

// ListCIDR returns a map from CIDR name to instance
func (h *CIDRHandler) ListCIDR() (map[string]multinicv1.CIDR, error) {
	cidrList := &multinicv1.CIDRList{}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.List(ctx, cidrList)
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
	if err == nil {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Checking %d cidrs", len(cidrMap)))
		validCIDRs := h.syncWithMultinicNetwork(defHandler, cidrMap)
		_ = h.syncIPPools(validCIDRs)
	} else {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to list cidr: %v", err))
	}
}

// syncWithMultinicNetwork patches status of MultiNicNetwork with CIDR spec
// and delete CIDR if no corresponding MultiNicNetwork
func (h *CIDRHandler) syncWithMultinicNetwork(defHandler *plugin.NetAttachDefHandler, cidrMap map[string]multinicv1.CIDR) map[string]multinicv1.CIDR {
	daemonSize := h.DaemonCacheHandler.SafeCache.GetSize()
	infoAvailableSize := h.HostInterfaceHandler.GetInfoAvailableSize()
	validCIDRs := make(map[string]multinicv1.CIDR, 0)
	for name, cidr := range cidrMap {
		multinicnetwork, err := h.MultiNicNetworkHandler.GetNetwork(name)
		if err != nil && k8serrors.IsNotFound(err) {
			vars.CIDRLog.V(3).Info(fmt.Sprintf("%v, delete pending resources (CIDR)", err))
			h.deletePendingCIDR(defHandler, name, cidr)
		} else if err == nil {
			_, err = h.MultiNicNetworkHandler.SyncAllStatus(name, cidr.Spec, multinicnetwork.Status.RouteStatus, daemonSize, infoAvailableSize, true)
			if err != nil {
				vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to SyncAllStatus: %v", err))
			}
			validCIDRs[name] = cidr
		} else {
			vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to get MultiNicNetwork: %v", err))
		}
	}
	return validCIDRs
}

// syncIPPools creates/updates/deletes IPPool corresponding to MultiNicNetwork and active Pods
// defHandler is used for deleting NAD which has no corresponding MultiNicNetwork
func (h *CIDRHandler) syncIPPools(validCIDRs map[string]multinicv1.CIDR) map[string][]multinicv1.Allocation {
	ippoolSnapshot := h.IPPoolHandler.ListCache()
	for name, cidr := range validCIDRs {
		h.CleanPendingIPPools(ippoolSnapshot, name, cidr.Spec)
		h.createOrUpdateIPPool(ippoolSnapshot, name, cidr)
	}
	newAllocationMap := h.SyncIPPoolWithActivePods(validCIDRs, ippoolSnapshot)
	return newAllocationMap
}

// deletePendingCIDR deletes NAD that has corresponding multinicnetwork
func (h *CIDRHandler) deletePendingCIDR(defHandler *plugin.NetAttachDefHandler, name string, cidr multinicv1.CIDR) {
	err := defHandler.Delete(name, metav1.NamespaceAll)
	if err != nil {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to delete NetworkAttachmentDefinition %s: %v", name, err))
	}
	err = h.DeleteCIDR(cidr)
	if err != nil {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to delete CIDR %s: %v", name, err))
	}
}

// createOrUpdateIPPool creates or updates IPPool in the snapshot corresponding to the given CIDR
func (h *CIDRHandler) createOrUpdateIPPool(ippoolSnapshot map[string]multinicv1.IPPoolSpec, name string, cidr multinicv1.CIDR) {
	h.SetCache(name, cidr.Spec)
	excludes := compute.SortAddress(cidr.Spec.Config.ExcludeCIDRs)
	for _, entry := range cidr.Spec.CIDRs {
		for _, host := range entry.Hosts {
			ippoolName := host.IPPool
			if ippoolName == "" {
				ippoolName = h.IPPoolHandler.GetIPPoolName(name, host.PodCIDR)
			}
			if _, found := ippoolSnapshot[ippoolName]; !found {
				err := h.UpdateIPPool(name, host.PodCIDR, entry.VlanCIDR, host.HostName, host.InterfaceName, excludes)
				if err != nil {
					vars.CIDRLog.V(5).Info(fmt.Sprintf("Failed to update IPPool %s: %v", ippoolName, err))
				}
			}
		}
	}
}

// DeleteCIDR deletes corresponding routes and IPPools, then deletes CIDR
func (h *CIDRHandler) DeleteCIDR(cidr multinicv1.CIDR) error {
	name := cidr.GetName()
	vars.CIDRLog.V(3).Info(fmt.Sprintf("Delete CIDR: %s", name))
	errorMsg := ""
	// delete corresponding routes
	h.deleteRoutesFromCIDR(cidr.Spec)
	// delete corresponding IPPools
	for _, entry := range cidr.Spec.CIDRs {
		for _, host := range entry.Hosts {
			podCIDR := host.PodCIDR
			err := h.IPPoolHandler.DeleteIPPool(name, podCIDR)
			if err != nil {
				errorMsg = errorMsg + fmt.Sprintf("%v,", err)
			}
		}
	}
	instance, err := h.GetCIDR(name)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		err = h.Client.Delete(ctx, instance)
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

// NewCIDR returns new CIDR from PluginConfig
func (h *CIDRHandler) NewCIDR(def multinicv1.PluginConfig, namespace string) (multinicv1.CIDRSpec, error) {
	if def.Subnet == "" {
		return h.GenerateCIDRFromHostSubnet(def)
	}
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

// run through update queue
func (h *CIDRHandler) Run() {
	defer close(h.UpdateRequestQueue)
	vars.CIDRLog.V(7).Info("start processing CIDR update request queue")
	wait.Until(h.ProcessUpdateRequest, 0, h.Quit)
	vars.CIDRLog.V(3).Info("stop processing CIDR update request queue")
	h.requestQueueClosed = true
}

// UpdateCIDR adds ticket to activate ProcessUpdateRequest
func (h *CIDRHandler) UpdateCIDRs() {
	vars.CIDRLog.V(7).Info(fmt.Sprintf("Add UpdateRequest (%d waiting)", len(h.UpdateRequestQueue)))
	if !h.requestQueueClosed {
		h.UpdateRequestQueue <- struct{}{}
	}
}

// ProcessUpdateRequest modifies existing CIDR from the new HostInterface information
func (h *CIDRHandler) ProcessUpdateRequest() {
	vars.CIDRLog.V(7).Info("ProcessUpdateRequest listening...")
	<-h.UpdateRequestQueue
	for len(h.UpdateRequestQueue) > 0 {
		<-h.UpdateRequestQueue
	}
	vars.CIDRLog.V(7).Info(fmt.Sprintf("Update CIDRs (%d in the queue)", len(h.UpdateRequestQueue)))
	cidrSnapshot := h.ListCache()
	for _, cidr := range cidrSnapshot {
		_, err := h.updateCIDR(cidr, false)
		if err != nil {
			vars.CIDRLog.V(3).Info(fmt.Sprintf("Fail to update CIDR: %v", err))
		}
	}
}

// NewCIDRWithNewConfig creates new CIDR by computing interface indexes from master networks
func (h *CIDRHandler) NewCIDRWithNewConfig(def multinicv1.PluginConfig, namespace string) (bool, error) {
	vars.CIDRLog.V(3).Info("NewCIDRWithNewConfig")
	cidrSpec, err := h.NewCIDR(def, namespace)
	if err != nil {
		return false, err
	}
	return h.updateCIDR(cidrSpec, true)
}

// UpdateEntries updates CIDR entry from current HostInterfaceCache
func (h *CIDRHandler) UpdateEntries(cidrSpec multinicv1.CIDRSpec, excludes []compute.IPValue, changed bool) (map[string]multinicv1.CIDREntry, bool) {
	vars.CIDRLog.V(7).Info("UpdateEntries")
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
					if node, foundErr := h.Clientset.CoreV1().Nodes().Get(context.TODO(), host.HostName, metav1.GetOptions{}); (foundErr != nil && k8serrors.IsNotFound(foundErr) && k8serrors.IsNotFound(foundErr)) || (foundErr == nil && node.Status.Phase == v1.NodeTerminated) {
						// host died or terminating
						vars.CIDRLog.V(3).Info(fmt.Sprintf("Host %s no longer exist, delete from entry of CIDR %s", host.HostName, cidrSpec.Config.Name))
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
			// new or has some in newHostList
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

	// initCheckInterfaceMap
	checkInterfaceMap := make(map[string]bool)
	for _, netAddr := range def.MasterNetAddrs {
		checkInterfaceMap[netAddr] = true
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
			if len(def.MasterNetAddrs) > 0 {
				if _, found := checkInterfaceMap[interfaceNetAddress]; !found {
					continue
				}
			}
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
						if host.IPPool == "" {
							// ippool not set (snapshot from previous version)
							ippoolName := h.IPPoolHandler.GetIPPoolName(def.Name, podCIDR)
							entry.Hosts[itemIndex].IPPool = ippoolName
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
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Delete died hosts: %v", diedHost))
		// delete died host
		for interfaceNetAddress, entry := range entriesMap {
			aliveHosts := []multinicv1.HostInterfaceInfo{}
			for _, host := range entry.Hosts {
				if _, died := diedHost[host.HostName]; !died {
					aliveHosts = append(aliveHosts, host)
				}
			}
			entry.Hosts = aliveHosts
			if len(aliveHosts) == 0 {
				continue
			}
			entriesMap[interfaceNetAddress] = entry
		}
	}

	vars.CIDRLog.V(7).Info("UpdateEntries done")

	return entriesMap, changed
}

// updateCIDR computes host indexes and coresponding pod VLAN from host interface list
func (h *CIDRHandler) updateCIDR(cidrSpec multinicv1.CIDRSpec, new bool) (bool, error) {
	if !ConfigReady {
		vars.CIDRLog.V(7).Info("Config is not ready yet, skip CIDR update")
		return false, nil
	}
	h.Mutex.Lock()
	def := cidrSpec.Config
	excludes := compute.SortAddress(def.ExcludeCIDRs)
	vars.CIDRLog.V(7).Info(fmt.Sprintf("Update CIDR %s", def.Name))
	entriesMap, changed := h.UpdateEntries(cidrSpec, excludes, new)
	// if pod CIDR changes, update CIDR and create corresponding IPPools and routes
	if changed {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("changeCIDR %s", def.Name))
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
			updatedCIDR := existCIDR.DeepCopy()
			updatedCIDR.Spec = spec
			ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
			defer cancel()
			err = h.Client.Update(ctx, updatedCIDR)
			if err == nil {
				h.SafeCache.SetCache(def.Name, updatedCIDR.Spec)
			}
			h.CleanPendingIPPools(ippoolSnapshot, def.Name, updatedCIDR.Spec)
		} else {
			if new {
				ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
				defer cancel()
				err = h.Client.Create(ctx, mapObj)
				if err == nil {
					h.SafeCache.SetCache(def.Name, mapObj.Spec)
				}
				h.CleanPendingIPPools(ippoolSnapshot, def.Name, mapObj.Spec)
			} else {
				err = fmt.Errorf("TryInitExistingConfig")
			}
		}
		if err != nil {
			vars.CIDRLog.V(3).Info(fmt.Sprintf("Cannot create or update CIDR %s: error=%v", def.Name, err))
			h.Mutex.Unlock()
			return false, err
		}

		if h.IsL3Mode(def) {
			// initialize the MultiNicNetwork status
			daemonSize := h.DaemonCacheHandler.GetSize()
			infoAvailableSize := h.GetInfoAvailableSize()
			netStatus, err := h.MultiNicNetworkHandler.SyncAllStatus(def.Name, spec, multinicv1.ApplyingRoute, daemonSize, infoAvailableSize, true)
			if err != nil {
				vars.CIDRLog.V(2).Info(fmt.Sprintf("Failed to update route status of %s: %v", def.Name, err))
			} else if netStatus.CIDRProcessedHost != netStatus.InterfaceInfoAvailable {
				h.UpdateCIDRs()
			}
		}

		// update IPPools
		h.IPPoolHandler.UpdateIPPools(def.Name, newEntries, excludes)
		vars.CIDRLog.V(7).Info(fmt.Sprintf("changeCIDR %s Done", def.Name))
	}
	h.Mutex.Unlock()
	return changed, nil
}

// SyncCIDRRoute try adding routes by CIDR
func (h *CIDRHandler) SyncCIDRRoute(cidrSpec multinicv1.CIDRSpec, forceDelete bool) (status multinicv1.RouteStatus) {
	def := cidrSpec.Config
	// try re-adding routes
	if h.IsL3Mode(def) {
		h.Mutex.Lock() // do not continue if cidr is under-processing
		entries := cidrSpec.CIDRs
		h.Mutex.Unlock()
		hostInterfaceInfoMap := h.GetHostInterfaceIndexMap(entries)
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Sync routes from CIDR (force delete: %v)", forceDelete))
		success, noConnection := h.RouteHandler.AddRoutes(cidrSpec, entries, hostInterfaceInfoMap, forceDelete)
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
				vars.CIDRLog.V(5).Info(fmt.Sprintf("Delete IPPool: %s", ippoolName))
				err := h.DeleteIPPool(ippool.NetAttachDefName, ippool.PodCIDR)
				if err != nil {
					vars.CIDRLog.V(5).Info(fmt.Sprintf("Failed to delete IPPool %s: %v", ippoolName, err))
				}
			}
		}
	}
	return snapshot
}

func (h *CIDRHandler) GetSyncAllocations(ippool multinicv1.IPPoolSpec, allocationMap map[string]map[string]multinicv1.Allocation, crAllocationMap map[string]map[string]multinicv1.Allocation) (bool, []multinicv1.Allocation) {
	changed := false
	newAllocations := make([]multinicv1.Allocation, 0)
	defName := ippool.NetAttachDefName
	if allocations, defFound := allocationMap[defName]; defFound && len(allocations) > 0 {
		for syncIP, allocation := range allocations {
			contains, index := h.CIDRCompute.GetIndexInRange(ippool.PodCIDR, allocation.Address)
			if !contains {
				continue
			}
			if crAllocations, found := crAllocationMap[defName]; found {
				if crAllocation, found := crAllocations[syncIP]; found && crAllocation.Pod == allocation.Pod && crAllocation.Namespace == allocation.Namespace {
					// existing item
					newAllocations = append(newAllocations, crAllocation)
					delete(allocationMap[defName], syncIP)
					delete(crAllocationMap[defName], syncIP)
					continue
				}
			}
			// new item
			allocation.Index = index
			newAllocations = append(newAllocations, allocation)
			delete(allocationMap[defName], syncIP)
			changed = true
		}
	}
	if len(ippool.Allocations) != len(newAllocations) {
		// some need to be cleaned
		changed = true
	}
	return changed, newAllocations
}

// SyncIPPoolWithActivePods adds assigned IP to the IPPool and reported unsync IPs
// returns mapping from IPPool name to new allocation list
func (h *CIDRHandler) SyncIPPoolWithActivePods(cidrMap map[string]multinicv1.CIDR, ippoolSnapshot map[string]multinicv1.IPPoolSpec) map[string][]multinicv1.Allocation {
	outputs := make(map[string][]multinicv1.Allocation, 0)
	vars.CIDRLog.V(5).Info("SyncIPPoolWithActivePods")
	crAllocationMap := make(map[string]map[string]multinicv1.Allocation)
	// delete pending ippool
	for ippoolName, ippool := range ippoolSnapshot {
		defName := ippool.NetAttachDefName
		if _, found := cidrMap[defName]; !found {
			vars.CIDRLog.V(5).Info(fmt.Sprintf("Delete IPPool: %s (no corresponding CIDR)", ippoolName))
			err := h.DeleteIPPool(ippool.NetAttachDefName, ippool.PodCIDR)
			if err != nil {
				vars.CIDRLog.V(5).Info(fmt.Sprintf("Failed to delete IPPool %s: %v", ippoolName, err))
			}
		}
	}
	// get syncedMap mapping Pod ID with the number of IPs found in valid IPPool based on CIDR
	for defName, cidr := range cidrMap {
		crAllocationMap[defName] = make(map[string]multinicv1.Allocation)
		for _, entry := range cidr.Spec.CIDRs {
			for _, host := range entry.Hosts {
				ippoolName := host.IPPool
				if ippoolName == "" {
					ippoolName = h.IPPoolHandler.GetIPPoolName(defName, host.PodCIDR)
				}
				if ippool, exist := ippoolSnapshot[ippoolName]; exist {
					for _, allocation := range ippool.Allocations {
						crAllocationMap[defName][allocation.Address] = allocation
					}
				}
			}
		}
	}

	allocationMap, err := h.getCurrentAllocationMap(cidrMap)
	if err != nil {
		vars.CIDRLog.V(5).Info(fmt.Sprintf("Cannot getCurrentAllocationMap: %v", err))
		return outputs
	}
	vars.CIDRLog.V(5).Info(fmt.Sprintf("allocationMap: %v", allocationMap))
	vars.CIDRLog.V(5).Info(fmt.Sprintf("crAllocationMap: %v", crAllocationMap))
	// check all valid ippool cache
	for ippoolName, ippool := range ippoolSnapshot {
		changed, newAllocations := h.GetSyncAllocations(ippool, allocationMap, crAllocationMap)
		if changed {
			err = h.IPPoolHandler.PatchIPPoolAllocations(ippoolName, newAllocations)
			if err != nil {
				vars.CIDRLog.V(5).Info(fmt.Sprintf("Cannot PatchIPPoolAllocations %v to %s: %v", newAllocations, ippoolName, err))
			} else {
				vars.CIDRLog.V(5).Info(fmt.Sprintf("Patch IPPool %s, new allocations: %v", ippoolName, newAllocations))
			}
		}
		outputs[ippoolName] = newAllocations
	}
	for defName, ipMap := range allocationMap {
		if len(ipMap) > 0 {
			vars.CIDRLog.V(5).Info(fmt.Sprintf("List of IPs still unsync for %s: %v", defName, ipMap))
		}
	}
	return outputs
}

// getCurrentAllocationMap returns mapping of deName->allocations
func (h *CIDRHandler) getCurrentAllocationMap(cidrMap map[string]multinicv1.CIDR) (map[string]map[string]multinicv1.Allocation, error) {
	selectors := fmt.Sprintf("%s=%s", vars.PodStatusField, vars.PodStatusRunning)
	listOptions := metav1.ListOptions{
		FieldSelector: selectors,
	}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	pods, err := h.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
	allocationMap := make(map[string]map[string]multinicv1.Allocation)
	if err == nil {
		for _, pod := range pods.Items {
			networksStatus := make([]plugin.NetworkStatus, 0)
			if networkStatusStr, valid := pod.Annotations[plugin.StatusesKey]; valid {
				err := json.Unmarshal([]byte(networkStatusStr), &networksStatus)
				if err != nil {
					vars.CIDRLog.V(5).Info(fmt.Sprintf("Cannot unmarshal NetworkStatus: %s", networkStatusStr))
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

					_, found := allocationMap[defName]
					if !found {
						allocationMap[defName] = make(map[string]multinicv1.Allocation)
					}
					for _, ip := range status.IPs {
						allocation := multinicv1.Allocation{
							Pod:       pod.GetName(),
							Namespace: pod.GetNamespace(),
							Address:   ip,
						}
						allocationMap[defName][ip] = allocation
					}
				}
			}
		}
	}
	return allocationMap, err
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
			vars.CIDRLog.V(3).Info(fmt.Sprintf("Cannot assign nodeIndex %d, %v", nodeIndex, err))
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
	if def.Subnet == "" {
		// set static entry index to latest
		index := len(entriesMap)
		entry := multinicv1.CIDREntry{
			NetAddress:     newNetAdress,
			InterfaceIndex: index,
			VlanCIDR:       newNetAdress,
			Hosts:          []multinicv1.HostInterfaceInfo{},
		}
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
		sort.Ints(excludedIndexes)
		masterIndex = h.CIDRCompute.FindAvailableIndex(excludedIndexes, 0, 0)
		if masterIndex < 0 {
			if len(excludedIndexes) > 0 {
				masterIndex = excludedIndexes[len(excludedIndexes)-1] + 1
			} else {
				masterIndex = 0
			}
		}
		if masterIndex > maxInterfaceIndex {
			vars.CIDRLog.V(3).Info("cannot add new interface (no available index)")
			return false, multinicv1.CIDREntry{}
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
	vars.CIDRLog.V(3).Info(fmt.Sprintf("TryAddNewHost %s:, LastIndex:%d, InterfaceName: %s, HostIP: %s", hostName, maxHostIndex, interfaceName, hostIP))
	podCIDR, hostIndex, err := h.addNewHost(existingHosts, maxHostIndex, entry.VlanCIDR, def.HostBlock, def.ExcludeCIDRs)
	if err == nil {
		ippoolName := h.IPPoolHandler.GetIPPoolName(def.Name, podCIDR)
		// successfully compute pod VLAN, create and append new entry of HostInterfaceInfo orderly
		newHost := multinicv1.HostInterfaceInfo{
			HostIndex:     hostIndex,
			HostName:      hostName,
			InterfaceName: interfaceName,
			HostIP:        hostIP,
			PodCIDR:       podCIDR,
			IPPool:        ippoolName,
		}
		hosts := append(existingHosts, newHost)
		sort.SliceStable(hosts, func(i, j int) bool {
			return hosts[i].HostIndex < hosts[j].HostIndex
		})
		entry.Hosts = hosts
		return entry, true
	} else {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Cannot add new host %s, %s: %v", hostName, interfaceName, err))
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

// GenerateCIDRFromHostSubnet generates CIDR from Host Subnet
// in the case that podCIDR must be shared with host subnet (e.g., in aws VPC)
func (h *CIDRHandler) GenerateCIDRFromHostSubnet(def multinicv1.PluginConfig) (multinicv1.CIDRSpec, error) {
	vlanIndexMap := make(map[string]int)
	entryMap := make(map[string]multinicv1.CIDREntry)
	lastIndex := 0

	// maxHostIndex = 2^(host bits) - 1
	maxHostIndex := int(math.Pow(2, float64(def.HostBlock)) - 1)

	snapshot := h.HostInterfaceHandler.ListCache()
	for hostName, hif := range snapshot {
		for _, iface := range hif.Spec.Interfaces {
			netAddr := iface.NetAddress
			if _, exist := vlanIndexMap[netAddr]; !exist {
				// new address
				vlanIndexMap[netAddr] = lastIndex
				entry := multinicv1.CIDREntry{
					NetAddress:     netAddr,
					InterfaceIndex: vlanIndexMap[netAddr],
					VlanCIDR:       netAddr,
					Hosts:          []multinicv1.HostInterfaceInfo{},
				}
				entryMap[netAddr] = entry
				lastIndex += 1
			}
			entry := entryMap[netAddr]
			entry, changed := h.tryAddNewHost(entry.Hosts, entry, maxHostIndex, def, hostName, iface.InterfaceName, iface.HostIP)
			if !changed {
				return multinicv1.CIDRSpec{}, fmt.Errorf("failed to add host to CIDR entry")
			}
			entryMap[netAddr] = entry
		}
	}
	entries := []multinicv1.CIDREntry{}
	for _, entry := range entryMap {
		entries = append(entries, entry)
	}
	cidrSpec := multinicv1.CIDRSpec{
		Config: def,
		CIDRs:  entries,
	}
	return cidrSpec, nil
}

// GetHostAddressesToExclude returns host addresses to be excluded
// in the case that dedicated podCIDR must be shared with host subnet (e.g., in aws VPC)
func (h *CIDRHandler) GetHostAddressesToExclude() []string {
	excludes := []string{}
	snapshot := h.HostInterfaceHandler.ListCache()
	for _, hif := range snapshot {
		for _, iface := range hif.Spec.Interfaces {
			hostIPCIDR := fmt.Sprintf("%s/32", iface.HostIP)
			excludes = append(excludes, hostIPCIDR)
		}
	}
	return excludes
}

// handling CIDRCache
func (h *CIDRHandler) SetCache(key string, value multinicv1.CIDRSpec) {
	h.SafeCache.SetCache(key, value)
}

func (h *CIDRHandler) GetCache(key string) (multinicv1.CIDRSpec, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return multinicv1.CIDRSpec{}, fmt.Errorf(vars.NotFoundError)
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
