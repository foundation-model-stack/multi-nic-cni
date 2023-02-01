#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#
all: build

ENVTEST_ASSETS_DIR=$(PWD)/test-bin
export PATH := $(PATH):$(ENVTEST_ASSETS_DIR)

test: SHELL := /bin/bash
test:  ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	@test -f ${ENVTEST_ASSETS_DIR}/kubebuilder/bin/etcd || (curl -sSLo ${ENVTEST_ASSETS_DIR}/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz && tar -zxvf  ${ENVTEST_ASSETS_DIR}/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz && rm -rf ${ENVTEST_ASSETS_DIR}/kubebuilder/kubebuilder_2.0.0-alpha.1_linux_amd64 || true && mv kubebuilder_2.0.0-alpha.1_linux_amd64 ${ENVTEST_ASSETS_DIR}/kubebuilder)
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

build: test
	@go build -o ../bin/daemon .

test-verbose: SHELL := /bin/bash
test-verbose:  ## Run tests with verbose option
	mkdir -p ${ENVTEST_ASSETS_DIR}
	@test -f ${ENVTEST_ASSETS_DIR}/kubebuilder/bin/etcd || (curl -sSLo ${ENVTEST_ASSETS_DIR}/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.0.0-alpha.1/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz && tar -zxvf  ${ENVTEST_ASSETS_DIR}/kubebuilder_2.0.0-alpha.1_linux_amd64.tar.gz && rm -rf ${ENVTEST_ASSETS_DIR}/kubebuilder/kubebuilder_2.0.0-alpha.1_linux_amd64 || true && mv kubebuilder_2.0.0-alpha.1_linux_amd64 ${ENVTEST_ASSETS_DIR}/kubebuilder)
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ./... -coverprofile cover.out
