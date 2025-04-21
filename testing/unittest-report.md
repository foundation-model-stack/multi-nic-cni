# Unit Test

Test | Description | File 
---|---|---
| Config Test | Check update from ConfigSpec | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/config_test.go |
| Daemon Test | Test TryGetDaemonPod for tainted daemon | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/daemon_test.go |
| Daemon Test | Test TryGetDaemonPod for valid daemon | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/daemon_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can add new while leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can check no change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can detect change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change in swop order | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing and some with new info | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Test CIDRCompute/CheckIfTabuIndex | cover tabu index | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | no excludes | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | not tabu index | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | tabu index | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid CIDR | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid Index | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | simple | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in left part | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in middle | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in right part | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | empty indexes | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | multiple index assigned | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index assigned | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index available | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | different CIDR and IP | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | invalid CIDR and IP | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | same CIDR and IP | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | valid CIDR and IP | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/compute_test.go |
| Test Compute Utils/SortAddress | empty slice | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | single ip | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | sorted ips | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | unsorted ips | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/compute/util_test.go |
| Test GetConfig of main plugins | ipvlan main plugin | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/plugin_test.go |
| Test GetConfig of main plugins | macvlan main plugin | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/plugin_test.go |
| Test GetConfig of main plugins | mellanox main plugin - GetSrIoVResource | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/plugin_test.go |
| Test GetConfig of main plugins | sriov main plugin with resource name | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/plugin_test.go |
| Test GetConfig of main plugins | sriov main plugin without resource name | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/plugin_test.go |
| Test Multi-NIC IPAM | Dynamically compute CIDR | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicipam_test.go |
| Test Multi-NIC IPAM | Empty subnet | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicipam_test.go |
| Test Multi-NIC IPAM | Sync CIDR/IPPool | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicipam_test.go |
| Test definition changes check | detect annotation change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test definition changes check | detect config change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test definition changes check | detect no change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition on new namespace | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change from no status | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on compute results | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on simple status | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/multinicnetwork_test.go |
| Unsync IPPool Test | All new | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Already synced | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Assignment on one interface is missing | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending and new pod assigned the same index | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Duplicated allocation | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Same pod different IP | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Unsync IPPool Test | Should all clean | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
