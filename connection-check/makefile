#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#

export IMAGE_REGISTRY ?= ghcr.io/foundation-model-stack

IMAGE_TAG_BASE = $(IMAGE_REGISTRY)/multi-nic-cni
IMAGE_VERSION ?= 1.0.0
CONCHECK_IMG ?= $(IMAGE_TAG_BASE)-concheck:v$(IMAGE_VERSION)


docker-build:
	docker build -t ${CONCHECK_IMG} .

docker-push:
	docker push ${CONCHECK_IMG}

build-push: docker-build
	docker push ${CONCHECK_IMG}

local-run:
	go build
	./concheck

clean:
	kubectl delete pod -n default --selector multi-nic-concheck
	kubectl delete job -n default --selector multi-nic-concheck
