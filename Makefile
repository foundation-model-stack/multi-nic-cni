#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#
include daemon/Makefile

DOCKER ?= $(shell command -v podman 2> /dev/null || echo docker)

export IMAGE_REGISTRY ?= ghcr.io/foundation-model-stack

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
# VERSION ?= 0.0.1
VERSION ?= 1.3.1
export CHANNELS = "alpha"
export DEFAULT_CHANNEL = "alpha"

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# multinic.fms.io/multi-nic-cni-bundle:$VERSION and multinic.fms.io/multi-nic-cni/catalog:$VERSION.
IMAGE_TAG_BASE = $(IMAGE_REGISTRY)/multi-nic-cni

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
export IMG = $(IMAGE_TAG_BASE)-controller:v$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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

BASE_DIR=$(shell pwd)

##@ Development
DEV_BIN_DIR		= $(BASE_DIR)/tools/devbin
GOLANGCI_LINT	?= $(DEV_BIN_DIR)/golangci-lint
CONTROLLER_GEN 	?= $(DEV_BIN_DIR)/controller-gen
KUSTOMIZE 		?= $(DEV_BIN_DIR)/kustomize

$(DEV_BIN_DIR):
	mkdir -p $(DEV_BIN_DIR)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(DEV_BIN_DIR)
	$(call go-get-tool,$(DEV_BIN_DIR),$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.1)

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(DEV_BIN_DIR) ## Download kustomize locally if necessary.
	$(call go-get-tool,$(DEV_BIN_DIR),$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

GOLANGCI_LINT_INSTALL_SCRIPT ?= 'https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh'
.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ### Download golangci-lint locally if necessary.
$(GOLANGCI_LINT):$(DEV_BIN_DIR)
	test -s $(GOLANGCI_LINT) || { curl -sSfL $(GOLANGCI_LINT_INSTALL_SCRIPT) | sh -s -- -b $(DEV_BIN_DIR)  v1.64.8; }

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./api/..." paths="./controllers/..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..." paths="./controllers/..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

tidy:
	go mod tidy

.PHONY: fix
fix: tidy fmt vet golangci-lint ## Fixup files in the repo.
	$(GOLANGCI_LINT) run --fix

.PHONY: lint
lint: tidy fmt vet golangci-lint ## Run the lint check
	$(GOLANGCI_LINT) run

.PHONY: pr
pr: lint test yaml ## Run targets required for PR

##@ Test
ARCH                = $(shell go env GOARCH)
TEST_BIN_DIR		= $(BASE_DIR)/tools/testbin
TEST_RESULT_DIR 	= $(BASE_DIR)/testing
ENVTEST				?= $(TEST_BIN_DIR)/setup-envtest
GINKGO				?= $(TEST_BIN_DIR)/ginkgo
ENVTEST_K8S_VERSION ?= 1.28.3

$(TEST_BIN_DIR):
	mkdir -p $(TEST_BIN_DIR)

$(TEST_RESULT_DIR):
	mkdir -p $(TEST_RESULT_DIR)

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download and install ginkgo locally if necessary.
$(GINKGO): $(TEST_BIN_DIR)
	$(call go-get-tool,$(TEST_BIN_DIR),$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.21.0)

.PHONY: envtest
envtest: $(ENVTEST) ## Download and install setup-envtest locally if necessary.
$(ENVTEST): $(TEST_BIN_DIR)
	$(call go-get-tool,$(TEST_BIN_DIR),$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240624150636-162a113134de)

.PHONY: test
test: $(TEST_RESULT_DIR) fmt vet ginkgo manifests generate envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(TEST_BIN_DIR)/ginkgo run --cover --coverprofile=coverage --json-report unittest-report.json   ./controllers/...  ./internal/...
	@./hack/json-report-to-markdown.sh unittest-report "Unit Test"
	@rm unittest-report.json
	@./hack/coverage-to-markdown.sh coverage "Unit Test Coverage"

.PHONY: test-sanity
test-sanity: fmt vet generate fix ## Test repo formatting, linting, etc.
	git diff --exit-code # fast-fail if generate or fix produced changes
	make setup-lint
	make lint
	git diff --exit-code # diff again to ensure other checks don't change repo

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	$(DOCKER) build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	$(DOCKER) push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

clean-resource:
	@cd ./live-migration && chmod +x live_migrate.sh && ./live_migrate.sh _clean_resource

predeploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd config/samples && $(KUSTOMIZE) edit set image multi-nic-cni-daemon=${DAEMON_IMG}

deploy: predeploy ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=true -f -

yaml: predeploy ## Dry-run controller to the K8s cluster specified in ~/.kube/config. to deploy/simple.yaml
	$(KUSTOMIZE) build config/default > deploy/simple-deploy.yaml

operator-secret: ## Modify kustomization files for image pull secret of operator
	@[ "${OPERATOR_SECRET_NAME}" ] || ( echo "OPERATOR_SECRET_NAME is not set, run 'export OPERATOR_SECRET_NAME=$$(cat config/secret/operator-secret.yaml|yq .metadata.name)'"; exit 1 )
	cd config/secret;$(KUSTOMIZE) edit add resource operator-secret.yaml
	envsubst < config/manager/patches/image_pull_secret.template > config/manager/patches/image_pull_secret.yaml
	cd config/manager;$(KUSTOMIZE) edit add patch --path patches/image_pull_secret.yaml

daemon-secret: ## Modify kustomization files for image pull secret of daemon
	@[ "${DAEMON_SECRET_NAME}" ] || ( echo "DAEMON_SECRET_NAME is not set, run 'export DAEMON_SECRET_NAME=$$(cat config/secret/daemon-secret.yaml|yq .metadata.name)'"; exit 1 )
	cd config/secret;$(KUSTOMIZE) edit add resource daemon-secret.yaml
	envsubst < config/samples/patches/image_pull_secret.template > config/samples/patches/image_pull_secret.yaml
	cd config/samples;$(KUSTOMIZE) edit add patch --path patches/image_pull_secret.yaml

concheck:
	@kubectl create -f connection-check/concheck.yaml
	@echo "Wait for job/multi-nic-concheck to complete"
	@kubectl wait --for=condition=complete job/multi-nic-concheck --timeout=3000s
	@kubectl logs job/multi-nic-concheck

clean-concheck:
	@kubectl delete -f connection-check/concheck.yaml
	@kubectl delete pod -n default --selector multi-nic-concheck
	@kubectl delete job -n default --selector multi-nic-concheck

sample-concheck:
	@cd ./live-migration && chmod +x live_migrate.sh && ./live_migrate.sh live_iperf3 ${SERVER_HOST_NAME} ${CLIENT_HOST_NAME} 5

.PHONY: bundle
bundle: manifests kustomize predeploy ## Generate bundle manifests and metadata, then validate generated files.
	rm -f config/manifests/bases/multi-nic-cni-operator.clusterserviceversion.yaml
	envsubst < config/manifests/bases/multi-nic-cni-operator.clusterserviceversion.template > config/manifests/bases/multi-nic-cni-operator.clusterserviceversion.yaml
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(DOCKER) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.51.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)/catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool $(DOCKER) --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

test-daemon:
	$(DOCKER) build --platform linux/$(ARCH) -t daemon-test:latest -f ./daemon/dockerfiles/Dockerfile.multi-nicd-test .
	$(DOCKER) run --platform linux/$(ARCH) -i --rm --privileged daemon-test /bin/bash -c "cd /usr/local/build/cni&&make test"
	$(DOCKER) run --platform linux/$(ARCH) -i --rm -v ./testing:/usr/local/build/daemon/src/testing --privileged daemon-test /bin/bash -c "cd /usr/local/build/daemon/src&&make test"

build-push-kbuilder-base:
	$(DOCKER) build -t $(IMAGE_TAG_BASE)-kbuilder:v$(VERSION) -f ./daemon/dockerfiles/Dockerfile.kbuilder .
	$(DOCKER) push $(IMAGE_TAG_BASE)-kbuilder:v$(VERSION)

daemon-build: test-daemon ## Build docker image with the manager.
	$(DOCKER) tag daemon-test:latest $(IMAGE_TAG_BASE)-daemon:v$(VERSION)

daemon-push:
	$(DOCKER) push $(IMAGE_TAG_BASE)-daemon:v$(VERSION)

##@ Release

# Determine correct 'sed' version to use based on OS
ifeq ($(shell uname), Darwin)
  # macOS: use gsed if available
  ifeq ($(shell which gsed),)
    $(error gsed not found. Install with 'brew install gnu-sed')
  endif
  SED_CMD := gsed
else
  SED_CMD := sed
endif

# update the version in Makefile, kustomization.yaml, config.yaml, and GitHub workflows
# use VERSION as an arg to the set_version target: make set_version VERSION=x.y.z
.PHONY: set_version
set_version:
	@echo "VERSION: $(VERSION)"
	@$(SED_CMD) -i 's/^\(VERSION ?= \).*/\1$(VERSION)/' Makefile
	@$(SED_CMD) -i 's/\(newTag: v\).*/\1$(VERSION)/' config/manager/kustomization.yaml
	@$(SED_CMD) -i 's/\(newTag: v\).*/\1$(VERSION)/' config/samples/kustomization.yaml
	@$(SED_CMD) -i 's/\(image: ghcr.io\/foundation-model-stack\/multi-nic-cni-daemon:v\).*/\1$(VERSION)/' config/samples/config.yaml
	@$(SED_CMD) -i 's/\(IMAGE_VERSION: \).*/\1\"$(VERSION)\"/' .github/workflows/*.yaml
	@$(SED_CMD) -i 's/\(VERSION: \).*/\1\"$(VERSION)\"/' .github/workflows/build_push_controller.yaml
	@$(SED_CMD) -i 's/\(VERSION: \)[0-9.]\+\(-pr-.*\)/\1$(VERSION)\2/' .github/workflows/build_push_pr.yaml
	@$(SED_CMD) -i 's/multi-nic-cni-bundle:v[0-9.]\+/multi-nic-cni-bundle:v$(VERSION)/' README.md
	@$(SED_CMD) -i 's/multi-nic-cni-concheck:v[0-9.]\+/multi-nic-cni-concheck:v$(VERSION)/' connection-check/concheck.yaml
	@$(SED_CMD) -i 's/multi-nic-cni-daemon:v[0-9.]\+/multi-nic-cni-daemon:v$(VERSION)/' internal/vars/vars.go
	@$(SED_CMD) -i 's/-daemon:v[0-9.]\+/-daemon:v$(VERSION)/' daemon/Makefile
	@$(SED_CMD) -i 's/\(FROM ghcr.io.*multi-nic-cni-kbuilder:v\)[0-9.]\+/\1$(VERSION)/' daemon/dockerfiles/Dockerfile.multi-nicd-test
