/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package compute

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	SHIFT_BYTE_VAL     = 256
	MAX_VALUE_PER_BYTE = 255
	BYTE_SIZE          = 8
)

var MASKCHECK = []byte{0, 128, 192, 224, 240, 248, 252, 254, 255}

type IPValue struct {
	Address string
	Value   int64
}

func maskIndex(b byte) int {
	for index, chk := range MASKCHECK {
		if b&0xff == chk {
			return index
		}
	}
	return -1
}

func addrToValue(address string) int64 {
	splits := strings.Split(address, ".")
	var sumValue int64
	sumValue = 0
	for _, split := range splits {
		val, _ := strconv.ParseInt(split, 10, 64)
		sumValue = sumValue*SHIFT_BYTE_VAL + val
	}
	return sumValue
}

func ValueToAddr(value int64) [4]byte {
	var output [4]byte
	for index := 3; index >= 0; index-- {
		output[index] = byte(value % SHIFT_BYTE_VAL)
		value = value / SHIFT_BYTE_VAL
	}
	return output
}

func bytesToStr(ip [4]byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

func getIPValue(address string) IPValue {
	ip := strings.Split(address, "/")[0]
	return IPValue{Address: address, Value: addrToValue(ip)}
}

// SortAddress sorts a list of IP addresses and returns a list of IPValues.
func SortAddress(addresses []string) []IPValue {
	var ipValues []IPValue
	for _, address := range addresses {
		ipValue := getIPValue(address)
		ipValues = append(ipValues, ipValue)
	}
	sort.SliceStable(ipValues, func(i, j int) bool {
		return ipValues[i].Value < ipValues[j].Value
	})
	return ipValues
}
