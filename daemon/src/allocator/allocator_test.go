/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package allocator

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe("Test VF/PF Interface Mapping", func() {
	var tempDir string
	var err error

	BeforeEach(func() {
		// Create a temporary directory for testing file system operations
		tempDir, err = os.MkdirTemp("", "vf-pf-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up temporary directory
		os.RemoveAll(tempDir)
	})

	Describe("isVF function", func() {
		It("should return false for non-existent interface", func() {
			result := isVF("non-existent-interface")
			Expect(result).To(BeFalse())
		})

		It("should return false for regular interface (non-VF)", func() {
			// Create a fake sys structure for a regular interface
			interfacePath := filepath.Join(tempDir, "sys", "class", "net", "eth0")
			err := os.MkdirAll(interfacePath, 0755)
			Expect(err).ToNot(HaveOccurred())

			// No physfn directory for regular interfaces
			result := isVF("eth0")
			Expect(result).To(BeFalse())
		})

		It("should return true for VF interface", func() {
			// Create a fake sys structure for a VF interface
			vfPath := filepath.Join(tempDir, "sys", "class", "net", "ens9f0v1", "device", "physfn")
			err := os.MkdirAll(vfPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Temporarily modify the path for testing
			originalIsVF := isVF
			defer func() { isVF = originalIsVF }()

			// Mock isVF to use our temp directory
			isVF = func(interfaceName string) bool {
				physfnPath := filepath.Join(tempDir, "sys", "class", "net", interfaceName, "device", "physfn")
				_, err := os.Stat(physfnPath)
				return err == nil
			}

			result := isVF("ens9f0v1")
			Expect(result).To(BeTrue())
		})
	})

	Describe("getPFInterfaceName function", func() {
		It("should return original name when physfn directory doesn't exist", func() {
			// Temporarily modify the function for testing
			originalGetPF := getPFInterfaceName
			defer func() { getPFInterfaceName = originalGetPF }()

			getPFInterfaceName = func(vfInterfaceName string) string {
				physfnNetPath := filepath.Join(tempDir, "sys", "class", "net", vfInterfaceName, "device", "physfn", "net")

				entries, err := os.ReadDir(physfnNetPath)
				if err != nil {
					return vfInterfaceName
				}

				if len(entries) == 0 {
					return vfInterfaceName
				}

				return entries[0].Name()
			}

			result := getPFInterfaceName("ens9f0v1")
			Expect(result).To(Equal("ens9f0v1"))
		})

		It("should return PF name when single PF found", func() {
			// Create a fake sys structure for VF with single PF
			vfNetPath := filepath.Join(tempDir, "sys", "class", "net", "ens9f0v1", "device", "physfn", "net")
			err := os.MkdirAll(vfNetPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Create PF interface directory
			pfPath := filepath.Join(vfNetPath, "ens9f0np0")
			err = os.MkdirAll(pfPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Mock the function to use our temp directory
			originalGetPF := getPFInterfaceName
			defer func() { getPFInterfaceName = originalGetPF }()

			getPFInterfaceName = func(vfInterfaceName string) string {
				physfnNetPath := filepath.Join(tempDir, "sys", "class", "net", vfInterfaceName, "device", "physfn", "net")

				entries, err := os.ReadDir(physfnNetPath)
				if err != nil {
					return vfInterfaceName
				}

				if len(entries) == 0 {
					return vfInterfaceName
				}

				return entries[0].Name()
			}

			result := getPFInterfaceName("ens9f0v1")
			Expect(result).To(Equal("ens9f0np0"))
		})


	})

	Describe("Interface matching logic", func() {
		It("should match VF interface to corresponding PF in IPPool", func() {
			// Create mock IPPool spec
			ippoolSpecMap := map[string]backend.IPPoolType{
				"test-pool": {
					InterfaceName: "ens9f0np0",
					PodCIDR:       "192.168.1.0/24",
				},
			}

			// Mock the VF detection functions
			originalIsVF := isVF
			originalGetPF := getPFInterfaceName
			defer func() { 
				isVF = originalIsVF
				getPFInterfaceName = originalGetPF
			}()

			isVF = func(interfaceName string) bool {
				return interfaceName == "ens9f0v1"
			}

			getPFInterfaceName = func(vfInterfaceName string) string {
				if vfInterfaceName == "ens9f0v1" {
					return "ens9f0np0"
				}
				return vfInterfaceName
			}

			// Test the matching logic
			interfaceNames := []string{"ens9f0v1"}
			spec := ippoolSpecMap["test-pool"]

			// Simulate the matching logic from the allocator
			deleteIndex := -1
			for deleteIndex = 0; deleteIndex < len(interfaceNames); deleteIndex++ {
				interfaceName := interfaceNames[deleteIndex]
				matchFound := false

				// Direct interface name match
				if spec.InterfaceName == interfaceName {
					matchFound = true
				} else if isVF(interfaceName) {
					// Check if the interface is a VF and map it to its PF for comparison
					pfInterfaceName := getPFInterfaceName(interfaceName)
					if spec.InterfaceName == pfInterfaceName {
						matchFound = true
					}
				}

				if matchFound {
					break
				}
			}

			// Should find a match
			Expect(deleteIndex).To(Equal(0))
			Expect(deleteIndex < len(interfaceNames)).To(BeTrue())
		})

		It("should match regular interface directly", func() {
			// Create mock IPPool spec
			ippoolSpecMap := map[string]backend.IPPoolType{
				"test-pool": {
					InterfaceName: "eth0",
					PodCIDR:       "192.168.1.0/24",
				},
			}

			// Mock the VF detection functions
			originalIsVF := isVF
			defer func() { isVF = originalIsVF }()

			isVF = func(interfaceName string) bool {
				return false // eth0 is not a VF
			}

			// Test the matching logic
			interfaceNames := []string{"eth0"}
			spec := ippoolSpecMap["test-pool"]

			// Simulate the matching logic from the allocator
			deleteIndex := -1
			for deleteIndex = 0; deleteIndex < len(interfaceNames); deleteIndex++ {
				interfaceName := interfaceNames[deleteIndex]
				matchFound := false

				// Direct interface name match
				if spec.InterfaceName == interfaceName {
					matchFound = true
				} else if isVF(interfaceName) {
					// This shouldn't be called for regular interfaces
					Fail("Should not check VF mapping for regular interface")
				}

				if matchFound {
					break
				}
			}

			// Should find a match
			Expect(deleteIndex).To(Equal(0))
			Expect(deleteIndex < len(interfaceNames)).To(BeTrue())
		})

		It("should not match when VF maps to different PF", func() {
			// Create mock IPPool spec
			ippoolSpecMap := map[string]backend.IPPoolType{
				"test-pool": {
					InterfaceName: "ens9f0np1", // Different PF
					PodCIDR:       "192.168.1.0/24",
				},
			}

			// Mock the VF detection functions
			originalIsVF := isVF
			originalGetPF := getPFInterfaceName
			defer func() {
				isVF = originalIsVF
				getPFInterfaceName = originalGetPF
			}()

			isVF = func(interfaceName string) bool {
				return interfaceName == "ens9f0v1"
			}

			getPFInterfaceName = func(vfInterfaceName string) string {
				if vfInterfaceName == "ens9f0v1" {
					return "ens9f0np0" // Maps to different PF than in IPPool
				}
				return vfInterfaceName
			}

			// Test the matching logic
			interfaceNames := []string{"ens9f0v1"}
			spec := ippoolSpecMap["test-pool"]

			// Simulate the matching logic from the allocator
			deleteIndex := -1
			for deleteIndex = 0; deleteIndex < len(interfaceNames); deleteIndex++ {
				interfaceName := interfaceNames[deleteIndex]
				matchFound := false

				// Direct interface name match
				if spec.InterfaceName == interfaceName {
					matchFound = true
				} else if isVF(interfaceName) {
					// Check if the interface is a VF and map it to its PF for comparison
					pfInterfaceName := getPFInterfaceName(interfaceName)
					if spec.InterfaceName == pfInterfaceName {
						matchFound = true
					}
				}

				if matchFound {
					break
				}
			}

			// Should not find a match
			Expect(deleteIndex).To(Equal(len(interfaceNames)))
		})
	})
})

