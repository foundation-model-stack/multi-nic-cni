#!/bin/bash

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
    itemlist=$(kubectl get $cr -ojson |jq '.items'| jq 'del(.[].metadata.resourceVersion,.[].metadata.uid,.[].metadata.selfLink,.[].metadata.creationTimestamp,.[].metadata.generation,.[].metadata.ownerReferences)') 
    echo {"apiVersion": "v1", "items": $itemlist, "kind": "List"}| yq eval - -P > $dir/$cr.yaml
}

status_cr="cidr.multinic ippool.multinic hostinterface.multinic"
activate_cr="multinicnetwork"

get_netname() {
    kubectl get multinicnetwork -ojson|jq .items| jq '.[].metadata.name'| tr -d '"'
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
    kubectl delete subscriptions.operators.coreos.com multi-nic-cni-operator -n openshift-operators
    kubectl delete clusterserviceversion multi-nic-cni-operator.v1.0.2 -n openshift-operators
    kubectl delete ds multi-nicd -n openshift-operators
}

# after reinstall operator
# deactivate_route_config

deploy_status_cr() {
    snapshot_dir="snapshot/$1"
    for cr in $status_cr
    do
        kubectl apply -f $snapshot_dir/$cr.yaml
    done
}

activate_route_config() {
    snapshot_dir="snapshot/$1"
    kubectl apply -f $snapshot_dir/$activate_cr.yaml
    sleep 5
    kubectl get net-attach-def $(get_netname) -ojson|jq '.spec.config'
}

"$@"


