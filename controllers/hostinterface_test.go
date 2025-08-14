/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers_test

import (
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Host Interface Test", func() {
	controllers.ConfigReady = true

	Context("UpdateNewInterfaces - original with a single device", func() {
		origInfos := []multinicv1.InterfaceInfoType{
			genInterfaceInfo("eth1", "10.0.0.0/24"),
		}
		It("can detect change", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth1", "10.0.1.0/24"),
			}
			newInfos, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeTrue())
			Expect(len(newInfos)).To(Equal(1))
			Expect(newInfos[0].InterfaceName).To(Equal("eth1"))
			Expect(newInfos[0].NetAddress).To(Equal("10.0.1.0/24"))
		})
		It("can check no change", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth1", "10.0.0.0/24"),
			}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can leave old one", func() {
			newInfos := []multinicv1.InterfaceInfoType{}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can add new while leave old one", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth2", "10.0.1.0/24"),
			}
			newInfos, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeTrue())
			Expect(len(newInfos)).To(Equal(2))
			for _, newInfo := range newInfos {
				Expect(newInfo.InterfaceName).To(BeElementOf("eth1", "eth2"))
				Expect(newInfo.NetAddress).To(BeElementOf("10.0.0.0/24", "10.0.1.0/24"))
			}
		})
	})
	Context("UpdateNewInterfaces - original with more than one devices", func() {
		origInfos := []multinicv1.InterfaceInfoType{
			genInterfaceInfo("eth1", "10.0.0.0/24"),
			genInterfaceInfo("eth2", "10.0.1.0/24"),
		}
		It("can leave old one", func() {
			newInfos := []multinicv1.InterfaceInfoType{}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can check no change", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth1", "10.0.0.0/24"),
				genInterfaceInfo("eth2", "10.0.1.0/24"),
			}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can check no change in swop order", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth2", "10.0.1.0/24"),
				genInterfaceInfo("eth1", "10.0.0.0/24"),
			}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can leave old one when some is missing", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth1", "10.0.0.0/24"),
			}
			_, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeFalse())
		})
		It("can leave old one when some is missing and some with new info", func() {
			newInfos := []multinicv1.InterfaceInfoType{
				genInterfaceInfo("eth1", "10.0.2.0/24"),
			}
			newInfos, updated := controllers.UpdateNewInterfaces(origInfos, newInfos)
			Expect(updated).To(BeTrue())
			Expect(len(newInfos)).To(Equal(2))
			for _, newInfo := range newInfos {
				Expect(newInfo.InterfaceName).To(BeElementOf("eth1", "eth2"))
				Expect(newInfo.NetAddress).To(BeElementOf("10.0.2.0/24", "10.0.1.0/24"))
			}
		})
	})

})

func genInterfaceInfo(devName, netAddress string) multinicv1.InterfaceInfoType {
	return multinicv1.InterfaceInfoType{
		InterfaceName: devName,
		NetAddress:    netAddress,
	}
}
