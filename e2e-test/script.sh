#!/bin/bash
# yq 
OPERATOR_NAMESPACE=multi-nic-cni-operator-system

get_controller() {
    kubectl get po -n ${OPERATOR_NAMESPACE}|grep multi-nic-cni-operator-controller-manager|awk '{print $1}'
}

get_controller_log() {
    controller=$(get_controller)
    kubectl logs $controller -n ${OPERATOR_NAMESPACE} -c manager
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

deploy_controller() {
    kubectl apply -f deploy/controller/deployment.yaml
    kubectl wait deployment -n ${OPERATOR_NAMESPACE} multi-nic-cni-operator-controller-manager --for condition=Available=True --timeout=90s
    ready=$(echo $(get_controller_log)|grep ConfigReady)
    while [ -z "$ready" ];
    do
        sleep 5
        echo "Wait for config to be ready..."
        ready=$(echo $(get_controller_log)|grep ConfigReady)
    done
    echo "Config Ready"
    deploy_fake_ds
}

deploy_fake_ds() {
    kubectl delete ds multi-nicd -n ${OPERATOR_NAMESPACE}
    kubectl apply -f deploy/controller/config.yaml
}

deploy_kwok() {
    kubectl apply -f deploy/kwok/kwok.yaml
    kubectl wait deployment -n kube-system kwok-controller --for condition=Available=True --timeout=90s
}

deploy_netattachdef(){
    kubectl apply -f deploy/net-attach-def-crd.yaml
}

deploy_network(){
    hb=$1
    kubectl delete multinicnetwork --all
    kubectl apply -f deploy/multinicnetwork/${hb}-hb-net.yaml
}

delete_controller() {
    kubectl delete multinicnetwork --all
    kubectl wait --for=delete multinicnetwork/multi-nic-sample --timeout=60s
    kubectl delete config.multinic --all
    kubectl wait --for=delete config.multinic/multi-nicd --timeout=60s
    kubectl delete -f deploy/controller/deployment.yaml
}

wait_node() {
    for nodename in $(kubectl get nodes |awk '(NR>1){print $1}'); do
        kubectl wait node ${nodename} --for condition=Ready --timeout=1000s
    done
}

_deploy_node() {
    i=$1
    export podname=multi-nicd-stub-$i
    yq e '.metadata.name = env(podname)' deploy/template/daemon-stub-pod.tpl|kubectl apply -f - > /dev/null 2>&1
    sleep 5
    kubectl wait pod ${podname} -n ${OPERATOR_NAMESPACE} --for condition=Ready --timeout=1000s > /dev/null 2>&1
    export hostIP=$(kubectl get po multi-nicd-stub-${i} -n ${OPERATOR_NAMESPACE} -oyaml|yq .status.podIP)
    export nodename=kwok-node-$i
    yq e '(.metadata.name=env(nodename)),(.metadata.labels."kubernetes.io/hostname"=env(nodename)),(.status.addresses[0].address=env(hostIP))' deploy/template/fake-node.tpl|kubectl apply -f - > /dev/null 2>&1
    kubectl wait node ${nodename} --for condition=Ready --timeout=1000s > /dev/null 2>&1
}

deploy_n_node() {
    from=$1
    to=$2
    pids=""
    i=$from
    while [ "$i" -le $to ]; do
        _deploy_node $i&
        pids="$pids $!"
        i=$(( i + 1 ))
    done 
    wait $pids
}

_delete_node() {
    i=$1
    export podname=multi-nicd-stub-$i
    kubectl delete pod ${podname} -n ${OPERATOR_NAMESPACE} > /dev/null 2>&1
    export nodename=kwok-node-$i
    kubectl delete node ${nodename} > /dev/null 2>&1
    kubectl delete po --field-selector spec.nodeName=${nodename} -n ${OPERATOR_NAMESPACE} --grace-period=0 > /dev/null 2>&1
}

delete_n_node() {
    from=$1
    to=$2
    pids=""
    i=$from
    while [ "$i" -le $to ]; do
        _delete_node $i&
        pids="$pids $!"
        i=$(( i + 1 ))
    done 
    wait $pids   
}

wait_n() {
    n=$1
    while [[ $(kubectl get multinicnetwork -o 'jsonpath={..status.discovery.cidrProcessed}') != $n ]]; do print_discovery_status && sleep 5; done
}

wait_n_old_way() {
    n=$1
    len=$(kubectl get cidr -o 'jsonpath={..spec.cidr[0].hosts}'| jq '.|length')
    while [[ $len != $n ]]; do 
        len=$(kubectl get cidr -o 'jsonpath={..spec.cidr[0].hosts}'| jq '.|length')
        echo $len $(date -u +"%Y-%m-%dT%H:%M:%SZ")
        sleep 5
    done
}

print_discovery_status() {
   kubectl get multinicnetwork -o custom-columns=NAME:.metadata.name,Total:.status.discovery.existDaemon,Available:.status.discovery.infoAvailable,Processed:.status.discovery.cidrProcessed 
}

clean_fake_cni() {
    kubectl delete po --selector app=cni-stub -n ${OPERATOR_NAMESPACE} > /dev/null 2>&1
    kubectl delete job --selector app=cni-stub -n ${OPERATOR_NAMESPACE} > /dev/null 2>&1
}

add_pod() {
    from=$1
    to=$2
    starti=$3
    n=$4
    i=$from
    while [ "$i" -le $to ]; do
        export hostName=kwok-node-$i
        export hostIP=$(kubectl get po multi-nicd-stub-${i} -n ${OPERATOR_NAMESPACE} -oyaml|yq .status.podIP)
        export jobName=cni-${hostName}
        export args="./cni --command=add --start=${starti} --n=${n} --host=${hostName} --dip=${hostIP}"
        export hostIP=$(kubectl get po multi-nicd-stub-${i} -n ${OPERATOR_NAMESPACE} -oyaml|yq .status.podIP)
        yq e '(.metadata.name=env(jobName)),(.spec.template.spec.containers[0].args=[env(args)])' deploy/template/cni-stub-job.tpl|kubectl apply -f - > /dev/null 2>&1
        i=$(( i + 1 ))
    done 
    kubectl wait --for=condition=complete --timeout=1000s job --selector app=cni-stub -n ${OPERATOR_NAMESPACE} > /dev/null 2>&1
    clean_fake_cni
}

add_pod_with_step() {
    from=$1
    to=$2
    step=$3
    starti=$4
    n=$5
    i=$from
    while [ "$i" -le $to ]; do
        nfrom=$i
        nto=$(( i + $step ))
        add_pod $nfrom $nto $starti $n
        i=$nto
    done
}

delete_pod() {
    from=$1
    to=$2
    starti=$3
    n=$4
    i=$from
    while [ "$i" -le $to ]; do
        export hostName=kwok-node-$i
        export hostIP=$(kubectl get po multi-nicd-stub-${i} -n ${OPERATOR_NAMESPACE} -oyaml|yq .status.podIP)
        export jobName=cni-${hostName}
        export args="./cni --command=delete --start=${starti} --n=${n} --host=${hostName} --dip=${hostIP}"
        export hostIP=$(kubectl get po multi-nicd-stub-${i} -n ${OPERATOR_NAMESPACE} -oyaml|yq .status.podIP)
        yq e '(.metadata.name=env(jobName)),(.spec.template.spec.containers[0].args=[env(args)])' deploy/template/cni-stub-job.tpl|kubectl apply -f - > /dev/null 2>&1
        i=$(( i + 1 ))
    done 
    kubectl wait --for=condition=complete --timeout=1000s job --selector app=cni-stub -n ${OPERATOR_NAMESPACE} > /dev/null 2>&1
    clean_fake_cni
}

delete_pod_with_step() {
    from=$1
    to=$2
    step=$3
    starti=$4
    n=$5
    i=$from
    while [ "$i" -le $to ]; do
        nfrom=$i
        nto=$(( i + $step ))
        delete_pod $nfrom $nto $starti $n
        i=$nto
    done
}

check_ippool() {
    from=$1
    to=$2
    n=$3
    i=$from
    while [ "$i" -le $to ]; do
        export "nodename=kwok-node-$i"
        ippools=$(kubectl get ippool -o custom-columns=NAME:.metadata.name,HOST:.spec.hostName|grep $nodename$|awk {'print $1'})
        for ippool in $ippools; do
            export len=$(kubectl get ippool $ippool -o json| jq '.spec.allocations | length')
            if [ "$len" != "$n" ] ; then
                echo >&2 "Fatal error: $ippool of $nodename $len != $n"
                exit 2
            else
                echo "IPPool $ippool of $nodename checked ($n)"
            fi
        done
        i=$(( i + 1 ))
    done 
}

check_ippool_all() {
    n=$1
    ippools=$(kubectl get ippools -ojson|jq .items)
    for row in $(echo "${ippools}" | jq -r '.[] | @base64'); do
        _jq() {
            echo ${row} | base64 --decode | jq -r ${1}
        }
        export allocations=$(_jq '.spec.allocations')
        export len=$(echo $allocations| jq '. | length')
        if [ "$len" != "$n" ] ; then
            echo >&2 "Fatal error: $len != $n"
            exit 2
        else
            echo "IPPool checked ($n)"
        fi
    done
}

check_hostinterface() {
    from=$1
    to=$2
    i=$from
    while [ "$i" -le $to ]; do
        export nodename=kwok-node-$i
        len=$(kubectl get hostinterface ${nodename} -ojson|jq '.spec.interfaces|length')
        if [ "$len" != 3 ] ; then
            echo >&2 "Fatal error: ${nodename} interface length $len != 3"
            exit 2
        else
            echo "HostInterface checked (${nodename})"
        fi
        i=$(( i + 1 ))
    done 
}

check_cidr(){
    from=$1
    to=$2
    cidr=$(kubectl get cidr multi-nic-sample -ojson|jq .spec.cidr)
    n=$((${to}-${from}+1))
    vlanlen=$(echo $cidr| jq '. | length')
    if [ "$n" != 0 ] && [ "$vlanlen" != 3 ] ; then
        echo >&2 "Fatal error: interface length $n != 0 and $len != 3"
        exit 2
    else
        i=0
        while [ "$i" -le 2 ]; do
            hosts=$(echo $cidr| jq .[${i}].hosts)
            hostlen=$(echo $hosts| jq '.|length')
            if [ "$hostlen" != $n ] ; then
                echo >&2 "Fatal error: host length $hostlen != $n"
                exit 2
            fi
            i=$(( i + 1 ))
        done 
        echo "CIDR checked (${n})"
    fi
}

watch_network() {
    kubectl get multinicnetwork -o custom-columns=NAME:.metadata.name,Total:.status.discovery.existDaemon,Available:.status.discovery.infoAvailable,Processed:.status.discovery.cidrProcessed,Time:.status.lastSyncTime -w
}

test_scale() {
    deploy_network 8
	echo $(date -u +"%Y-%m-%dT%H:%M:%SZ")
	START=$(date +%s)
	time deploy_n_node 1 100
	time wait_n 100
	check_hostinterface 1 100
	check_cidr 1 100
	time deploy_n_node 101 200
	time wait_n 200
	check_hostinterface 1 200
	check_cidr 1 200
	test_clean
    END=$(date +%s)
	echo $((END-START)) | awk '{print "Test time: "int($1/60)":"int($1%60)}'
}

test_step_scale() {
    deploy_network 8
	echo $(date -u +"%Y-%m-%dT%H:%M:%SZ")
	START=$(date +%s)
	time deploy_n_node 1 10
	time wait_n 10
	check_cidr 1 10
	time deploy_n_node 11 20
	time wait_n 20
	check_cidr 1 20
	time deploy_n_node 21 50
	time wait_n 50
	check_cidr 1 50
	time deploy_n_node 51 100
	time wait_n 100
	check_cidr 1 100
	time deploy_n_node 101 200
	time wait_n 200
	check_cidr 1 200
	test_step_clean
	export END=$(date +%s)
	echo $((END-START)) | awk '{print "Test time: "int($1/60)":"int($1%60)}'
}

test_clean() {
	time delete_n_node 101 200
	time wait_n 100
	check_cidr 1 100
	time delete_n_node 1 100
	time wait_n 0
	check_cidr 1 0
}

test_step_clean() {
    time delete_n_node 101 200
	time wait_n 100
    check_cidr 1 100
	time delete_n_node 51 100
	time wait_n 50
	check_cidr 1 50
	time delete_n_node 21 50
	time wait_n 20
	check_cidr 1 20
	time delete_n_node 11 20
	time wait_n 10
	check_cidr 1 10
	time delete_n_node 1 10
	time wait_n 0
    check_cidr 1 0
}

test_allocate() {
    echo "Deploying 10 nodes"
    time deploy_n_node 1 10
	time wait_n 10
	check_cidr 1 10
    echo "Start test"
	export START=$(date +%s)
	time add_pod 1 10 1 5
	check_ippool 1 10 5
	time delete_pod 1 10 1 5
	check_ippool 1 10 0
	export END=$(date +%s)
	echo $((END-START)) | awk '{print "Test time: "int($1/60)":"int($1%60)}'
    echo "Cleaning nodes"
	time delete_n_node 1 10
	time wait_n 0
	check_cidr 1 0
}

test_taint() {
    echo "Deploying 5 nodes"
    time deploy_n_node 1 5
	time wait_n 5
	check_cidr 1 5
    echo "Taint a node"
    kubectl taint node kwok-node-5 test=true:NoSchedule
    echo "Add pods to non-taint"
	time add_pod 1 4 1 1
	check_ippool 1 4 1
    echo "Untaint"
    kubectl taint node kwok-node-5 test-
    echo "Add pods to untaint node"
	time add_pod 5 5 1 1
	check_ippool 1 5 1
    echo "Taint already allocated node"
    kubectl taint node kwok-node-1 test=true:NoSchedule
    check_ippool 1 5 1
    echo "Untaint"
    kubectl taint node kwok-node-1 test-
    check_ippool 1 5 1
    time delete_pod 1 5 1 1
	check_ippool 1 5 0
    echo "Cleaning nodes"
	time delete_n_node 1 5
	time wait_n 0
	check_cidr 1 0
}

"$@"