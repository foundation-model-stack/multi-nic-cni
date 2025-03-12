#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#
all: build

ENVTEST_ASSETS_DIR=$(PWD)/test-bin
export PATH := $(PATH):$(ENVTEST_ASSETS_DIR)

test-env: SHELL := /bin/bash
test-env:
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); 
	cp -r ${ENVTEST_ASSETS_DIR}/bin /usr/local/kubebuilder/bin

test: SHELL := /bin/bash
test: test-env ## Run tests.
	go test ./... -coverprofile cover.out

build: test
	@go build -o ../bin/daemon .

test-verbose: SHELL := /bin/bash
test-verbose: test-env ## Run tests with verbose option
	go test -v ./... -coverprofile cover.out
