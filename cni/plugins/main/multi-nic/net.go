/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"net"
	"strings"
)

// getInterfaceNameFromNetAddress returns container interface name from net address
func getInterfaceNameFromNetAddress(targetNet string) string {
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil {
					ifaceNet := getNetAddress(v)
					if ifaceNet == targetNet {
						return i.Name
					}
				}
			}
		}
	}
	return ""
}

// getMinimumIndexNetAddress returns net address with minimum index (default net such as ens3 eth0)
func getMinimumIndexNetAddress() string {
	ifaces, _ := net.Interfaces()
	minIndex := 99
	ifaceNet := ""
	for _, i := range ifaces {
		if i.Index > minIndex {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil {
					minIndex = i.Index
					ifaceNet = getNetAddress(v)
					break
				}
			}
		}
	}
	return ifaceNet
}

// getNetAddress converts IPNet to string
func getNetAddress(v *net.IPNet) string {
	blockSize := strings.Split(v.String(), "/")[1]
	ip := v.IP.Mask(v.Mask).String()
	return ip + "/" + blockSize
}
