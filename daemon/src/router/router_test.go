/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package router

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fmt"
	"os"

	"github.com/vishvananda/netlink"
)

const (
	LOCAL_TABLE_ID   = 255
	LOCAL_TABLE_NAME = "local"

	NEW_TABLE_NAME = "newtable"

	POD_RT_PATH = "/opt/rt_tables"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Test Suite")
}

var _ = BeforeSuite(func() {
	DeleteTable(NEW_TABLE_NAME, 100)
	os.Unsetenv("RT_TABLE_PATH")
})

var _ = AfterSuite(func() {
	DeleteTable(NEW_TABLE_NAME, 100)
	os.Unsetenv("RT_TABLE_PATH")
})

var _ = Describe("Test Path", func() {
	It("Set RT path", func() {
		SetRTTablePath()
		Expect(RT_TABLE_PATH).To(Equal(DEFAULT_RT_TABLE_PATH))
		os.Setenv("RT_TABLE_PATH", POD_RT_PATH)
		SetRTTablePath()
		Expect(RT_TABLE_PATH).To(Equal(POD_RT_PATH))
		os.Unsetenv("RT_TABLE_PATH")
		SetRTTablePath()
		Expect(RT_TABLE_PATH).To(Equal(DEFAULT_RT_TABLE_PATH))
	})
})

var _ = Describe("Test RT Table", func() {
	It("Get local RT", func() {
		tableID, err := GetTableID(LOCAL_TABLE_NAME, "192.168.0.0/16", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(tableID).To(Equal(LOCAL_TABLE_ID))
	})
	It("Add then delete new table.", func() {
		tableID, err := GetTableID(NEW_TABLE_NAME, "192.168.0.0/16", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(tableID).To(Equal(-1))
		tableID, err = GetTableID(NEW_TABLE_NAME, "192.168.0.0/16", true)
		Expect(err).NotTo(HaveOccurred())
		Expect(tableID).Should(BeNumerically(">", 0))
		fmt.Println("After adding table")
		routes, err := GetRoutes(tableID)
		rules, err := netlink.RuleList(netlink.FAMILY_V4)
		fmt.Printf("Routes: %v\n", routes)
		fmt.Printf("Rules: %v\n", rules)
		newTableID, err := GetTableID(NEW_TABLE_NAME, "192.168.0.0/16", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(newTableID).To(Equal(tableID))
		err = DeleteTable(NEW_TABLE_NAME, tableID)
		Expect(err).NotTo(HaveOccurred())
		fmt.Println("After deleting table")
		routes, err = GetRoutes(tableID)
		rules, err = netlink.RuleList(netlink.FAMILY_V4)
		fmt.Printf("Routes: %v\n", routes)
		fmt.Printf("Rules: %v\n", rules)
	})
})
