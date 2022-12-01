package controllers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

func RunPeriodicUpdate(ticker *time.Ticker, cidrHandler *CIDRHandler, hostInterfaceReconciler *HostInterfaceReconciler, logger logr.Logger, quit chan struct{}) {
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				// update interface

				logger.Info(fmt.Sprintf("synchronizing state... %d HostInterfaces, %d CIDRs", len(HostInterfaceCache), len(CIDRCache)))
				for _, instance := range HostInterfaceCache {
					hostInterfaceReconciler.UpdateInterfaces(instance)
				}
				for name, instanceSpec := range CIDRCache {
					routeStatus := cidrHandler.SyncCIDRRoute(instanceSpec, false)
					cidrHandler.CleanPendingIPPools(name, instanceSpec)
					err := cidrHandler.MultiNicNetworkHandler.SyncStatus(name, instanceSpec, routeStatus)
					if err != nil {
						logger.Info(fmt.Sprintf("failed to update route status of %s: %v", name, err))
					}
				}
			}
		}
	}()
}
