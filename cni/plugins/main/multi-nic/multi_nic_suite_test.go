/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main_test

import (
	"testing"

	"github.com/containernetworking/plugins/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	logFilePath = "/var/log/multi-nic-cni.log"
)

func TestMultiNic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultiNic Suite")
}

var _ = BeforeSuite(func() {
	utils.InitializeLogger(logFilePath)
})
