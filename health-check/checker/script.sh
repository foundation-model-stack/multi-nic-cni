#!/bin/bash

if [ -z ${OPERATOR_NAMESPACE} ]; then
    OPERATOR_NAMESPACE=openshift-operators
fi

get_netname() {
    kubectl get multinicnetwork -ojson|jq .items| jq -r '.[].metadata.name'
}

apply() {
    export REPLACEMENT=$1
    export YAMLFILE=$2
    yq -e ${REPLACEMENT} ${YAMLFILE}.yaml|kubectl apply -f -
}

create_replacement() {
    export LOCATION=$1
    export REPLACE_VALUE=$2
    echo "(${LOCATION}=${REPLACE_VALUE})"
}

deploy() {
    kubectl apply -f ./checker/rbac.yaml
    NETWORK_NAME=$1
    if [ -z $1 ]; then
        NETWORK_NAME=$(get_netname)
    fi
    echo "Set network name ${NETWORK_NAME}"
    NETWORK_REPLACEMENT=$(create_replacement .spec.template.metadata.annotations.\"k8s.v1.cni.cncf.io/networks\" \"${NETWORK_NAME}\")
    apply ${NETWORK_REPLACEMENT} ./checker/deployment

}

"$@"