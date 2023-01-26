/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package iface

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

type InterfaceInfoType struct {
	InterfaceName string `json:"interfaceName"`
	NetAddress    string `json:"netAddress"`
	HostIP        string `json:"hostIP"`
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	PciAddress    string `json:"pciAddress"`
}

var LastestInterfaceMap map[string]InterfaceInfoType

func getNetAddressFromLink(devLink netlink.Link) (string, error) {
	addrs, err := netlink.AddrList(devLink, netlink.FAMILY_V4)
	devName := devLink.Attrs().Name
	if err != nil || len(addrs) == 0 {
		return "", fmt.Errorf("cannot list address on %s: %v", devName, err)
	}
	addr := addrs[0].IPNet
	if addr == nil {
		return "", fmt.Errorf("no address set on %s", devName)
	}
	return getNetAddress(addr), nil
}

func getNetAddress(v *net.IPNet) string {
	blockSize := strings.Split(v.String(), "/")[1]
	ip := v.IP.Mask(v.Mask).String()
	return ip + "/" + blockSize
}

// GetNameNetMap returns a map from interface name to network address
func GetNameNetMap() map[string]string {
	nameNetMap := make(map[string]string)
	if LastestInterfaceMap == nil {
		// update LastestInterfaceMap
		interfaces := GetInterfaces()
		if len(interfaces) == 0 {
			return nameNetMap
		}
	}
	for devName, info := range LastestInterfaceMap {
		nameNetMap[devName] = info.NetAddress
	}
	return nameNetMap
}

// GetInterfaceNameMap returns a map from network address to interface name
func GetInterfaceNameMap() map[string]string {
	ifaceNameMap := make(map[string]string)
	if LastestInterfaceMap == nil {
		// update LastestInterfaceMap
		interfaces := GetInterfaces()
		if len(interfaces) == 0 {
			return ifaceNameMap
		}
	}
	for devName, info := range LastestInterfaceMap {
		ifaceNameMap[info.NetAddress] = devName
	}
	return ifaceNameMap
}

// GetDefaultInterfaceSubNet returns default subnetwork to be omitted
func GetDefaultInterfaceSubNet() (string, error) {
	routeToDstIP, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return "", err
	}
	for _, v := range routeToDstIP {
		if v.Dst == nil {
			l, err := netlink.LinkByIndex(v.LinkIndex)
			if err != nil {
				return "", err
			}
			return getNetAddressFromLink(l)
		}
	}
	return "", fmt.Errorf("not found")
}

func GetInterfaces() []InterfaceInfoType {
	ifaceNameMap := make(map[string]InterfaceInfoType)

	interfaces := []InterfaceInfoType{}
	netDevices := GetTargetNetworks()
	defaultSubnet, err := GetDefaultInterfaceSubNet()
	if err != nil {
		log.Printf("cannot get default subnet: %v", err)
	}

	for _, netDevice := range netDevices {
		devName := netDevice.Name
		devLink, err := netlink.LinkByName(devName)
		if err != nil {
			log.Printf("cannot find link %s: %v", devName, err)
			continue
		}
		addrs, err := netlink.AddrList(devLink, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			log.Printf("cannot list address on %s: %v", devName, err)
			continue
		}
		addr := addrs[0].IPNet
		if addr == nil {
			log.Printf("no address set on %s", devName)
			continue
		}
		if devLink.Attrs().Flags&net.FlagUp == 0 {
			// interface down
			log.Printf("%s down", devName)
			continue
		}
		netAddress := getNetAddress(addr)
		if netAddress == defaultSubnet {
			// omit default
			log.Printf("omit %s (default subnet %s)", devName, netAddress)
			continue
		}

		if addr.IP.To4() != nil {
			iface := InterfaceInfoType{
				InterfaceName: devName,
				NetAddress:    netAddress,
				HostIP:        addr.IP.To4().String(),
				Vendor:        netDevice.Vendor,
				Product:       netDevice.Product,
				PciAddress:    netDevice.PciAddress,
			}
			interfaces = append(interfaces, iface)
			ifaceNameMap[devName] = iface
		}
	}
	LastestInterfaceMap = ifaceNameMap
	return interfaces
}
