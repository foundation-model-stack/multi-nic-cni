/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
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
		mIndex := MaskIndex(value)
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

func (c CIDRCompute) valueToBytes(addValue int) ([]byte, error) {
	var valueInBytes []byte

	s := fmt.Sprintf("%b", addValue)
	for s != "" {
		target := ""
		if len(s) > 8 {
			target = s[len(s)-8:]
			s = s[0 : len(s)-8]
		} else {
			target = s
			for len(target) < 8 {
				target = "0" + target
			}
			s = ""
		}
		if intValue, err := strconv.ParseInt(target, 2, 64); err != nil {
			return []byte{}, err
		} else {
			valueInBytes = append([]byte{byte(intValue)}, valueInBytes...)
		}
	}
	return valueInBytes, nil
}

func (c CIDRCompute) addAddress(baseAddress []byte, mask []byte, block int, addValue int) ([4]byte, error) {
	// check if valid sum value
	maxValue := math.Pow(2, float64(block)) - 1
	if addValue > int(maxValue) {
		return [4]byte{}, errors.New(fmt.Sprintf("InvalidRequest: %.0f > %d", maxValue, addValue))
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
		mIndex := MaskIndex(value)
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

func (c CIDRCompute) GetIPVLANSubnet(interfaceNodeCIDR [4]byte, subnet string, interfaceBlock int) string {
	baseBlock, _ := strconv.ParseInt(strings.Split(subnet, "/")[1], 10, 64)
	_, subnetNet, _ := net.ParseCIDR(subnet)
	mask := subnetNet.Mask
	interfaceMask := c.appendMask(mask, interfaceBlock)
	interfaceIPMask := net.IPv4Mask(interfaceMask[0], interfaceMask[1], interfaceMask[2], interfaceMask[3])
	intefaceNodeIP := net.IPv4(interfaceNodeCIDR[0], interfaceNodeCIDR[1], interfaceNodeCIDR[2], interfaceNodeCIDR[3])
	maskedInterfaceNodeIP := intefaceNodeIP.Mask(interfaceIPMask).To4()

	return fmt.Sprintf("%s/%d", maskedInterfaceNodeIP, int(baseBlock)+interfaceBlock)
}

func (c CIDRCompute) GetPodSubnet(interfaceNodeCIDR [4]byte, subnet string, interfaceBlock int, nodeBlock int) string {
	baseBlock, _ := strconv.ParseInt(strings.Split(subnet, "/")[1], 10, 64)
	blockSize := int(baseBlock) + interfaceBlock + nodeBlock
	return fmt.Sprintf("%s/%d", bytesToStr(interfaceNodeCIDR), blockSize)
}

func (c CIDRCompute) GetCIDRFromByte(cidrInByte [4]byte, subnet string, blocksize int) string {
	baseBlock, _ := strconv.ParseInt(strings.Split(subnet, "/")[1], 10, 64)
	blockSize := int(baseBlock) + blocksize
	return fmt.Sprintf("%s/%d", bytesToStr(cidrInByte), blockSize)
}
