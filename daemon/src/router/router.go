/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package router

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"log"

	"github.com/vishvananda/netlink"
)

// For L3 Configuration
type HostRoute struct {
	Subnet        string `json:"net"`
	NextHop       string `json:"via"`
	InterfaceName string `json:"iface"`
}
type RouteUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"msg"`
}

func AddRoute(r *http.Request) RouteUpdateResponse {
	res_msg := ""
	var success bool
	route, dev, err := getRouteFromRequest(r)
	if err == nil {
		exists, _ := isRouteExist(route, dev) 
		log.Printf("Add route %s; (%v)", route.String(), exists)
		if !exists {
			// delete unequal existing route first
			err = netlink.RouteDel(&netlink.Route{
				Scope:     netlink.SCOPE_UNIVERSE,
				Dst:       route.Dst,
			})

			err = netlink.RouteAdd(&route)
			if err != nil {
				res_msg += fmt.Sprintf("AddRouteError %v;", err)
				success = false
			} else {
				res_msg += fmt.Sprintf("Add route %s;", route.String())
				success = true
			}
		} else {
			res_msg += "Route exists"
			success = false
		}
	} else {
		res_msg += fmt.Sprintf("GetRouteError %v;", err)
		success = false
	}
	response := RouteUpdateResponse{Success: success, Message: res_msg}
	return response
}

func DeleteRoute(r *http.Request) RouteUpdateResponse {
	res_msg := ""
	var success bool
	route, _, err := getRouteFromRequest(r)
	if err == nil {
		err = netlink.RouteDel(&netlink.Route{
			Scope:     netlink.SCOPE_UNIVERSE,
			Dst:       route.Dst,
		})
		if err != nil {
			res_msg += fmt.Sprintf("DeleteRouteError %v;", err)
			success = false
		} else {
			res_msg += fmt.Sprintf("Delete route %s;", route.String())
			success = true
		}
	} else {
		res_msg += fmt.Sprintf("GetRouteError %v;", err)
		success = false
	}
	response := RouteUpdateResponse{Success: success, Message: res_msg}
	return response
}

func isRouteExist(cmpRoute netlink.Route, dev netlink.Link) (bool, error) {
	routes, err := netlink.RouteList(dev, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}
	for _, route := range routes {
		if route.Gw != nil && cmpRoute.Gw != nil {
			if route.Dst.String() == cmpRoute.Dst.String() && route.Gw.String() == cmpRoute.Gw.String() {
				return true, nil
			}
		}
	}
	return false, nil
}

func getRouteFromRequest(r *http.Request) (netlink.Route, netlink.Link, error) {
	reqBody, err := ioutil.ReadAll(r.Body)
	var route netlink.Route
	var dev netlink.Link
	if err != nil {
		return route, dev, err
	}
	var req HostRoute
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		return route, dev, err
	}
	dev, err = netlink.LinkByName(req.InterfaceName)
	if err != nil {
		return route, dev, err
	}
	nextHop := net.ParseIP(req.NextHop)
	_, dst, _ := net.ParseCIDR(req.Subnet)
	route = netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       dst,
		Gw:        nextHop,
	}
	return route, dev, err
}
