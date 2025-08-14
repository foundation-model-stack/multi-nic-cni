#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#
all: build

BASE_DIR=$(shell pwd)

##@ Test
TEST_BIN_DIR		= $(BASE_DIR)/tools/testbin
TEST_RESULT_DIR 	= $(BASE_DIR)/testing
ENVTEST				?= $(TEST_BIN_DIR)/setup-envtest
GINKGO				?= $(TEST_BIN_DIR)/ginkgo
ENVTEST_K8S_VERSION ?= 1.28.3

$(TEST_BIN_DIR):
	mkdir -p $(TEST_BIN_DIR)

$(TEST_RESULT_DIR):
	mkdir -p $(TEST_RESULT_DIR)

# go-get-tool will 'go get' any package $3 and install it to $2 in folder $1.
define go-get-tool
@[ -f $(2) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(3)" ;\
GOBIN=$(1) go install $(3) ;\
ls $$TMP_DIR;\
echo $(2);\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download and install ginkgo locally if necessary.
$(GINKGO): $(TEST_BIN_DIR)
	$(call go-get-tool,$(TEST_BIN_DIR),$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.21.0)

.PHONY: envtest
envtest: $(ENVTEST) ## Download and install setup-envtest locally if necessary.
$(ENVTEST): $(TEST_BIN_DIR)
	$(call go-get-tool,$(TEST_BIN_DIR),$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240624150636-162a113134de)

.PHONY: test
test: $(TEST_RESULT_DIR) ginkgo envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(TEST_BIN_DIR)/ginkgo run --cover --coverprofile=multi-nicd-coverage --json-report multi-nicd-unittest-report.json ./...
	@../hack/json-report-to-markdown.sh multi-nicd-unittest-report "Unit Test"
	@rm multi-nicd-unittest-report.json
	@../hack/coverage-to-markdown.sh multi-nicd-coverage "Multi-NIC Daemon Unit Test Coverage"

build: test
	@go build -o ../bin/daemon .

test-verbose: SHELL := /bin/bash
test-verbose: test-env ## Run tests with verbose option
	go test -v ./... -coverprofile cover.out
