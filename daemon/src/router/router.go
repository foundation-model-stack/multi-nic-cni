/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package router

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/vishvananda/netlink"
)

// For L3 Configuration
type L3ConfigRequest struct {
	Name   string      `json:"name"`
	Subnet string      `json:"subnet"`
	Routes []HostRoute `json:"routes"`
	Force  bool        `json:"force"`
}

type HostRoute struct {
	Subnet        string `json:"net"`
	NextHop       string `json:"via"`
	InterfaceName string `json:"iface"`
}
type RouteUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"msg"`
}

func ApplyL3Config(r *http.Request) RouteUpdateResponse {
	res_msg := ""
	success := true
	_, tableID, devRoutesMap, err := getRoutesFromRequest(r, true)
	if err == nil {
		for dev, routes := range devRoutesMap {
			for _, route := range routes {
				exists, _ := isRouteExist(route, dev)
				log.Printf("Add route %s; (%v)", route.String(), exists)
				if !exists {
					err = netlink.RouteAdd(&route)
					if err != nil {
						res_msg += fmt.Sprintf("AddRouteError %v;", err)
						success = false
					} else {
						res_msg += fmt.Sprintf("Add route %s;", route.String())
						success = success && true
					}
				} else {
					res_msg += "Route exists"
					success = false
				}
			}
		}
	} else {
		res_msg += fmt.Sprintf("AddRoutesError %v;", err)
		success = false
	}
	log.Printf("Apply L3 config %d; (%v)", tableID, success)
	response := RouteUpdateResponse{Success: success, Message: res_msg}
	return response
}

func DeleteL3Config(r *http.Request) RouteUpdateResponse {
	tableName, tableID, _, _ := getRoutesFromRequest(r, false)
	success, res_msg := deleteL3Config(tableName, tableID)
	response := RouteUpdateResponse{Success: success, Message: res_msg}
	return response
}

func deleteL3Config(tableName string, tableID int) (bool, string) {
	res_msg := ""
	success := true

	if tableID == -1 {
		success = false
		res_msg += "Failed to get tableID"
	} else {
		err := DeleteTable(tableName, tableID)
		if err != nil {
			res_msg += err.Error()
			success = false
		}
	}
	log.Printf("Delete L3 config %s (%v): %s", tableName, success, res_msg)
	return success, res_msg
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
				Scope: netlink.SCOPE_UNIVERSE,
				Dst:   route.Dst,
			})

			err = netlink.RouteAdd(&route)
			if err != nil {
				res_msg += fmt.Sprintf("AddRouteError %v;", err)
				success = false
			} else {
				log.Printf("Successfully add route %s", route.String())
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
			Scope: netlink.SCOPE_UNIVERSE,
			Dst:   route.Dst,
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

func GetRoutes(tableID int) ([]netlink.Route, error) {
	findTable := &netlink.Route{Table: tableID}
	routeFilter := netlink.RT_FILTER_TABLE

	family := netlink.FAMILY_V4

	return netlink.RouteListFiltered(family, findTable, routeFilter)
}

func isRouteExist(cmpRoute netlink.Route, dev netlink.Link) (bool, error) {
	routes, err := netlink.RouteList(dev, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}
	for _, route := range routes {
		if route.Gw != nil && cmpRoute.Gw != nil && route.Dst != nil && cmpRoute.Dst != nil {
			if route.Dst.String() == cmpRoute.Dst.String() && route.Gw.String() == cmpRoute.Gw.String() {
				return true, nil
			}
		}
	}
	return false, nil
}

func getRoutesFromRequest(r *http.Request, addIfNotExists bool) (string, int, map[netlink.Link][]netlink.Route, error) {
	reqBody, err := ioutil.ReadAll(r.Body)
	devRoutesMap := make(map[netlink.Link][]netlink.Route)

	if err != nil {
		return "", -1, devRoutesMap, err
	}
	var req L3ConfigRequest
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		return "", -1, devRoutesMap, err
	}

	if req.Force {
		tableID, err := GetTableID(req.Name, req.Subnet, false)
		if err == nil {
			deleteL3Config(req.Name, tableID)
			log.Printf("force delete %s (%d)", req.Name, tableID)
		} else {
			log.Printf("cannot delete %s: %v", req.Name, err)
		}
	}

	tableID, err := GetTableID(req.Name, req.Subnet, addIfNotExists)
	if tableID == -1 || err != nil {
		return req.Name, tableID, devRoutesMap, err
	}

	for _, hostRoute := range req.Routes {
		dev, err := netlink.LinkByName(hostRoute.InterfaceName)
		if err != nil {
			continue
		}
		if _, ok := devRoutesMap[dev]; !ok {
			devRoutesMap[dev] = []netlink.Route{}
		}
		nextHop := net.ParseIP(hostRoute.NextHop)
		_, dst, _ := net.ParseCIDR(hostRoute.Subnet)
		route := netlink.Route{
			LinkIndex: dev.Attrs().Index,
			Scope:     netlink.SCOPE_UNIVERSE,
			Dst:       dst,
			Gw:        nextHop,
			Table:     tableID,
		}
		devRoutesMap[dev] = append(devRoutesMap[dev], route)
	}

	return req.Name, tableID, devRoutesMap, err
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
