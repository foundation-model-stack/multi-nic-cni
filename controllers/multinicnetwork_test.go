/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"

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

	multinicnetwork := getMultiNicCNINetwork(cniVersion, cniType, cniArgs)

	It("successfully create/delete network attachment definition", func() {
		mainPlugin, annotations, err := multinicnetworkReconciler.GetMainPluginConf(multinicnetwork, hifList)
		Expect(err).NotTo(HaveOccurred())
		err = multinicnetworkReconciler.NetAttachDefHandler.CreateOrUpdate(multinicnetwork, mainPlugin, annotations)
		Expect(err).NotTo(HaveOccurred())
		err = multinicnetworkReconciler.NetAttachDefHandler.DeleteNets(multinicnetwork)
		Expect(err).NotTo(HaveOccurred())
	})

	It("successfully deploy and delete", func() {
		Expect(k8sClient.Create(context.TODO(), multinicnetwork)).Should(Succeed())
		Expect(k8sClient.Delete(context.TODO(), multinicnetwork)).Should(Succeed())
	})

})
