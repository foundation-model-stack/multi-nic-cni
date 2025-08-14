# Unit Test

Test | Description | File 
---|---|---
| Common IPPool Test/checkPoolValidity | contains excluded address | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | empty address | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | empty exclude | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | nil | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | cover | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | subset | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | unrelated | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/ippool_test.go |
| Common Plugin Test | RemoveEmpty | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Config Test | Check update from ConfigSpec | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/config_test.go |
| Daemon Test | Test TryGetDaemonPod for tainted daemon | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/daemon_test.go |
| Daemon Test | Test TryGetDaemonPod for valid daemon | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/daemon_test.go |
| DynamicHandler/GetFirst | should handle unmarshalling errors | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return error when no items exist | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return the first item when items exist | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/dynamic_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can add new while leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can check no change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can detect change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change in swop order | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing and some with new info | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/hostinterface_test.go |
| Mellanox Plugin/GetConfig | should generate valid CNI config | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle empty resource list | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle invalid IPAM config | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle invalid resource list | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle resources with different prefixes | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should return empty list when no resources are available | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should successfully retrieve SrIoV resources from policy | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/Init | should initialize successfully with valid config | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| Mellanox Plugin/Init/when config is invalid | should return error | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/mellanox_test.go |
| NetAttachDef test/handler | create and delete | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler | finalizer and owner reference work together for NAD cleanup | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different annotation count | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different annotation values | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different configurations | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | identical definitions | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/net_attach_def_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Dynamically compute CIDR | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Empty subnet | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Empty subnet and interfaces | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/InitCustomCRCache | should initialize IPPool and HostInterface caches | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasActivePod | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost and hasActivePod | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | simple | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should delete CIDR when no corresponding MultiNicNetwork exists | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should keep CIDR when corresponding MultiNicNetwork exists | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should sync network attachments and update internal state | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetAllNetAddrs | returns all unique network addresses from HostInterfaceHandler | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | handles CIDREntries with empty Host lists | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | returns a map from (host name, interface index) to HostInterfaceInfo of CIDR | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | returns an empty map when there are no CIDR entries | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[1] | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[2] | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[3] | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | uncontained address | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/cidr_handler_test.go |
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
| Test Config Controller | default config | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/config_controller_test.go |
| Test Config Controller/Multus | get CNI path | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/controllers/config_controller_test.go |
| Test GetConfig of main plugins | aws-ipvlan main plugin | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | ipvlan main plugin | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | macvlan main plugin | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | mellanox main plugin - GetSrIoVResource | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | with resource name | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | without resource name | /Users/aa404681/Documents/internal_ws/cni/multi-nic-cni-operator/internal/plugin/plugin_test.go |
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
