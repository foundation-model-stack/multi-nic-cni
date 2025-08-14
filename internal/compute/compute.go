/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package compute

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

type CIDRCompute struct{}

func (c CIDRCompute) appendMask(baseMask []byte, block int) [4]byte {
	remain := block
	var output [4]byte
	for index, value := range baseMask {
		if value&0xff == 255 || remain == 0 {
			output[index] = value
			continue
		}
		mIndex := maskIndex(value)
		addable := BYTE_SIZE - mIndex

		if remain > addable {
			remain = remain - addable
			output[index] = 255 & 0xff
		} else {
			output[index] = MASKCHECK[mIndex+remain]
			remain = 0
		}
	}
	return output
}

// addAddress adds the addValue to the baseAddress with the mask.
// It returns the new IP address and an error if the addValue is invalid.
func (c CIDRCompute) addAddress(baseAddress []byte, mask []byte, block int, addValue int) ([4]byte, error) {
	// check if valid sum value
	maxValue := math.Pow(2, float64(block)) - 1
	if addValue > int(maxValue) {
		return [4]byte{}, fmt.Errorf("InvalidRequest: %.0f > %d", maxValue, addValue)
	}

	// get value in binary length equals to block
	valueInBinary := fmt.Sprintf("%b", addValue)
	for len(valueInBinary) < block {
		valueInBinary = "0" + valueInBinary
	}

	var output [4]byte
	for index, value := range mask {
		if value&0xff == 255 || len(valueInBinary) == 0 {
			output[index] = baseAddress[index]
			continue
		}
		mIndex := maskIndex(value)
		addable := BYTE_SIZE - mIndex
		target := ""
		if block > addable {
			block = block - addable
			target = valueInBinary[0:addable]

			valueInBinary = valueInBinary[addable:]
		} else {
			target = valueInBinary
			valueInBinary = ""
		}
		for i := 0; i < mIndex; i++ {
			target = "0" + target
		}
		for len(target) < 8 {
			target = target + "0"
		}
		if intValue, err := strconv.ParseInt(target, 2, 64); err != nil {
			return [4]byte{}, err
		} else {
			output[index] = baseAddress[index] + byte(intValue)
		}
	}

	// confirm mask not change
	newIP := net.IPv4(output[0], output[1], output[2], output[3])
	baseIP := net.IPv4(baseAddress[0], baseAddress[1], baseAddress[2], baseAddress[3])
	if newIP.Mask(mask).String() != baseIP.Mask(mask).String() {
		return [4]byte{}, errors.New("InvalidRequest: out of mask")
	}

	return output, nil
}

// CheckIfTabuIndex checks if the given index is tabu in the context of the base CIDR and excludes.
func (c CIDRCompute) CheckIfTabuIndex(baseCIDR string, index int, blocksize int, excludes []string) bool {
	baseBlock, _ := strconv.ParseInt(strings.Split(baseCIDR, "/")[1], 10, 64)
	for _, exclude := range excludes {
		excludeIPSplits := strings.Split(exclude, "/")
		if len(excludeIPSplits) >= 2 {
			excludeBlock, _ := strconv.ParseInt(excludeIPSplits[1], 10, 64)
			if excludeBlock <= baseBlock+int64(blocksize) {
				_, excludeNet, _ := net.ParseCIDR(exclude)
				netInByte, _ := c.ComputeNet(baseCIDR, index, blocksize)
				netIP, _, _ := net.ParseCIDR(bytesToStr(netInByte) + "/32")
				if excludeNet.Contains(netIP) {
					return true
				}
			}
		}
	}
	return false
}

// FindAvailableIndex returns the first reusable index in the given range of indexes.
func (c CIDRCompute) FindAvailableIndex(indexes []int, leftIndex int, startIndex int) int {
	if len(indexes) == 0 {
		return -1
	}
	lastAllocationIndex := indexes[len(indexes)-1]
	if lastAllocationIndex-leftIndex == len(indexes)-1+startIndex {
		// all address in the range is assigned
		return -1
	} else {
		if indexes[0] != leftIndex+startIndex {
			return leftIndex + startIndex
		}
		midIndex := len(indexes) / 2
		leftPart := indexes[0:midIndex]
		leftResult := c.FindAvailableIndex(leftPart, leftIndex, startIndex)
		if leftResult != -1 {
			return leftResult
		}
		rightPart := indexes[midIndex:]
		rightResult := c.FindAvailableIndex(rightPart, leftIndex+midIndex, startIndex)
		return rightResult
	}
}

// ComputeNet computes the network address for a given CIDR and index.
// It takes the base CIDR, index, and blocksize as input parameters.
// It returns the network address as a [4]byte array and an error if any.
func (c CIDRCompute) ComputeNet(baseCIDR string, index int, blocksize int) ([4]byte, error) {
	startIPStr := strings.Split(baseCIDR, "/")[0]
	startIP := net.ParseIP(startIPStr)
	_, subnetNet, err := net.ParseCIDR(baseCIDR)
	if err != nil {
		return [4]byte{}, err
	}
	mask := subnetNet.Mask
	interfaceMask := c.appendMask(mask, blocksize)
	interfaceIPMask := net.IPv4Mask(interfaceMask[0], interfaceMask[1], interfaceMask[2], interfaceMask[3])
	baseIP := startIP.Mask(interfaceIPMask).To4()

	cidrInByte, err := c.addAddress(baseIP, mask, blocksize, index)
	if err != nil {
		return [4]byte{}, err
	}
	return cidrInByte, nil
}

// GetCIDRFromByte returns a CIDR string from a byte array, subnet, and block size.
func (c CIDRCompute) GetCIDRFromByte(cidrInByte [4]byte, subnet string, blocksize int) string {
	baseBlock, _ := strconv.ParseInt(strings.Split(subnet, "/")[1], 10, 64)
	blockSize := int(baseBlock) + blocksize
	return fmt.Sprintf("%s/%d", bytesToStr(cidrInByte), blockSize)
}

// GetIndexInRange returns a boolean indicating if the pod IP address is within the pod CIDR range,
// and the index of the pod IP address within the range.
func (c CIDRCompute) GetIndexInRange(podCIDR string, podIPAddress string) (bool, int) {
	startPodIP, podNet, _ := net.ParseCIDR(podCIDR)
	netIP, _, _ := net.ParseCIDR(podIPAddress + "/32")
	if !podNet.Contains(netIP) {
		return false, -1
	}
	cidrValues := strings.Split(startPodIP.String(), ".")
	podValues := strings.Split(podIPAddress, ".")

	podIndex := int(0)
	for index, value := range podValues {
		if value == cidrValues[index] {
			continue
		}
		cidrValue, _ := strconv.Atoi(cidrValues[index])
		podValue, _ := strconv.Atoi(podValues[index])
		diff := podValue - cidrValue
		diff *= int(math.Pow(256, float64(3-index)))
		podIndex += int(diff)
	}
	return true, podIndex
}
