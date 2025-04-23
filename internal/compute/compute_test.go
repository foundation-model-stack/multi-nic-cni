package compute_test

import (
	. "github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test CIDRCompute", func() {
	compute := CIDRCompute{}

	DescribeTable("ComputeNet", func(baseCIDR string, index int, blocksize int, expectedOutput string, expectedError bool) {
		vlanInByte, err := compute.ComputeNet(baseCIDR, index, blocksize)
		Expect(err == nil).To(BeEquivalentTo(!expectedError))
		if !expectedError {
			vlanCIDR := compute.GetCIDRFromByte(vlanInByte, baseCIDR, blocksize)
			Expect(vlanCIDR).To(BeEquivalentTo(expectedOutput))
		}
	},
		Entry("simple", "192.168.0.0/16", 0, 2, "192.168.0.0/18", false),
		Entry("invalid CIDR", "192.168.0.0", 0, 2, "", true),
		Entry("invalid Index", "192.168.0.0/16", 4, 2, "", true),
	)

	DescribeTable("CheckIfTabuIndex", func(baseCIDR string, index int, blocksize int, excludes []string, expected bool) {
		output := compute.CheckIfTabuIndex(baseCIDR, index, blocksize, excludes)
		Expect(output).To(BeEquivalentTo(expected))
	},
		Entry("no excludes", "192.168.0.0/16", 0, 8, []string{}, false),
		Entry("tabu index", "192.168.0.0/16", 0, 8, []string{"192.168.0.0/24"}, true),
		Entry("cover tabu index", "192.168.0.0/16", 0, 8, []string{"192.168.0.0/8"}, true),
		Entry("not tabu index", "192.168.0.0/16", 0, 8, []string{"192.168.1.0/24"}, false),
	)

	DescribeTable("FindAvailableIndex", func(indexes []int, leftIndex, startIndex, expected int) {
		output := compute.FindAvailableIndex(indexes, leftIndex, startIndex)
		Expect(output).To(BeEquivalentTo(expected))
	},
		Entry("empty indexes", []int{}, 0, 0, -1),
		Entry("single index assigned", []int{0}, 0, 0, -1),
		Entry("single index available", []int{1}, 0, 0, 0),
		Entry("multiple index assigned", []int{0, 1, 2}, 0, 0, -1),
		Entry("available index in left part", []int{0, 2, 3}, 0, 0, 1),
		Entry("available index in right part", []int{0, 1, 3}, 0, 0, 2),
		Entry("available index in middle", []int{0, 1, 3, 4}, 0, 0, 2),
	)

	DescribeTable("GetIndexInRange", func(podCIDR string, podIPAddress string, expectedContains bool, expectedPodIndex int) {
		contains, podIndex := compute.GetIndexInRange(podCIDR, podIPAddress)
		Expect(contains).To(BeEquivalentTo(expectedContains))
		Expect(podIndex).To(BeEquivalentTo(expectedPodIndex))
	},
		Entry("valid CIDR and IP", "192.168.1.0/24", "192.168.1.100", true, 100),
		Entry("invalid CIDR and IP", "192.168.1.0/24", "192.168.2.100", false, -1),
		Entry("same CIDR and IP", "192.168.1.0/24", "192.168.1.0", true, 0),
		Entry("different CIDR and IP", "10.0.0.0/8", "192.168.1.100", false, -1),
	)
})
