#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache-2.0
#
export DAEMON_REGISTRY ?= ghcr.io/foundation-model-stack

# DAEMON_IMG defines the image:tag used for daemon
IMAGE_TAG_BASE = $(DAEMON_REGISTRY)/multi-nic-cni
DAEMON_IMG ?= $(IMAGE_TAG_BASE)-daemon:v1.3.1


test-verbose:
	$(MAKE) -C src test-verbose

test:
	$(MAKE) -C src test

update-cni-local:
ifeq ($(shell uname), Linux)
	$(MAKE) -C ../cni all
else
	@echo "Cannot build daemon on $(shell uname)"
	exit 1
endif

image-build:
ifeq ($(shell uname), Linux)
	@echo "Build deamon component"
	docker build -t ${DAEMON_IMG} -f ./dockerfiles/Dockerfile .
else
	@echo "Cannot build daemon on $(shell uname)"
	exit 1
endif

docker-build-push: update-cni-local image-build
	docker push ${DAEMON_IMG}
