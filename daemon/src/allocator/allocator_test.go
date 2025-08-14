/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package allocator

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"time"

	"github.com/foundation-model-stack/multi-nic-cni/daemon/backend"
)

func genAllocation(indexes []int) []backend.Allocation {
	var allocations []backend.Allocation
	for _, index := range indexes {
		allocations = append(allocations, backend.Allocation{Index: index})
	}
	return allocations
}

var _ = Describe("Test Allocator", func() {

	Context("Allocate", func() {
		initIndexes := []int{1, 2, 3, 8, 13, 18}
		allocations := genAllocation(initIndexes)

		DescribeTable("FindAvailableIndex",
			func(excludes []ExcludeRange, expectedIndex []int, expected int) {
				indexes := GenerateAllocateIndexes(allocations, 20, excludes)
				Expect(indexes).To(Equal(expectedIndex))
				nextIndex := FindAvailableIndex(indexes, 0)
				Expect(nextIndex).To(Equal(expected))
			},
			Entry("no excludes", []ExcludeRange{}, []int{1, 2, 3, 8, 13, 18}, 4),
			Entry("excludes consecutive order", []ExcludeRange{
				ExcludeRange{
					MinIndex: 4,
					MaxIndex: 6,
				},
			},
				[]int{1, 2, 3, 4, 5, 6, 8, 13, 18},
				7,
			),
			Entry("excludes non-consecutive order", []ExcludeRange{
				ExcludeRange{
					MinIndex: 4,
					MaxIndex: 7,
				},
			},
				[]int{1, 2, 3, 4, 5, 6, 7, 8, 13, 18},
				9,
			),
			Entry("excludes non-consecutive and then consecutive order", []ExcludeRange{
				ExcludeRange{
					MinIndex: 4,
					MaxIndex: 7,
				},
				ExcludeRange{
					MinIndex: 9,
					MaxIndex: 12,
				},
			},
				[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 18},
				14,
			),
		)

		DescribeTable("getAddressByIndex", func(cidr string, index int, expectedIP string) {
			result := getAddressByIndex(cidr, index)
			Expect(result).To(Equal(expectedIP))
		},
			Entry("zero index", "10.0.0.0/16", 0, "10.0.0.0"),
			Entry("first index", "10.0.0.0/16", 1, "10.0.0.1"),
			Entry("shifted index", "10.0.0.0/16", 256, "10.0.1.0"),
		)

		DescribeTable("getExcludeRanges", func(cidr string, excludes []string, expected []ExcludeRange) {
			result := getExcludeRanges(cidr, excludes)
			Expect(result).To(BeEquivalentTo(expected))
		},
			Entry("empty", "10.0.0.0/16", []string{}, []ExcludeRange{}),
			Entry("inner exclude", "10.0.0.0/16", []string{"10.0.0.0/24"},
				[]ExcludeRange{
					ExcludeRange{
						MinIndex: 0,
						MaxIndex: 255,
					},
				},
			),
			Entry("multiple inner excludes", "10.0.0.0/16", []string{"10.0.0.0/24", "10.0.1.1/32"},
				[]ExcludeRange{
					ExcludeRange{
						MinIndex: 0,
						MaxIndex: 255,
					},
					ExcludeRange{
						MinIndex: 257,
						MaxIndex: 257,
					},
				},
			),
			Entry("outer exclude", "10.0.0.0/24", []string{"10.0.0.0/23"},
				[]ExcludeRange{
					ExcludeRange{
						MinIndex: 0,
						MaxIndex: 511,
					},
				},
			),
		)

		DescribeTable("allocateIP", func(interfaceNames []string, ippoolSpecMap map[string]backend.IPPoolType, expectedAddress map[string]string) {
			newAllocations := allocateIP("test-pod", "test-namespace", interfaceNames, 1, ippoolSpecMap)
			Expect(newAllocations).To(HaveLen(len(expectedAddress)))
			for ippoolName, allocation := range newAllocations {
				address, found := expectedAddress[ippoolName]
				Expect(found).To(BeTrue())
				Expect(allocation.Address).To(BeEquivalentTo(address))
			}
		},
			Entry("no interface name", []string{}, map[string]backend.IPPoolType{
				"eth0": backend.IPPoolType{InterfaceName: "eth0", PodCIDR: "192.168.0.0/24"},
			}, map[string]string{}),
			Entry("no ippool", []string{"eth0"}, map[string]backend.IPPoolType{}, map[string]string{}),
			Entry("first allocation", []string{"eth0"}, map[string]backend.IPPoolType{
				"eth0": backend.IPPoolType{InterfaceName: "eth0", PodCIDR: "192.168.0.0/24"},
			}, map[string]string{
				"eth0": "192.168.0.1",
			}),
			Entry("second allocation", []string{"eth0"}, map[string]backend.IPPoolType{
				"eth0": backend.IPPoolType{
					InterfaceName: "eth0",
					PodCIDR:       "192.168.0.0/24",
					Allocations: []backend.Allocation{
						{
							Pod:       "dummy",
							Namespace: "test-namespace",
							Index:     1,
							Address:   "192.168.0.1"},
					},
				},
			}, map[string]string{
				"eth0": "192.168.0.2",
			}),
			Entry("reuse allocation", []string{"eth0"}, map[string]backend.IPPoolType{
				"eth0": backend.IPPoolType{
					InterfaceName: "eth0",
					PodCIDR:       "192.168.0.0/24",
					Allocations: []backend.Allocation{
						{
							Pod:       "dummy",
							Namespace: "test-namespace",
							Index:     255,
							Address:   "192.168.0.255"},
					},
				},
			}, map[string]string{
				"eth0": "192.168.0.1",
			}),
		)
	})

	Context("Deallocate", func() {

		It("force expired", func() {
			podName := "A"
			deallocateHistory[podName] = &allocateRecord{
				Time:       time.Now(),
				LastOffset: 1,
			}
			Expect(deallocateHistory[podName].Expired()).To(Equal(false))
			deallocateHistory[podName].Time = deallocateHistory[podName].Time.Add(time.Duration(-HISTORY_TIMEOUT-1) * time.Second)
			Expect(deallocateHistory[podName].Expired()).To(Equal(true))
			FlushExpiredHistory()
			_, found := deallocateHistory[podName]
			Expect(found).To(BeFalse())
		})

	})

	Context("with IPPool", func() {
		podCIDR := "192.168.0.0/26"
		defName := "netname"
		ippoolName := "netname-192.168.0.0-26"
		hostName := "hostname"
		interfaceName := "eth1"

		BeforeEach(func() {
			yamlStr := fmt.Sprintf(`
apiVersion: multinic.fms.io/v1
kind: IPPool
metadata:
  name: %s
  labels:
    hostname: %s
    netname: %s
spec:
  allocations: []
  excludes: []
  hostName: %s
  interfaceName: %s
  netAttachDef: %s
  podCIDR: %s
  vlanCIDR: 192.168.0.0/18
`, ippoolName, hostName, defName, hostName, interfaceName, defName, podCIDR)
			// Create decoder
			dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

			// Decode YAML string to unstructured.Unstructured
			obj := &unstructured.Unstructured{}
			_, _, err := dec.Decode([]byte(yamlStr), nil, obj)
			Expect(err).NotTo(HaveOccurred())
			mapObj := obj.Object
			_, err = IppoolHandler.Create(mapObj, metav1.NamespaceAll, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			IppoolHandler.Delete(ippoolName, metav1.NamespaceAll, metav1.DeleteOptions{})
		})

		It("Allocate-DeallocateIP", func() {
			req := IPRequest{
				PodName:          "podA",
				PodNamespace:     "default",
				HostName:         hostName,
				NetAttachDefName: defName,
				InterfaceNames:   []string{interfaceName},
			}
			By("Allocating IP")
			responses := AllocateIP(req)
			Expect(responses).To(HaveLen(1))
			By("Deallocating IP")
			responses = DeallocateIP(req)
			Expect(responses).To(HaveLen(1))
		})

		It("CleanHangingAllocation", func() {
			By("Patching allocation", func() {
				allocations := []backend.Allocation{
					{
						Pod:       "dummyPod",
						Namespace: "default",
						Index:     10,
						Address:   "192.168.0.10",
					},
				}
				_, err := IppoolHandler.PatchIPPool(ippoolName, allocations)
				Expect(err).NotTo(HaveOccurred())
				allocs := getAllocations(ippoolName)
				Expect(allocs).To(HaveLen(1))
			})
			By("Cleaning hanging allocation")
			err := CleanHangingAllocation(hostName)
			Expect(err).NotTo(HaveOccurred())
			allocs := getAllocations(ippoolName)
			Expect(allocs).To(HaveLen(0))
		})
	})

})

var _ = Describe("Test VF/PF Interface Mapping", func() {
	var tempDir string
	var err error
	var originalNetClassDir = NetClassDir

	BeforeEach(func() {
		// Create a temporary directory for testing file system operations
		tempDir, err = os.MkdirTemp("", "vf-pf-test")
		Expect(err).ToNot(HaveOccurred())
		// Temporarily modify the path for testing
		originalNetClassDir = NetClassDir
		NetClassDir = filepath.Join(tempDir, "sys", "class", "net")
	})

	AfterEach(func() {
		// Clean up temporary directory
		os.RemoveAll(tempDir)
		NetClassDir = originalNetClassDir
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
			mockVF(tempDir, "ens9f0v1", "ens9f0np0")

			result := isVF("ens9f0v1")
			Expect(result).To(BeTrue())
		})
	})

	Describe("getPFInterfaceName function", func() {

		It("should return original name when physfn directory doesn't exist", func() {
			result := getPFInterfaceName("ens9f0v1")
			Expect(result).To(Equal("ens9f0v1"))
		})

		It("should return PF name when single PF found", func() {
			// Create a fake sys structure for a VF interface
			mockVF(tempDir, "ens9f0v1", "ens9f0np0")

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
			// Create a fake sys structure for a VF interface
			mockVF(tempDir, "ens9f0v1", "ens9f0np0")

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

			// Create a fake sys structure for a VF interface
			mockVF(tempDir, "ens9f0v1", "ens9f0np0")

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

func mockVF(tempDir, vf, pf string) {
	// Mock VF net directory
	vfNetPath := filepath.Join(tempDir, "sys", "class", "net", vf, "device", "physfn", "net")
	err := os.MkdirAll(vfNetPath, 0755)
	Expect(err).ToNot(HaveOccurred())

	// Create PF interface directory
	pfPath := filepath.Join(vfNetPath, pf)
	err = os.MkdirAll(pfPath, 0755)
	Expect(err).ToNot(HaveOccurred())
}

func getAllocations(ippoolName string) []interface{} {
	ippool, err := IppoolHandler.Get(ippoolName, metav1.NamespaceAll, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	spec := ippool.Object["spec"].(map[string]interface{})
	allocations := spec["allocations"].([]interface{})
	return allocations
}
