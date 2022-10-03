/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package allocator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"

	"time"
	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
)

func TestAllocator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Allocator Test Suite")
}

func genAllocation(indexes []int) []backend.Allocation {
	var allocations []backend.Allocation
	for _, index := range indexes {
		allocations = append(allocations, backend.Allocation{Index: index})
	}
	return allocations
}

var _ = Describe("Test Allocator", func() {
	initIndexes := []int{1, 2, 3, 8, 13, 18}
	allocations := genAllocation(initIndexes)

	It("find simple next available index", func() {
		indexes := []int{1, 2, 3, 8, 13, 18}
		nextIndex := FindAvailableIndex(indexes, 0)
		Expect(nextIndex).To(Equal(4))
	})

	It("find next available index with exclude range over consecutive order", func() {
		excludes := []ExcludeRange{
			ExcludeRange{
				MinIndex: 4,
				MaxIndex: 6,
			},
		}
		indexes := GenerateAllocateIndexes(allocations, 20, excludes)
		Expect(indexes).To(Equal([]int{1, 2, 3, 4, 5, 6, 8, 13, 18}))
		nextIndex := FindAvailableIndex(indexes, 0)
		Expect(nextIndex).To(Equal(7))
	})
	It("find next available index with exclude range over non-consecutive order", func() {
		excludes := []ExcludeRange{
			ExcludeRange{
				MinIndex: 4,
				MaxIndex: 7,
			},
		}

		indexes := GenerateAllocateIndexes(allocations, 20, excludes)
		Expect(indexes).To(Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 13, 18}))
		nextIndex := FindAvailableIndex(indexes, 0)
		Expect(nextIndex).To(Equal(9))
	})

	It("find next available index with exclude range over non-consecutive and then consecutive order", func() {
		excludes := []ExcludeRange{
			ExcludeRange{
				MinIndex: 4,
				MaxIndex: 7,
			},
			ExcludeRange{
				MinIndex: 9,
				MaxIndex: 12,
			},
		}

		indexes := GenerateAllocateIndexes(allocations, 20, excludes)
		Expect(indexes).To(Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 18}))
		nextIndex := FindAvailableIndex(indexes, 0)
		Expect(nextIndex).To(Equal(14))
	})

	It("force expired", func() {
		podName := "A"
		deallocateHistory[podName] = &allocateRecord{
			Time:       time.Now(),
			LastOffset: 1,
		}
		Expect(deallocateHistory[podName].Expired()).To(Equal(false))
		deallocateHistory[podName].Time = deallocateHistory[podName].Time.Add(time.Duration(-HISTORY_TIMEOUT-1) * time.Second)
		Expect(deallocateHistory[podName].Expired()).To(Equal(true))
	})
})
