#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#
export DAEMON_REGISTRY = ghcr.io/foundation-model-stack

# DAEMON_IMG defines the image:tag used for daemon
DAEMON_TAG_BASE = multi-nic-cni
DAEMON_VERSION ?= v1.0.2
DAEMON_IMG ?= $(DAEMON_REGISTRY)/$(DAEMON_TAG_BASE)-daemon:$(DAEMON_VERSION)


test-verbose:
	$(MAKE) -C src test-verbose


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
	docker build -t ${DAEMON_IMG} .
else
	@echo "Cannot build daemon on $(shell uname)"
	exit 1
endif

docker-build-push: update-cni-local image-build
	docker push ${DAEMON_IMG}