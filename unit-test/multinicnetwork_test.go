/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("Test deploying MultiNicNetwork", func() {
	cniVersion := "0.3.0"
	cniType := "ipvlan"
	mode := "l2"
	mtu := 1500
	cniArgs := make(map[string]string)
	cniArgs["mode"] = mode
	cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

	multinicnetwork := getMultiNicCNINetwork("test-mn", cniVersion, cniType, cniArgs)

	It("successfully create/delete network attachment definition", func() {
		mainPlugin, annotations, err := multinicnetworkReconciler.GetMainPluginConf(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
		err = multinicnetworkReconciler.NetAttachDefHandler.CreateOrUpdate(multinicnetwork, mainPlugin, annotations)
		Expect(err).NotTo(HaveOccurred())
		err = multinicnetworkReconciler.NetAttachDefHandler.DeleteNets(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})
})
