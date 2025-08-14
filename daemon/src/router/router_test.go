/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"os"

	"github.com/vishvananda/netlink"
)

func TestRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Router Test Suite")
}

var _ = Describe("Test Route", Ordered, func() {

	Context("RT Path", func() {
		POD_RT_PATH := "/opt/rt_tables"
		LOCAL_TABLE_ID := 255
		LOCAL_TABLE_NAME := "local"

		BeforeAll(func() {
			os.Unsetenv("RT_TABLE_PATH")
		})

		AfterAll(func() {
			os.Unsetenv("RT_TABLE_PATH")
		})

		It("Get local RT", func() {
			tableID, err := GetTableID(LOCAL_TABLE_NAME, "192.168.0.0/16", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(tableID).To(Equal(LOCAL_TABLE_ID))
		})

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

	Context("Table", func() {
		var tableID int
		var dst = "192.168.0.0/24"
		var route HostRoute
		var testTableName = "newtable"

		BeforeAll(func() {
			DeleteTable(testTableName, 100)
			route = HostRoute{
				Subnet:        dst,
				NextHop:       "0.0.0.0",
				InterfaceName: getValidIface(),
			}
		})

		AfterAll(func() {
			DeleteTable(testTableName, 100)
		})

		BeforeEach(func() {
			By("Adding new table")
			var err error
			tableID, err = GetTableID(testTableName, "192.168.0.0/16", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(tableID).To(Equal(-1))
			By("Getting table")
			tableID, err = GetTableID(testTableName, "192.168.0.0/16", true)
			Expect(err).NotTo(HaveOccurred())
			Expect(tableID).Should(BeNumerically(">", 0))
			_, err = GetRoutes(tableID)
			Expect(err).NotTo(HaveOccurred())
			rules, err := netlink.RuleList(netlink.FAMILY_V4)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(rules)).Should(BeNumerically(">", 0))
		})

		AfterEach(func() {
			presentTableID, err := GetTableID(testTableName, "192.168.0.0/16", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(tableID).Should(BeNumerically(">", 0))
			Expect(presentTableID).To(Equal(tableID))
			By("Deleting table")
			err = DeleteTable(testTableName, tableID)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("getRoutesFromL3Config", Ordered, func() {

			It("not-existing table", func() {
				notExistName := "notexists"
				req := L3ConfigRequest{
					Name:   notExistName,
					Subnet: "192.168.0.0/16",
					Routes: []HostRoute{route},
				}
				reqName, tid, devRoutesMap, err := getRoutesFromL3Config(req, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(tid).To(Equal(-1))
				Expect(devRoutesMap).To(HaveLen(0))
				Expect(reqName).To(Equal(notExistName))
			})

			It("valid table", func() {
				req := L3ConfigRequest{
					Name:   testTableName,
					Subnet: "192.168.0.0/16",
					Routes: []HostRoute{route},
				}
				reqName, tid, devRoutesMap, err := getRoutesFromL3Config(req, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(tid).To(Equal(tableID))
				Expect(devRoutesMap).To(HaveLen(1))
				for _, routes := range devRoutesMap {
					Expect(routes).To(HaveLen(1))
					Expect(routes[0].Dst).NotTo(BeNil())
					Expect(routes[0].Dst.String()).To(BeEquivalentTo(dst))
				}
				Expect(reqName).To(Equal(testTableName))
			})
		})
	})

	Context("API", func() {
		DescribeTable("ApplyL3Config/DeleteL3Config", Ordered, func(applyReq, deleteReq *http.Request,
			expectedAppliedSuccess, expectedDeleteSuccess bool) {
			if applyReq != nil {
				response := ApplyL3Config(applyReq)
				Expect(response.Success).To(Equal(expectedAppliedSuccess))
			}
			if deleteReq != nil {
				response := DeleteL3Config(deleteReq)
				Expect(response.Success).To(Equal(expectedDeleteSuccess))
			}
		},
			Entry("valid request",
				httpL3Request("valid_req", "192.168.0.0/16", "192.168.0.0/24", false),
				httpL3Request("valid_req", "192.168.0.0/16", "192.168.0.0/24", false),
				true, true,
			),
			Entry("not-existing delete request",
				nil,
				httpL3Request("valid_req", "192.168.0.0/16", "192.168.0.0/24", false),
				false, false,
			),
		)

		DescribeTable("Add/DeleteRoute", func(addReq, deleteReq *http.Request,
			expectedAddSuccess, expectedDeleteSuccess bool) {
			if addReq != nil {
				response := AddRoute(addReq)
				Expect(response.Success).To(Equal(expectedAddSuccess))
			}
			if deleteReq != nil {
				response := DeleteRoute(deleteReq)
				Expect(response.Success).To(Equal(expectedDeleteSuccess))
			}
		},
			Entry("valid request",
				httpRouteRequest("192.168.1.0/16", "192.168.1.0/24"),
				httpRouteRequest("192.168.1.0/16", "192.168.1.0/24"),
				true, true,
			),
			Entry("invalid request",
				nil,
				httpRouteRequest("192.168.1.0/16", "192.168.1.0/24"),
				false, false,
			),
		)
	})

})

func httpL3Request(netName, subnet, dst string, force bool) *http.Request {
	route := HostRoute{
		Subnet:        dst,
		NextHop:       "0.0.0.0",
		InterfaceName: getValidIface(),
	}
	requestL3Config := L3ConfigRequest{
		Name:   netName,
		Subnet: subnet,
		Routes: []HostRoute{route},
		Force:  force,
	}
	l3config, err := json.Marshal(requestL3Config)
	Expect(err).NotTo(HaveOccurred())
	req, err := http.NewRequest("PUT", "", bytes.NewBuffer(l3config))
	Expect(err).NotTo(HaveOccurred())
	return req
}

func httpRouteRequest(subnet, dst string) *http.Request {
	route := HostRoute{
		Subnet:        dst,
		NextHop:       "0.0.0.0",
		InterfaceName: getValidIface(),
	}
	requestRoute, err := json.Marshal(route)
	Expect(err).NotTo(HaveOccurred())
	req, err := http.NewRequest("PUT", "", bytes.NewBuffer(requestRoute))
	Expect(err).NotTo(HaveOccurred())
	return req
}

func getValidIface() string {
	links, err := netlink.LinkList()
	Expect(err).NotTo(HaveOccurred())
	notFound := true
	for _, link := range links {
		devName := link.Attrs().Name
		if link.Type() == "device" {
			return devName
		}
	}
	Expect(notFound).To(BeFalse())
	return ""
}
