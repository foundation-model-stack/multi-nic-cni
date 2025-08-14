/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package selector

import (
	"log"

	"github.com/foundation-model-stack/multi-nic-cni/daemon/iface"
)

type DevClassSelector struct{}

func (DevClassSelector) Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string, resourceMap map[string][]string) []string {
	if req.NicSet.DevClass != "" {
		devSpec, err := DeviceClassHandler.Get(req.NicSet.DevClass)
		if err == nil {
			devSpecMap := make(map[string][]string)
			for _, deviceID := range devSpec.DeviceIDs {
				devSpecMap[deviceID.Vendor] = deviceID.Products
			}

			for _, devName := range interfaceNameMap {
				if netAddress, exists := nameNetMap[devName]; exists {
					interfaceMap := iface.GetInterfaceInfoCache()
					if info, exists := interfaceMap[devName]; exists {
						if products, exists := devSpecMap[info.Vendor]; exists {
							found := false
							for _, product := range products {
								if product == info.Product {
									found = true
									break
								}
							}
							if !found {
								// not in expected product
								delete(interfaceNameMap, netAddress)
							}
						} else {
							// not in expected vendor
							delete(interfaceNameMap, netAddress)
						}
					}
				}
			}
		} else {
			log.Printf("cannot get device class %s: %v", req.NicSet.DevClass, err)
		}
	}
	log.Printf("no device class")
	return (DefaultSelector{}).Select(req, interfaceNameMap, nameNetMap, resourceMap)
}
