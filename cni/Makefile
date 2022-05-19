#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#
all: build copy-to-daemon

ENVTEST_ASSETS_DIR=$(PWD)/test-bin
export PATH := $(PATH):$(PWD)/test-bin
export GOPATH := $(HOME)/go

get-ginkgo:
	@go get -u github.com/onsi/ginkgo/ginkgo && go install github.com/onsi/ginkgo/ginkgo && \
	 go get -u github.com/onsi/gomega/... && go install github.com/onsi/gomega/... && \
	 mkdir -p ${ENVTEST_ASSETS_DIR} && \
	 cp $(GOPATH)/bin/ginkgo $(ENVTEST_ASSETS_DIR)/ginkgo

ginkgo-set:
	@test -f $(ENVTEST_ASSETS_DIR)/ginkgo || ($(MAKE) get-ginkgo)

get-plugins:
	@cd /tmp && \
	 git clone https://github.com/containernetworking/plugins.git && \
	 cd plugins && \
	 ./build_linux.sh && \
	ls /tmp/plugins/bin && \
	mkdir -p $(ENVTEST_ASSETS_DIR) && \
	cp /tmp/plugins/bin/ipvlan $(ENVTEST_ASSETS_DIR)/ipvlan && \
	cp /tmp/plugins/bin/static $(ENVTEST_ASSETS_DIR)/static
	@rm -r /tmp/plugins

plugins-set:
	@test -f $(ENVTEST_ASSETS_DIR)/ipvlan || ($(MAKE) get-plugins)
	@test -f $(ENVTEST_ASSETS_DIR)/static || ($(MAKE) get-plugins)

tidy:
	@go mod tidy

test: ginkgo-set plugins-set
	@cd ./plugins/main/multi-nic && ginkgo

build: ginkgo-set plugins-set tidy
	@mkdir -p bin
	@go build -o bin/multi-nic ./plugins/main/multi-nic
	@go build -o bin/multi-nic-ipam ./plugins/ipam/multi-nic-ipam

build-with-test: test build

copy-to-daemon:
	@rm -r ../daemon/cni
	@mkdir -p ../daemon/cni
	@cp -r ./pkg ../daemon/cni/pkg
	@cp -r ./plugins ../daemon/cni/plugins
	@cp -r ./go.mod ../daemon/cni/go.mod