package controllers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

func RunPeriodicUpdate(ticker *time.Ticker, daemonWatcher *DaemonWatcher, cidrHandler *CIDRHandler, hostInterfaceReconciler *HostInterfaceReconciler, logger logr.Logger, quit chan struct{}) {
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				// update interface
				if ConfigReady && daemonWatcher.IsDaemonSetReady() {
					logger.V(7).Info(fmt.Sprintf("synchronizing state... %d HostInterfaces, %d CIDRs", cidrHandler.HostInterfaceHandler.SafeCache.GetSize(), cidrHandler.SafeCache.GetSize()))
					daemonSize := daemonWatcher.DaemonCacheHandler.GetSize()
					hostInterfaceSnapshot := cidrHandler.HostInterfaceHandler.ListCache()
					infoAvailableSize := 0
					for _, instance := range hostInterfaceSnapshot {
						hostInterfaceReconciler.UpdateInterfaces(instance)
						if len(instance.Spec.Interfaces) > 0 {
							infoAvailableSize += 1
						}
					}
					cidrSnapshot := cidrHandler.ListCache()
					ippoolSnapshot := cidrHandler.IPPoolHandler.ListCache()
					for name, instanceSpec := range cidrSnapshot {
						routeStatus := cidrHandler.SyncCIDRRoute(instanceSpec, false)
						cidrHandler.CleanPendingIPPools(ippoolSnapshot, name, instanceSpec)
						err := cidrHandler.MultiNicNetworkHandler.SyncAllStatus(name, instanceSpec, routeStatus, daemonSize, infoAvailableSize, false)
						if err != nil {
							logger.V(2).Info(fmt.Sprintf("failed to update route status of %s: %v", name, err))
						}
					}
				}
			}
		}
	}()
}
