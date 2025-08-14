/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"fmt"
	"time"

	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

func RunPeriodicUpdate(ticker *time.Ticker, daemonWatcher *DaemonWatcher, cidrHandler *CIDRHandler, hostInterfaceReconciler *HostInterfaceReconciler, quit chan struct{}) {
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				// update interface
				if ConfigReady && daemonWatcher.IsDaemonSetReady() {
					vars.SyncLog.V(7).Info(fmt.Sprintf("synchronizing state... %d HostInterfaces, %d CIDRs", cidrHandler.HostInterfaceHandler.SafeCache.GetSize(), cidrHandler.SafeCache.GetSize()))
					daemonSize := daemonWatcher.DaemonCacheHandler.GetSize()
					hostInterfaceSnapshot := cidrHandler.HostInterfaceHandler.ListCache()
					infoAvailableSize := 0
					for hostName, instance := range hostInterfaceSnapshot {
						err := hostInterfaceReconciler.UpdateInterfaces(instance)
						if err != nil {
							vars.SyncLog.V(4).Info(fmt.Sprintf("Failed to update HostInterface %s: %v", hostName, err))
						} else if len(instance.Spec.Interfaces) > 0 {
							infoAvailableSize += 1
						}
					}
					cidrSnapshot := cidrHandler.ListCache()
					ippoolSnapshot := cidrHandler.IPPoolHandler.ListCache()
					for name, instanceSpec := range cidrSnapshot {
						routeStatus := cidrHandler.SyncCIDRRoute(instanceSpec, false)
						cidrHandler.CleanPendingIPPools(ippoolSnapshot, name, instanceSpec)
						netStatus, err := cidrHandler.MultiNicNetworkHandler.SyncAllStatus(name, instanceSpec, routeStatus, daemonSize, infoAvailableSize, false)
						if err != nil {
							vars.SyncLog.V(3).Info(fmt.Sprintf("Failed to update route status of %s: %v", name, err))
						} else if netStatus.CIDRProcessedHost != netStatus.InterfaceInfoAvailable {
							cidrHandler.UpdateCIDRs()
						}
					}
				}
			}
		}
	}()
}
