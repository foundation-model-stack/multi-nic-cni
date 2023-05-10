#
# Copyright 2022- IBM Inc. All rights reserved
# SPDX-License-Identifier: Apache2.0
#
CLUSTER_NAME?="kind"
START=$(date +%s)

export DAEMON_STUB_IMG?="e2e-test/daemon-stub:latest"
export CNI_STUB_IMG?="e2e-test/cni-stub:latest"

##########################################
# Cluster Preparation
##########################################
# cluster
create-kind:
	kind create cluster --name ${CLUSTER_NAME} --config ./deploy/kind/${CLUSTER_NAME}.yaml
	./script.sh wait_node

delete-kind:
	kind delete cluster --name ${CLUSTER_NAME} 

# images
daemon-image-build:
	docker build -t ${DAEMON_STUB_IMG} -f ./daemon-stub/Dockerfile ./daemon-stub
.PHONY: daemon-image-build

cni-image-build:
	docker build -t ${CNI_STUB_IMG} -f ./cni-stub/Dockerfile ./cni-stub
.PHONY: cni-image-build

controller-build:
	docker build -t e2e-test/multi-nic-cni-controller:latest -f ../Dockerfile ..
.PHONY: controller-build

daemon-build-load: daemon-image-build
	kind load docker-image ${DAEMON_STUB_IMG} --name ${CLUSTER_NAME}
.PHONY: daemon-build-load

cni-build-load: cni-image-build
	kind load docker-image ${CNI_STUB_IMG} --name ${CLUSTER_NAME}
.PHONY: cni-build-load

controller-build-load: controller-build
	kind load docker-image e2e-test/multi-nic-cni-controller:latest --name ${CLUSTER_NAME}
.PHONY: controller-build-load

load-images: load-kwok
	kind load docker-image ${DAEMON_STUB_IMG} --name ${CLUSTER_NAME}
	kind load docker-image ${CNI_STUB_IMG} --name ${CLUSTER_NAME}
	kind load docker-image e2e-test/multi-nic-cni-controller:latest --name ${CLUSTER_NAME}
.PHONY: load-images

build-load-images: controller-build-load daemon-build-load cni-build-load
.PHONY: build-load-images

load-kwok:
	docker pull ghcr.io/sunya-ch/kwok:ab07cfa-dirty
	kind load docker-image ghcr.io/sunya-ch/kwok:ab07cfa-dirty --name ${CLUSTER_NAME}

# deployment
prepare-controller:
	./script.sh deploy_controller
	./script.sh deploy_kwok
	./script.sh deploy_netattachdef