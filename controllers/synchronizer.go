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
					logger.Info(fmt.Sprintf("synchronizing state... %d HostInterfaces, %d CIDRs", cidrHandler.HostInterfaceHandler.SafeCache.GetSize(), cidrHandler.SafeCache.GetSize()))
					hostInterfaceSnapshot := cidrHandler.HostInterfaceHandler.ListCache()
					for _, instance := range hostInterfaceSnapshot {
						hostInterfaceReconciler.UpdateInterfaces(instance)
					}
					cidrSnapshot := cidrHandler.ListCache()
					ippoolSnapshot := cidrHandler.IPPoolHandler.ListCache()
					for name, instanceSpec := range cidrSnapshot {
						routeStatus := cidrHandler.SyncCIDRRoute(instanceSpec, false)
						cidrHandler.CleanPendingIPPools(ippoolSnapshot, name, instanceSpec)
						err := cidrHandler.MultiNicNetworkHandler.SyncStatus(name, instanceSpec, routeStatus)
						if err != nil {
							logger.Info(fmt.Sprintf("failed to update route status of %s: %v", name, err))
						}
					}
				}
			}
		}
	}()
}
