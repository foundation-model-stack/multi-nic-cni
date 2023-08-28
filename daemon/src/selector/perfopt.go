/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package selector

type PerfOptSelector struct{}

// PerfOptSelector
func (PerfOptSelector) Select(req NICSelectRequest, interfaceNameMap map[string]string, nameNetMap map[string]string, resourceMap map[string][]string) []string {
	// TODO
	return (DefaultSelector{}).Select(req, interfaceNameMap, nameNetMap, resourceMap)
}
