/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package selector

type CostOptSelector struct {}

func (CostOptSelector) Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string) []string {
	// TODO
	return (DefaultSelector{}).Select(req, interfaceNameMap, nameNetMap)
}