# Safe live migration
To reinstall/upgrade multi-nic-cni-operator without affecting workloads running with L3 route configuration.
#### Requirement commands
- [jq](https://stedolan.github.io/jq/download/)
- [yq](https://github.com/mikefarah/yq/#install)
- [watch](https://www.2daygeek.com/linux-watch-command-to-monitor-a-command/) - optional

### Pre-steps
1. Clone and make script be executable
    ```bash
    git clone https://github.com/foundation-model-stack/multi-nic-cni.git
    cd multi-nic-cni/live_migration
    chmod +x ./live_migrate.sh
    ```
2. If operator is not installed in the `openshift-operators` namespace, run
    ```bash
    export OPERATOR_NAMESPACE=<deployed-namespace>
    ```
3. To confirm that there is no interruption to deployed pods, try live iperf3 between any two nodes with at least one secondary NIC on another terminal:
    ```bash
    export SERVER_HOST_NAME=<target-server-node-name>
    export CLIENT_HOST_NAME=<target-client-node-name>
    export LIVE_TIME=2000 # expected time to perform migration (s)
    ./live_migrate.sh live_iperf3 ${SERVER_HOST_NAME} ${CLIENT_HOST_NAME} ${LIVE_TIME}
    ```
### Live Migration Steps

1. Snapshot multi-nic-cni CR on your cluster
    ```bash
    export CLUSTER_NAME=<cluster-name> # snapshot is saved to `snapshot/default` folder if not set
    ./live_migrate.sh snapshot
    ```
    ```
    # expected output
    rename multinicnetwork_l2.yaml with "<multinicnetwork-name>"
    cidr.multinic.yaml          hostinterface.multinic.yaml 
    ippool.multinic.yaml        multinicnetwork.yaml
    saved in snapshot/<cluster-name>
    ```
    This will create a folder containing relevant CR and update multinetwork name in `multinicnetwork_l2.yaml`
2. Deactivate controller from updating route on host

    2.1 Deactivate route configuration
    ```bash
    ./live_migrate.sh deactivate_route_config
    ```
    ```
    # expected output
    multinicnetwork.multinic.fms.io/<multinicnetwork-name> configured
    Deactivate route configuration.
    ```
    2.2 For migrating to new channel with updated CRDs, clean the old resources first.
    ```bash
    ./live_migrate.sh clean_resource
    ```

3. Uninstall operator

    3.1 Uninstall by CLI
    ```bash
    OLD_VERSION=<version-to-uninstall> # e.g., OLD_VERSION=1.0.2
    ./live_migrate.sh uninstall_operator $OLD_VERSION 
    ```
    For OperatorSDK, run `operator-sdk cleanup multi-nic-cni-operator --delete-all`<br>
    3.2 Wait until all multi-nicd daemon is terminated<br>
    ```bash
    ./live_migrate.sh wait_daemon_terminated
    ```
    3.3 For migrating to new channel with updated CRDs, need to also clean CRD.
    ```bash
    ./live_migrate.sh clean_crd
    ```

4. Reinstall operator
   
    4.1 install operator via GUI (recommended). For other installation, check [Installation Guide](https://foundation-model-stack.github.io/multi-nic-cni/user_guide/#quick-installation).

    4.2 If multi-nicd image also need to be updated, run
    ```bash
    ./live_migrate.sh patch_daemon
    ```

    4.3 Wait until multi-nicd daemon is all running:
    ```bash
    ./live_migrate.sh wait_daemon
    ```
    4.4 Check if CRs are deleted or not (not deleted by default):
    ```bash
    kubectl get cidr
    ```
    ```
    # expected output if CRs are deleted
    No resources found
    ```
    If CRs are deleted (for example, by operator-sdk cleanup or with CRD updates, by updated CDR), do 4.5 - 4.7
    
    4.5 deploy dump multinicnetwork
    ```bash
    ./live_migrate.sh deactivate_route_config
    ```
    ```
    # expected output
    multinicnetwork.multinic.fms.io/<multinicnetwork-name> configured
    Deactivate route configuration.
    ```
    4.6 apply snapshot status CR
    ```bash
    ./live_migrate.sh deploy_status_cr
    ```
    4.7 restart controller to activate cache initialization
    ```bash
    ./live_migrate.sh restart_controller
    ```
    ```
    # expected output
    Wait for deployment to be available
    deployment.apps/multi-nic-cni-operator-controller-manager condition met
    Wait for config to be ready...
    ...
    Config Ready
    ```

5. Activate route config
    ```bash
    ./live_migrate.sh activate_route_config
    ```
    ```
    # expected output
    multinicnetwork.multinic.fms.io/<multinicnetwork-name> configured
    Activate route configuration.
    ```
    
6. Check multinicnetwork status (available from v1.0.3)
   ```bash
   ./live_migrate.sh get_status
    ```
    ```
   # expected output
   NAME                ConfigStatus   RouteStatus   TotalHost   HostWithSecondaryNIC   ProcessedHost   Time
   <multinicnetwork-name>   Success        Success       5           5                      5          2023-02-02T09:31:06Z
   ```
