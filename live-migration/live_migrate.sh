#!/bin/bash

if [ -z ${OPERATOR_NAMESPACE} ]; then
    OPERATOR_NAMESPACE=multi-nic-cni-operator
fi

if [ -z ${CLUSTER_NAME} ]; then
    CLUSTER_NAME="default"
fi

#############################################
# utility functions

get_netname() {
    kubectl get multinicnetwork -ojson|jq .items| jq '.[].metadata.name'| tr -d '"' | head -n 1
}

get_controller() {
    kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nic-cni-operator-controller-manager|awk '{print $1}'
}

get_controller_log() {
    controller=$(get_controller)
    kubectl logs $controller -n ${OPERATOR_NAMESPACE} -c manager
}

get_status() {
    kubectl get multinicnetwork -o custom-columns=NAME:.metadata.name,ConfigStatus:.status.configStatus,RouteStatus:.status.routeStatus,TotalHost:.status.discovery.existDaemon,HostWithSecondaryNIC:.status.discovery.infoAvailable,ProcessedHost:.status.discovery.cidrProcessed,Time:.status.lastSyncTime -w
}

get_secondary_ip() {
   PODNAME=$1
   kubectl get po $PODNAME -ojson|jq .metadata.annotations|jq '.["k8s.v1.cni.cncf.io/network-status"]'| jq -r |jq .[1].ips[0]
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

#############################################

#############################################
# cr handling

status_cr="cidrs.multinic ippools.multinic hostinterfaces.multinic"
activate_cr="multinicnetworks.multinic"
config_cr="configs.multinic deviceclasses.multinic"

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
    itemlist=$(kubectl get $cr -ojson |jq '.items'| jq 'del(.[].status,.[].metadata.finalizers,.[].metadata.resourceVersion,.[].metadata.uid,.[].metadata.selfLink,.[].metadata.creationTimestamp,.[].metadata.generation,.[].metadata.ownerReferences)') 
    echo {"apiVersion": "v1", "items": $itemlist, "kind": "List"}| yq eval - -P > $dir/$cr.yaml
}

snapshot() {
    mkdir -p snapshot
    # update l2 file used to stop controller to modify the route
    cp multinicnetwork_l2.yaml snapshot/multinicnetwork_l2.yaml
    netname=$(get_netname)
    yq -e -i .metadata.name=\"$netname\" snapshot/multinicnetwork_l2.yaml
    echo "rename multinicnetwork_l2.yaml with $netname"
    # snapshot state
    snapshot_dir="snapshot/${CLUSTER_NAME}"
    mkdir -p $snapshot_dir
    for cr in $status_cr $activate_cr
    do
        _snapshot $snapshot_dir $cr
    done
    ls $snapshot_dir
    echo "saved in $snapshot_dir"
}

deploy_status_cr() {
    snapshot_dir="snapshot/${CLUSTER_NAME}"
    for cr in $status_cr
    do
        kubectl apply -f $snapshot_dir/$cr.yaml
    done
}

#############################################

#############################################
# route handling

deactivate_route_config() {
    kubectl apply -f snapshot/multinicnetwork_l2.yaml
    sleep 5
    configSTR=$(kubectl get multinicnetwork $(get_netname) -ojson|jq '.spec.multiNICIPAM')
    if [[ "$configSTR" == "false" ]]; then
        echo "Deactivate route configuration."
    fi
}

activate_route_config() {
    snapshot_dir="snapshot/${CLUSTER_NAME}"
    kubectl apply -f $snapshot_dir/$activate_cr.yaml
    sleep 5
    configSTR=$(kubectl get multinicnetwork $(get_netname) -ojson|jq '.spec.multiNICIPAM')
    if [[ "$configSTR" == "true" ]]; then
        echo "Activate route configuration."
    fi
}

#############################################

#############################################
# operator resource handling: controller, daemon, crd

_clean_resource() {
    for cr in $status_cr $activate_cr $config_cr
    do
        kubectl delete $cr --all
    done
    wait_daemon_terminated
}

clean_resource() {
    deactivate_route_config
    _clean_resource
}

wait_daemon_terminated() {
    kubectl wait --for=delete daemonset/multi-nicd -n ${OPERATOR_NAMESPACE} --timeout=300s
    # wait for all terminated
    daemonTerminated=$(kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nicd|wc -l|tr -d ' ')
    while [ "$daemonTerminated" != 0 ] ; 
    do
        echo "Wait for daemonset to be fully terminated...($daemonTerminated left)"
        sleep 10
        daemonTerminated=$(kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nicd|wc -l|tr -d ' ')
    done
    echo "Done"
}

uninstall_operator() {
    version=$1
    # uninstall operator
    kubectl delete subscriptions.operators.coreos.com multi-nic-cni-operator -n $OPERATOR_NAMESPACE
    kubectl delete clusterserviceversion multi-nic-cni-operator.v${version} -n $OPERATOR_NAMESPACE
    kubectl delete ds multi-nicd -n $OPERATOR_NAMESPACE
}

clean_crd() {
    for cr in $status_cr $activate_cr $config_cr
    do
        kubectl delete crd $cr.fms.io
    done
}

# after reinstall operator
# deactivate_route_config

patch_daemon() {
    kubectl patch config.multinic multi-nicd --type merge --patch '{"spec": {"daemon": {"imagePullPolicy": "Always"}}}'
    kubectl delete po -l app=multi-nicd -n ${OPERATOR_NAMESPACE}
}

wait_daemon() {
    # wait for daemon creation
    sleep 5
    daemonCreate=$(kubectl get ds multi-nicd -n ${OPERATOR_NAMESPACE}|wc -l|tr -d ' ')
    while [ "$daemonCreate" == 0 ] ; 
    do
        echo "Wait for daemonset to be created by controller..."
        sleep 2
        daemonCreate=$(kubectl get ds multi-nicd -n ${OPERATOR_NAMESPACE}|wc -l|tr -d ' ')
    done
    echo "Wait for daemonset to be ready"
    kubectl rollout status daemonset multi-nicd -n ${OPERATOR_NAMESPACE} --timeout 300s
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
        kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nic-cni-operator-controller-manager
        echo "Wait for config to be ready..."
        ready=$(echo $(get_controller_log)|grep ConfigReady)
    done
    echo "Config Ready"
}

#############################################

#############################################
# iperf live

# ./live_migrate.sh live_iperf3 <SERVER_HOST_NAME> <CLIENT_HOST_NAME> <LIVE_TIME>
# or
# ./live_migrate.sh live_iperf3 <LIVE_TIME>
live_iperf3() {
    # use first two pods if server and client hosts are not specified.
    if [ $# -eq 3 ]; then
        SERVER_HOST_NAME=$1
        CLIENT_HOST_NAME=$2
        LIVE_TIME=$3
    else
        SERVER_HOST_NAME=$(kubectl get nodes|tail -n 2|head -n 1|awk '{ print $1 }')
        CLIENT_HOST_NAME=$(kubectl get nodes|tail -n 1|awk '{ print $1 }')
        LIVE_TIME=$1
    fi

   NETWORK_NAME=$(get_netname)
   echo "Test connection of ${NETWORK_NAME} from ${CLIENT_HOST_NAME} to ${SERVER_HOST_NAME}"
   NETWORK_REPLACEMENT=$(create_replacement .metadata.annotations.\"k8s.v1.cni.cncf.io/networks\" \"${NETWORK_NAME}\")
   SERVER_HOSTNAME_REPLACEMENT=$(create_replacement .spec.nodeName \"${SERVER_HOST_NAME}\")
   CLIENT_HOSTNAME_REPLACEMENT=$(create_replacement .spec.nodeName \"${CLIENT_HOST_NAME}\")

   SERVER_NAME="multi-nic-iperf3-server"
   CLIENT_NAME="multi-nic-iperf3-client"

   SERVER_NAME_REPLACEMENT=$(create_replacement .metadata.name \"${SERVER_NAME}\")
   # deploy server pod
   apply ${SERVER_NAME_REPLACEMENT},${NETWORK_REPLACEMENT},${SERVER_HOSTNAME_REPLACEMENT} ./test/iperf3/server

   # wait until server available
   kubectl wait pod ${SERVER_NAME} --for condition=ready --timeout=90s

   SECONDARY_IP=$(get_secondary_ip ${SERVER_NAME}| tr -d '"')
   CLIENT_NAME_REPLACEMENT=$(create_replacement .metadata.name \"${CLIENT_NAME}\")
   # deploy client pod
   apply ${CLIENT_NAME_REPLACEMENT},${NETWORK_REPLACEMENT},${CLIENT_HOSTNAME_REPLACEMENT} ./test/iperf3/client

   if [[ "${SECONDARY_IP}" == "null" ]]; then
        echo >&2 "cannot get secondary IP of server ${SERVER_NAME}"
        exit 2
   fi
   # wait until client available
   kubectl wait pod ${CLIENT_NAME} --for condition=ready --timeout=90s
   # run live client
   kubectl exec -it ${CLIENT_NAME} -- iperf3 -c ${SECONDARY_IP} -t ${LIVE_TIME} -p 30000

   # clean up
   kubectl delete pod ${CLIENT_NAME} ${SERVER_NAME}
}

#############################################


"$@"


