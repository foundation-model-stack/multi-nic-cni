/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package selector

type Metric struct {
	name   string
	values []float64
}

func GetInterfaceStat(interfaceNameMap map[string]string) map[string]Metric {
	//TODO
	metricMap := make(map[string]Metric)
	for _, master := range interfaceNameMap {
		metricMap[master] = Metric{}
	}
	return metricMap
}

type Monitor struct {
	ifaceStat map[string]Metric
}
