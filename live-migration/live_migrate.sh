#!/bin/bash

if [ -z ${OPERATOR_NAMESPACE} ]; then
    OPERATOR_NAMESPACE=openshift-operators
fi

_snapshot_resource() {
    dir=$1
    mkdir -p $dir
    kind=$2
    item=$3
    kubectl get $kind $item -ojson | jq 'del(.metadata.resourceVersion,.metadata.uid,.metadata.selfLink,.metadata.creationTimestamp,.metadata.generation,.metadata.ownerReferences)' | yq eval - -P > $dir/$kind-$item.yaml
    echo "snapshot $dir/$kind-$item.yaml"

}

_snapshot() {
    dir=$1
    cr=$2
    itemlist=$(kubectl get $cr -ojson |jq '.items'| jq 'del(.[].status,.[].metadata.resourceVersion,.[].metadata.uid,.[].metadata.selfLink,.[].metadata.creationTimestamp,.[].metadata.generation,.[].metadata.ownerReferences)') 
    echo {"apiVersion": "v1", "items": $itemlist, "kind": "List"}| yq eval - -P > $dir/$cr.yaml
}

status_cr="cidr.multinic ippool.multinic hostinterface.multinic"
activate_cr="multinicnetwork"

get_netname() {
    kubectl get multinicnetwork -ojson|jq .items| jq '.[].metadata.name'| tr -d '"'
}

get_controller() {
    kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nic-cni-operator-controller-manager|awk '{print $1}'
}

get_controller_log() {
    controller=$(get_controller)
    kubectl logs $controller -n ${OPERATOR_NAMESPACE} -c manager
}

snapshot() {
    mkdir -p snapshot
    # update l2 file used to stop controller to modify the route
    cp multinicnetwork_l2.yaml snapshot/multinicnetwork_l2.yaml
    netname=$(get_netname)
    yq -e -i .metadata.name=\"$netname\" snapshot/multinicnetwork_l2.yaml
    echo "rename multinicnetwork_l2.yaml with $netname"
    # snapshot state
    snapshot_dir="snapshot/$1"
    mkdir -p $snapshot_dir
    for cr in $status_cr $activate_cr
    do
        _snapshot $snapshot_dir $cr
    done
    ls $snapshot_dir
    echo "saved in $snapshot_dir"
}

deactivate_route_config() {
    kubectl apply -f snapshot/multinicnetwork_l2.yaml
    sleep 5
    kubectl get net-attach-def $(get_netname) -ojson|jq '.spec.config'
}

uninstall_operator() {
    # uninstall operator
    kubectl delete subscriptions.operators.coreos.com multi-nic-cni-operator -n $OPERATOR_NAMESPACE
    kubectl delete clusterserviceversion multi-nic-cni-operator.v1.0.2 -n $OPERATOR_NAMESPACE
    kubectl delete ds multi-nicd -n $OPERATOR_NAMESPACE
}

# after reinstall operator
# deactivate_route_config

patch_daemon() {
    kubectl patch config.multinic multi-nicd --type merge --patch '{"spec": {"daemon": {"imagePullPolicy": "Always"}}}'
    kubectl delete po -l app=multi-nicd -n ${OPERATOR_NAMESPACE}
}

wait_daemon() {
    echo "Wait for daemonset to be ready"
    kubectl rollout status daemonset multi-nicd -n ${OPERATOR_NAMESPACE} --timeout 300s
}

deploy_status_cr() {
    snapshot_dir="snapshot/$1"
    for cr in $status_cr
    do
        kubectl apply -f $snapshot_dir/$cr.yaml
    done
}

restart_controller() {
    controller=$(get_controller)
    kubectl delete po $controller -n ${OPERATOR_NAMESPACE}
    echo "Wait for deployment to be available"
    kubectl wait deployment -n ${OPERATOR_NAMESPACE} multi-nic-cni-operator-controller-manager --for condition=Available=True --timeout=90s
    ready=$(echo $(get_controller_log)|grep ConfigReady)
    while [ -z "$ready" ];
    do
        sleep 5
        echo "Wait for config to be ready..."
        ready=$(echo $(get_controller_log)|grep ConfigReady)
    done
    echo "Config Ready"
}

activate_route_config() {
    snapshot_dir="snapshot/$1"
    kubectl apply -f $snapshot_dir/$activate_cr.yaml
    sleep 5
    kubectl get net-attach-def $(get_netname) -ojson|jq '.spec.config'
}

"$@"


