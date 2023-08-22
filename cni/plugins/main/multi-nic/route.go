/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"fmt"
	"log"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/plugins/pkg/utils"
	"github.com/vishvananda/netlink"
)

func separateMultiPathRoutes(ipamRoutes []*types.Route) (multiPathRoutes map[string][]*netlink.NexthopInfo, nonMultiPathRoutes []*types.Route) {
	nonMultiPathRoutes = []*types.Route{}
	multiPathRoutes = make(map[string][]*netlink.NexthopInfo)

	dstMap := make(map[string][]*types.Route)

	for _, r := range ipamRoutes {
		copiedRoute := r.Copy()
		dst := r.Dst.String()
		if _, found := dstMap[dst]; !found {
			dstMap[dst] = []*types.Route{copiedRoute}
		} else {
			dstMap[dst] = append(dstMap[dst], copiedRoute)
		}
	}
	log.Printf("dstMap: %v", dstMap)

	for dst, routes := range dstMap {
		if len(routes) == 1 {
			nonMultiPathRoutes = append(nonMultiPathRoutes, routes[0])
		} else {
			// For multipath routes
			nexthops := []*netlink.NexthopInfo{}
			for _, route := range routes {
				// Create the first next-hop info with gateway1 and weight 1
				nexthop := &netlink.NexthopInfo{
					Gw: route.GW,
				}
				nexthops = append(nexthops, nexthop)
			}
			// Append a route
			multiPathRoutes[dst] = nexthops
		}
	}
	return
}

func addMultiPathRoutes(iface *net.Interface, multiPathRoutes map[string][]*netlink.NexthopInfo) error {
	if multiPathRoutes != nil {
		for dst, nexthops := range multiPathRoutes {
			_, destIPNet, err := net.ParseCIDR(dst)
			if err == nil {
				multipathRoute := &netlink.Route{
					Dst:       destIPNet,
					LinkIndex: iface.Index,
					MultiPath: nexthops,
				}

				if err := netlink.RouteAddEcmp(multipathRoute); err != nil {
					return fmt.Errorf("failed to add multipath route to %s: %v", multipathRoute.Dst.IP.To4().String(), err)
				}
			}
		}
	}
	return nil
}

func delMultiPathRoutes(iface *net.Interface, multiPathRoutes map[string][]*netlink.NexthopInfo) {
	if multiPathRoutes != nil {
		for dst, nexthops := range multiPathRoutes {
			_, destIPNet, err := net.ParseCIDR(dst)
			if err == nil {
				multipathRoute := &netlink.Route{
					Dst:       destIPNet,
					LinkIndex: iface.Index,
					MultiPath: nexthops,
				}
				if err := netlink.RouteDel(multipathRoute); err != nil {
					utils.Logger.Debug(fmt.Sprintf("failed to delete multipath route to %s: %v", multipathRoute.Dst.IP.To4().String(), err))
				}
			}
		}
	}
}
