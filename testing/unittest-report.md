# Unit Test

Test | Description | File 
---|---|---
| Config Test | Check update from ConfigSpec | /home/cbis-admin/set_version/multi-nic-cni/controllers/config_test.go |
| Daemon Test | Test TryGetDaemonPod for tainted daemon | /home/cbis-admin/set_version/multi-nic-cni/controllers/daemon_test.go |
| Daemon Test | Test TryGetDaemonPod for valid daemon | /home/cbis-admin/set_version/multi-nic-cni/controllers/daemon_test.go |
| DynamicHandler/GetFirst | should handle unmarshalling errors | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return error when no items exist | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return the first item when items exist | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/dynamic_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can add new while leave old one | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can check no change | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can detect change | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can leave old one | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change in swop order | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing and some with new info | /home/cbis-admin/set_version/multi-nic-cni/controllers/hostinterface_test.go |
| Mellanox Plugin/Init | should initialize successfully with valid config | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/mellanox_test.go |
| Mellanox Plugin/Init/when config is invalid | should return error | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/mellanox_test.go |
| NetAttachDef test/handler | create and delete | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged | should detect changed configurations | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged | should detect changes in annotation count | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged | should detect changes in annotations | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged | should identify unchanged definitions | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/net_attach_def_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Dynamically compute CIDR | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Empty subnet | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasActivePod | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost and hasActivePod | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | simple | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions | Sync CIDR/IPPool | /home/cbis-admin/set_version/multi-nic-cni/controllers/cidr_handler_test.go |
| Test CIDRCompute/CheckIfTabuIndex | cover tabu index | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | no excludes | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | not tabu index | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | tabu index | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid CIDR | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid Index | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | simple | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in left part | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in middle | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in right part | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | empty indexes | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | multiple index assigned | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index assigned | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index available | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | different CIDR and IP | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | invalid CIDR and IP | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | same CIDR and IP | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | valid CIDR and IP | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/compute_test.go |
| Test Compute Utils/SortAddress | empty slice | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | single ip | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | sorted ips | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/util_test.go |
| Test Compute Utils/SortAddress | unsorted ips | /home/cbis-admin/set_version/multi-nic-cni/internal/compute/util_test.go |
| Test Config Controller | default config | /home/cbis-admin/set_version/multi-nic-cni/controllers/config_controller_test.go |
| Test Config Controller/Multus | get CNI path | /home/cbis-admin/set_version/multi-nic-cni/controllers/config_controller_test.go |
| Test GetConfig of main plugins | aws-ipvlan main plugin | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | ipvlan main plugin | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | macvlan main plugin | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | mellanox main plugin - GetSrIoVResource | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | with resource name | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | without resource name | /home/cbis-admin/set_version/multi-nic-cni/internal/plugin/plugin_test.go |
| Test definition changes check | detect annotation change | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test definition changes check | detect config change | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test definition changes check | detect no change | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition on new namespace | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change from no status | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on compute results | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on simple status | /home/cbis-admin/set_version/multi-nic-cni/controllers/multinicnetwork_test.go |
| Unsync IPPool Test | All new | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Already synced | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Assignment on one interface is missing | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending and new pod assigned the same index | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Duplicated allocation | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Same pod different IP | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
| Unsync IPPool Test | Should all clean | /home/cbis-admin/set_version/multi-nic-cni/controllers/ippool_test.go |
