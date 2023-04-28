/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package allocator

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	SHIFT_BYTE_VAL  = 256
	HISTORY_TIMEOUT = 60 // seconds

	HOSTNAME_LABEL_NAME = "hostname"
	DEFNAME_LABEL_NAME  = "netname"
)

var allocatorLock sync.Mutex

var K8sClientset *kubernetes.Clientset
var IppoolHandler *backend.IPPoolHandler

type IPValue struct {
	Address string
	Value   int64
}

type allocateRecord struct {
	time.Time
	LastOffset int
}

func (r *allocateRecord) Expired() bool {
	curr := time.Now()
	return curr.Sub(r.Time).Seconds() > HISTORY_TIMEOUT
}

var deallocateHistory map[string]*allocateRecord = make(map[string]*allocateRecord)

func FindAvailableIndex(indexes []int, leftIndex int) int {
	if len(indexes) == 0 {
		return -1
	}
	lastAllocationIndex := indexes[len(indexes)-1]
	if lastAllocationIndex-leftIndex == len(indexes) {
		// all address in the range is assigned
		// log.Printf("lastAllocationIndex - leftIndex == len(indexes), lastIndex %d, leftIndex: %d, %v", lastAllocationIndex, leftIndex, indexes)
		return -1
	} else {
		if indexes[0] != leftIndex+1 {
			// log.Printf("indexes[0] != leftIndex + 1, lastIndex %d, leftIndex: %d, %v", lastAllocationIndex, leftIndex, indexes)
			return leftIndex + 1
		}
		midIndex := len(indexes) / 2
		leftPart := indexes[0:midIndex]
		leftResult := FindAvailableIndex(leftPart, leftIndex)
		if leftResult != -1 {
			return leftResult
		}
		rightPart := indexes[midIndex:]
		rightResult := FindAvailableIndex(rightPart, leftIndex+midIndex)
		return rightResult
	}
}

func valueToAddr(value int64) [4]byte {
	var output [4]byte
	for index := 3; index >= 0; index-- {
		output[index] = byte(value % SHIFT_BYTE_VAL)
		value = value / SHIFT_BYTE_VAL
	}
	return output
}

func valueToAddrStr(value int64) string {
	ip := valueToAddr(value)
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
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

func getIPValue(address string) IPValue {
	ip := strings.Split(address, "/")[0]
	return IPValue{Address: address, Value: addrToValue(ip)}
}

func getAddressByIndex(cidr string, index int) string {
	startIPInIpValue := getIPValue(cidr)
	addressByIndex := startIPInIpValue.Value + int64(index)
	return valueToAddrStr(addressByIndex)
}

type ExcludeRange struct {
	MinIndex int
	MaxIndex int
}

func (r ExcludeRange) Contains(index int) bool {
	return index >= r.MinIndex && index <= r.MaxIndex
}

func getExcludeRanges(cidr string, excludes []string) []ExcludeRange {
	exludeRanges := []ExcludeRange{}
	startIPInIpValue := getIPValue(cidr)

	for _, exclude := range excludes {
		excludeInIPValue := getIPValue(exclude)
		excludeStartIndex := int(excludeInIPValue.Value - startIPInIpValue.Value)
		if excludeStartIndex < 0 {
			log.Println(fmt.Sprintf("exclude index %d < 0: %s", excludeStartIndex, exclude))
		} else {
			excludeIPSplits := strings.Split(exclude, "/")
			excludeBlock := int64(32)
			if len(excludeIPSplits) >= 2 {
				excludeBlock, _ = strconv.ParseInt(excludeIPSplits[1], 10, 64)
			}
			availableBlock := 32 - excludeBlock
			maxIndex := int(math.Pow(2, float64(availableBlock)) - 1)
			r := ExcludeRange{
				MinIndex: excludeStartIndex,
				MaxIndex: excludeStartIndex + maxIndex,
			}
			exludeRanges = append(exludeRanges, r)

		}
	}
	return exludeRanges
}

func GenerateAllocateIndexes(allocations []backend.Allocation, maxIndex int, excludes []ExcludeRange) []int {
	indexes := []int{}
	for _, allocation := range allocations {
		indexes = append(indexes, allocation.Index)
	}
	for _, exclude := range excludes {
		maxValue := int(math.Min(float64(exclude.MaxIndex), float64(maxIndex)))
		for excludeI := exclude.MinIndex; excludeI <= maxValue; excludeI++ {
			indexes = append(indexes, excludeI)
		}
	}
	sort.Ints(indexes)
	return indexes
}

func AllocateIP(req IPRequest) []IPResponse {
	podName := req.PodName
	podNamespace := req.PodNamespace
	defName := req.NetAttachDefName
	hostName := req.HostName
	interfaceNames := req.InterfaceNames

	FlushExpiredHistory()
	offset := 1
	if record, ok := deallocateHistory[podName]; ok {
		// anomaly
		record.LastOffset += 1
		offset = record.LastOffset
		log.Printf("Found anomaly allocating %s: %d\n", podName, offset)
	}

	var responses []IPResponse
	startAllocate := time.Now()
	allocatorLock.Lock()
	labelMap := map[string]string{HOSTNAME_LABEL_NAME: hostName, DEFNAME_LABEL_NAME: defName}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	ippoolSpecMap, err := IppoolHandler.ListIPPool(listOptions)
	if err != nil {
		return responses
	}

	for ippoolName, _ := range ippoolSpecMap {
		if len(interfaceNames) == 0 {
			// no more interfaces to allocate
			log.Println("No more interfaces to assign")
			break
		}
		spec := ippoolSpecMap[ippoolName]
		deleteIndex := -1
		for deleteIndex = 0; deleteIndex < len(interfaceNames); deleteIndex++ {
			interfaceName := interfaceNames[deleteIndex]
			if spec.InterfaceName == interfaceName {
				break
			}
		}
		if deleteIndex >= 0 && deleteIndex != len(interfaceNames) {
			interfaceNames = append(interfaceNames[0:deleteIndex], interfaceNames[deleteIndex+1:]...)
		} else {
			// not match
			log.Printf("Interface %s is not requested by %v\n", spec.InterfaceName, interfaceNames)
			continue
		}

		podCIDR := spec.PodCIDR
		allocations := spec.Allocations
		cidrBlockStr := strings.Split(podCIDR, "/")[1]
		cirdBlock, _ := strconv.ParseInt(cidrBlockStr, 10, 64)
		excludes := spec.Excludes

		exludeRanges := getExcludeRanges(podCIDR, excludes)
		availableBlock := 32 - cirdBlock
		maxIndex := math.Pow(2, float64(availableBlock)) - 2 // except broadcast address
		indexes := GenerateAllocateIndexes(allocations, int(maxIndex), exludeRanges)
		log.Printf("exclude %v, indexes %v\n", exludeRanges, indexes)
		var nextIndex int
		if len(indexes) > 0 {
			lastIndex := indexes[len(indexes)-1]
			nextIndex = lastIndex + offset
		} else {
			nextIndex = offset // except network address
		}

		nextAddress := ""
		if nextIndex < int(maxIndex) {
			nextAddress = getAddressByIndex(podCIDR, nextIndex)
		} else {
			nextIndex = FindAvailableIndex(indexes, 0)
			if nextIndex != -1 {
				nextAddress = getAddressByIndex(podCIDR, nextIndex)
			}
		}
		if nextAddress != "" {
			newAllocation := backend.Allocation{
				Pod:       podName,
				Namespace: podNamespace,
				Index:     nextIndex,
				Address:   nextAddress,
			}
			log.Println(newAllocation)
			toInsertIndex := -1
			for allocationIndex, allocation := range allocations {
				if allocation.Index > newAllocation.Index {
					toInsertIndex = allocationIndex
				}
			}
			if toInsertIndex == -1 {
				allocations = append(allocations, newAllocation)
			} else {
				appendedAllocation := append(allocations[0:toInsertIndex], newAllocation)
				allocations = append(appendedAllocation, allocations[toInsertIndex:]...)
			}

			_, err = IppoolHandler.PatchIPPool(ippoolName, allocations)
			if err == nil {
				response := IPResponse{
					InterfaceName: spec.InterfaceName,
					IPAddress:     nextAddress,
					VLANBlockSize: strings.Split(spec.VlanCIDR, "/")[1],
				}
				log.Println(fmt.Sprintf("Append response %v (ip=%s)", response, nextAddress))
				responses = append(responses, response)
			} else {
				log.Println(fmt.Sprintf("Cannot patch IPPool: %v", err))
			}
		} else {
			log.Println(fmt.Sprintf("Cannot get NextAddress for %s", podCIDR))
		}
	}
	allocatorLock.Unlock()

	elapsed := time.Since(startAllocate)
	log.Println(fmt.Sprintf("Allocate elapsed: %d us", int64(elapsed/time.Microsecond)))
	return responses
}

func getPod(podName, podNamespace string) (*corev1.Pod, error) {
	return K8sClientset.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})

}
func CleanHangingAllocation(hostName string) error {
	labelMap := map[string]string{HOSTNAME_LABEL_NAME: hostName}
	// hostName suffix
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	ippoolSpecMap, err := IppoolHandler.ListIPPool(listOptions)
	if err != nil {
		return err
	}
	for ippoolName, _ := range ippoolSpecMap {
		spec := ippoolSpecMap[ippoolName]
		allocations := spec.Allocations
		remains := []backend.Allocation{}
		for _, allocation := range allocations {
			_, err := getPod(allocation.Pod, allocation.Namespace)
			if err == nil {
				remains = append(remains, allocation)
			}
		}
		_, err = IppoolHandler.PatchIPPool(ippoolName, remains)
		if err != nil {
			log.Println(fmt.Sprintf("Cannot patch IPPool: %v", err))
		}
	}
	return nil
}

func DeallocateIP(req IPRequest) []IPResponse {
	podName := req.PodName
	podNamespace := req.PodNamespace
	defName := req.NetAttachDefName
	hostName := req.HostName

	// set first record
	if _, ok := deallocateHistory[podName]; !ok {
		log.Printf("Add %s to deallocateHistory\n", podName)
		deallocateHistory[podName] = &allocateRecord{
			Time:       time.Now(),
			LastOffset: 1,
		}
	}

	var responses []IPResponse
	startDeallocate := time.Now()
	allocatorLock.Lock()
	labelMap := map[string]string{HOSTNAME_LABEL_NAME: hostName, DEFNAME_LABEL_NAME: defName}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}
	ippoolSpecMap, err := IppoolHandler.ListIPPool(listOptions)
	if err != nil {
		return responses
	}
	for ippoolName, _ := range ippoolSpecMap {
		spec := ippoolSpecMap[ippoolName]
		if spec.NetAttachDefName == defName && strings.Contains(spec.HostName, hostName) {
			allocations := spec.Allocations

			for index, allocation := range allocations {
				if allocation.Pod == podName && allocation.Namespace == podNamespace {
					allocations = append(allocations[0:index], allocations[index+1:]...)
					_, err = IppoolHandler.PatchIPPool(ippoolName, allocations)
					if err != nil {
						log.Println(fmt.Sprintf("Cannot patch IPPool: %v", err))
					}
					break
				}
			}
		}
	}
	allocatorLock.Unlock()

	elapsed := time.Since(startDeallocate)
	log.Println(fmt.Sprintf("Deallocate elapsed: %d us", int64(elapsed/time.Microsecond)))
	return responses
}

func FlushExpiredHistory() {
	for podName, record := range deallocateHistory {
		if record.Expired() {
			log.Printf("Flush expired deallocateHistory: %s\n", podName)
			delete(deallocateHistory, podName)
		}
	}
}
