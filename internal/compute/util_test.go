package compute_test

import (
	. "github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Compute Utils", func() {
	DescribeTable("SortAddress", func(addressees []string, expected []IPValue) {
		ips := SortAddress(addressees)
		Expect(ips).To(BeEquivalentTo(ips))
	},
		Entry("empty slice", []string{}, []IPValue{}),
		Entry("single ip", []string{"0.0.1.0"}, []IPValue{{Address: "0.0.1.0", Value: 256}}),
		Entry("sorted ips", []string{"0.0.0.1", "0.0.0.2"},
			[]IPValue{{Address: "0.0.0.2", Value: 2}, {Address: "0.0.0.1", Value: 1}}),
		Entry("unsorted ips", []string{"0.0.0.2", "0.0.0.1"},
			[]IPValue{{Address: "0.0.0.2", Value: 2}, {Address: "0.0.0.1", Value: 1}}),
	)
})
