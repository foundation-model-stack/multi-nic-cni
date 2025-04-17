/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package compute

import (
	"fmt"
	"net"
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

func MaskIndex(b byte) int {
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

func valueToAddrStr(value int64) string {
	ip := ValueToAddr(value)
	return bytesToStr(ip)
}

func GetIPValue(address string) IPValue {
	ip := strings.Split(address, "/")[0]
	return IPValue{Address: address, Value: addrToValue(ip)}
}

func SortAddress(addresses []string) []IPValue {
	var ipValues []IPValue
	for _, address := range addresses {
		ipValue := GetIPValue(address)
		ipValues = append(ipValues, ipValue)
	}
	sort.SliceStable(ipValues, func(i, j int) bool {
		return ipValues[i].Value < ipValues[j].Value
	})
	return ipValues
}

func GetMinMaxValue(subnet string) (int64, int64) {
	_, subnetNet, _ := net.ParseCIDR(subnet)
	baseIP := subnetNet.IP
	mask := subnetNet.Mask

	var sumMaxValue, sumMinValue int64
	sumMaxValue = 0
	sumMinValue = 0
	for index, value := range mask {
		if value&0xff == 255 {
			sumMaxValue = sumMaxValue*SHIFT_BYTE_VAL + int64(baseIP[index])
			sumMinValue = sumMinValue*SHIFT_BYTE_VAL + int64(baseIP[index])
			continue
		}
		if value&0xff == 0 {
			sumMaxValue = sumMaxValue*SHIFT_BYTE_VAL + MAX_VALUE_PER_BYTE
			sumMinValue = sumMinValue * SHIFT_BYTE_VAL
			continue
		}
		mIndex := MaskIndex(value)
		fillUpValue := 255 - int64(MASKCHECK[mIndex])
		sumMaxValue = sumMaxValue*SHIFT_BYTE_VAL + int64(baseIP[index]) + fillUpValue
		sumMinValue = sumMinValue*SHIFT_BYTE_VAL + int64(baseIP[index])
	}
	return sumMinValue, sumMaxValue
}

func GetPreviousAddress(lastAddress string) string {
	lastAddressInIpValue := GetIPValue(lastAddress)
	prevValue := lastAddressInIpValue.Value - 1
	return valueToAddrStr(prevValue)
}

func GetAddressByIndex(cidr string, index int) string {
	startIP := strings.Split(cidr, "/")[0]
	startIPInIpValue := GetIPValue(startIP)
	addressByIndex := startIPInIpValue.Value + int64(index)
	return valueToAddrStr(addressByIndex)
}
