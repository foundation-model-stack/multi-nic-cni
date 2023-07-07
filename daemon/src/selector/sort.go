/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package selector

import "sort"

type KV struct {
	Key   string
	Value int
}

type ByValue []KV

func (b ByValue) Len() int {
	return len(b)
}

func (b ByValue) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b ByValue) Less(i, j int) bool {
	return b[i].Value < b[j].Value
}

func getSortedKeyByMap(valueMap map[string]int) []string {
	var kvs []KV
	for key, value := range valueMap {
		kvs = append(kvs, KV{key, value})
	}
	sort.Sort(sort.Reverse(ByValue(kvs)))
	sortedKey := []string{}
	for _, kv := range kvs {
		sortedKey = append(sortedKey, kv.Key)
	}
	return sortedKey
}
