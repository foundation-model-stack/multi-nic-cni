#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#

export IMAGE_REGISTRY ?= res-cpe-team-docker-local.artifactory.swg-devops.com

IMAGE_TAG_BASE ?= net/multi-nic-cni
CONCHECK_VERSION ?= v1.0.0-alpha

CONCHECK_IMG ?= $(IMAGE_REGISTRY)/$(IMAGE_TAG_BASE)-concheck:$(CONCHECK_VERSION)


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
