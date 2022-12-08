# Safe live migration
To reinstall/upgrade multi-nic-cni-operator without affecting workloads running with L3 route configuration.
#### Requirement commands
- [jq](https://stedolan.github.io/jq/download/)
- [yq](https://github.com/mikefarah/yq/#install)
- [watch](https://www.2daygeek.com/linux-watch-command-to-monitor-a-command/) - optional

### Steps
1. Clone and checkout this branch.
   ```bash
   git clone https://github.com/foundation-model-stack/multi-nic-cni.git
   cd multi-nic-cni
   git checkout -b doc origin/doc
   cd live_migration
   chmod +x ./live_migrate.sh

   ##  If operator is not installed in the openshift-operators namespace, also run
   # export OPERATOR_NAMESPACE=<deployed namespace>
   ```
2. Snapshot multi-nic-cni CR on your cluster
    ```bash
    CLUSTER_NAME=<cluster-name>
    ./live_migrate.sh snapshot $CLUSTER_NAME
    ```
    > rename multinicnetwork_l2.yaml with "multinic-ipvlanl3"<br>cidr.multinic.yaml          hostinterface.multinic.yaml ippool.multinic.yaml        multinicnetwork.yaml<br>saved in snapshot-a100-huge

    This will create a folder containing relevant CR and update multinetwork name in `multinicnetwork_l2.yaml`
3. Deactivate controller from updating route on host
    ```bash
    ./live_migrate.sh deactivate_route_config
    ```
    > multinicnetwork.multinic.fms.io/multinic-ipvlanl3 configured
    ...
4. Uninstall operator
    ```bash
    ./live_migrate.sh uninstall_operator
    ```
    For OperatorSDK, run `operator-sdk cleanup --delete-all`<br>
    Wait until all multi-nicd daemon is terminated<br>
    ```bash
    watch kubectl get po -n openshift-operators
    ```
5. Reinstall operator
   
    5.1 install operator via GUI (recommended). For other installation, check [Installation Guide](https://foundation-model-stack.github.io/multi-nic-cni/user_guide/#quick-installation).

    If multi-nicd image also need to be updated, run
    ```bash
    ./live_migrate.sh patch_daemon
    ```
    Wait until multi-nicd daemon is all running:
    ```bash
    ./live_migrate.sh wait_daemon
    ```
    Check if CRs are deleted or not (not deleted by default):
    ```bash
    kubectl get cidr
    # NAME                AGE
    # multinic-ipvlanl3   
    ```
    *[if CRs are deleted (for example, by operator-sdk cleanup), do 5.2 - 5.4]* <br>
    5.2 deploy dump multinicnetwork with l2
    ```bash
    ./live_migrate.sh deactivate_route_config
    ```
    > multinicnetwork.multinic.fms.io/multinic-ipvlanl3 configured <br>"{\"cniVersion\":\"0.3.0\",\"name\":\"multinic-ipvlanl3\",\"type\":\"multi-nic\",\"ipam\":null,\"dns\":{},\"plugin\":{\"cniVersion\":\"0.3.0\",\"mode\":\"l2\",\"type\":\"ipvlan\"},\"subnet\":\"192.168.0.0/16\",\"masterNets\":[\"10.241.130.0/24\",\"10.241.131.0/24\"],\"daemonIP\":\"\",\"daemonPort\":11000}"

    5.3 apply snapshot status CR
    ```bash
    ./live_migrate.sh deploy_status_cr $CLUSTER_NAME
    ```
    5.4 restart controller to activate cache initialization
    ```bash
    ./live_migrate.sh restart_controller
    ```
    > Wait for deployment to be available<br>deployment.apps/multi-nic-cni-operator-controller-manager condition met<br>Wait for config to be ready...<br>Config Ready
6. activate route config
    ```bash
    ./live_migrate.sh activate_route_config $CLUSTER_NAME
    ```
    > multinicnetwork.multinic.fms.io/multinic-ipvlanl3 configured<br>"{\"cniVersion\":\"0.3.0\",\"name\":\"multinic-ipvlanl3\",\"type\":\"multi-nic\",\"ipam\":{\"hostBlock\":8,\"interfaceBlock\":2,\"type\":\"multi-nic-ipam\",\"vlanMode\":\"l3\"},\"dns\":{},\"plugin\":{\"cniVersion\":\"0.3.0\",\"mode\":\"l3\",\"type\":\"ipvlan\"},\"subnet\":\"192.168.0.0/16\",\"masterNets\":[\"10.241.130.0/24\",\"10.241.131.0/24\"],\"multiNICIPAM\":true,\"daemonIP\":\"\",\"daemonPort\":11000}"
