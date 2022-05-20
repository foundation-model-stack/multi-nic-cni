/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMultiNic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultiNic Suite")
}
