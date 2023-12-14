#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#

export IMAGE_REGISTRY ?= ghcr.io/foundation-model-stack
export OPERATOR_NAMESPACE ?= multi-nic-cni-operator

IMAGE_TAG_BASE = $(IMAGE_REGISTRY)/multi-nic-cni
IMAGE_VERSION ?= 1.0.5
SIDECAR_IMG ?= $(IMAGE_TAG_BASE)-health-check-sidecar:v$(IMAGE_VERSION)
CHECKER_IMG ?= $(IMAGE_TAG_BASE)-health-checker:v$(IMAGE_VERSION)

build-sidecar:
	go mod tidy && go build -o sidecar
	
build-checker:
	go mod tidy && go build -o checker

docker-build-sidecar:
	docker build -t ${SIDECAR_IMG} -f ./sidecar/Dockerfile ./sidecar

docker-build-checker:
	docker build -t ${CHECKER_IMG} -f ./checker/Dockerfile ./checker

docker-push:
	docker push ${SIDECAR_IMG}
	docker push ${CHECKER_IMG}

test-sidecar: docker-build-sidecar
	@docker run -i --privileged ${SIDECAR_IMG} /bin/bash -c "make test"

exec-sidecar:
	@docker run -it --privileged ${SIDECAR_IMG} /bin/bash

test-checker: docker-build-checker
	@docker run -i --privileged ${CHECKER_IMG} /bin/bash -c "make test"

patch-sidecar:
	@kubectl patch daemonset multi-nicd --type merge -patch-file ./sidecar/patch.yaml -n ${OPERATOR_NAMESPACE}

deploy-agent:
	@kubectl apply -f ./sidecar/manifest.yaml

deploy: deploy-agent
	@./checker/script.sh deploy

clean:
	@kubectl delete -f ./checker/deployment.yaml  && echo "Completely delete checkers" || echo "Failed to clean checkers"  
	@kubectl delete -f ./checker/configmap.yaml && echo "Completely delete configmap of checkers" || echo "Failed to clean configmap of checkers"  
	@kubectl delete -f ./checker/rbac.yaml && echo "Completely delete RBAC resource of checkers" || echo "Failed to clean RBAC resource of checkers"  
	@kubectl delete -f ./sidecar/manifest.yaml && echo "Completely delete agents" || echo "Failed to clean agents"  