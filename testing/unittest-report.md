# Unit Test

Test | Description | File 
---|---|---
| Common IPPool Test/checkPoolValidity | contains excluded address | ./controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | empty address | ./controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | empty exclude | ./controllers/ippool_test.go |
| Common IPPool Test/checkPoolValidity | nil | ./controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | cover | ./controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | subset | ./controllers/ippool_test.go |
| Common IPPool Test/extractMatchExcludesFromPodCIDR | unrelated | ./controllers/ippool_test.go |
| Common Plugin Test | RemoveEmpty | ./internal/plugin/plugin_test.go |
| Config Test | Check update from ConfigSpec | ./controllers/config_test.go |
| Daemon Test | Test TryGetDaemonPod for tainted daemon | ./controllers/daemon_test.go |
| Daemon Test | Test TryGetDaemonPod for valid daemon | ./controllers/daemon_test.go |
| DynamicHandler/GetFirst | should handle unmarshalling errors | ./internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return error when no items exist | ./internal/plugin/dynamic_test.go |
| DynamicHandler/GetFirst | should return the first item when items exist | ./internal/plugin/dynamic_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can add new while leave old one | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can check no change | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can detect change | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with a single device | can leave old one | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can check no change in swop order | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing | ./controllers/hostinterface_test.go |
| Host Interface Test/UpdateNewInterfaces - original with more than one devices | can leave old one when some is missing and some with new info | ./controllers/hostinterface_test.go |
| Mellanox Plugin/GetConfig | should generate valid CNI config | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle empty resource list | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle invalid IPAM config | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle invalid resource list | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should handle resources with different prefixes | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should return empty list when no resources are available | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/GetConfig | should successfully retrieve SrIoV resources from policy | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/Init | should initialize successfully with valid config | ./internal/plugin/mellanox_test.go |
| Mellanox Plugin/Init/when config is invalid | should return error | ./internal/plugin/mellanox_test.go |
| NetAttachDef test/handler | create and delete | ./internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler | finalizer and owner reference work together for NAD cleanup | ./internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different annotation count | ./internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different annotation values | ./internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | different configurations | ./internal/plugin/net_attach_def_test.go |
| NetAttachDef test/handler/CheckDefChanged/comparing network attachment definitions | identical definitions | ./internal/plugin/net_attach_def_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Dynamically compute CIDR | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Empty subnet | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/IPAM | Empty subnet and interfaces | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/InitCustomCRCache | should initialize IPPool and HostInterface caches | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasActivePod | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | hasNewHost and hasActivePod | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/Sync/CIDR and IPPool | simple | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should delete CIDR when no corresponding MultiNicNetwork exists | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should keep CIDR when corresponding MultiNicNetwork exists | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Handler functions/SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR | should sync network attachments and update internal state | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetAllNetAddrs | returns all unique network addresses from HostInterfaceHandler | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | handles CIDREntries with empty Host lists | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | returns a map from (host name, interface index) to HostInterfaceInfo of CIDR | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/GetHostInterfaceIndexMap | returns an empty map when there are no CIDR entries | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[1] | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[2] | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | index at bytes[3] | ./controllers/cidr_handler_test.go |
| Test CIDR Handler\t/Util functions/Sync CIDR/IPPool/Getting index in range | uncontained address | ./controllers/cidr_handler_test.go |
| Test CIDRCompute/CheckIfTabuIndex | cover tabu index | ./internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | no excludes | ./internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | not tabu index | ./internal/compute/compute_test.go |
| Test CIDRCompute/CheckIfTabuIndex | tabu index | ./internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid CIDR | ./internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | invalid Index | ./internal/compute/compute_test.go |
| Test CIDRCompute/ComputeNet | simple | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in left part | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in middle | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | available index in right part | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | empty indexes | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | multiple index assigned | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index assigned | ./internal/compute/compute_test.go |
| Test CIDRCompute/FindAvailableIndex | single index available | ./internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | different CIDR and IP | ./internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | invalid CIDR and IP | ./internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | same CIDR and IP | ./internal/compute/compute_test.go |
| Test CIDRCompute/GetIndexInRange | valid CIDR and IP | ./internal/compute/compute_test.go |
| Test Compute Utils/SortAddress | empty slice | ./internal/compute/util_test.go |
| Test Compute Utils/SortAddress | single ip | ./internal/compute/util_test.go |
| Test Compute Utils/SortAddress | sorted ips | ./internal/compute/util_test.go |
| Test Compute Utils/SortAddress | unsorted ips | ./internal/compute/util_test.go |
| Test Config Controller | default config | ./controllers/config_controller_test.go |
| Test Config Controller/Multus | get CNI path | ./controllers/config_controller_test.go |
| Test GetConfig of main plugins | aws-ipvlan main plugin | ./internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | ipvlan main plugin | ./internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | macvlan main plugin | ./internal/plugin/plugin_test.go |
| Test GetConfig of main plugins | mellanox main plugin - GetSrIoVResource | ./internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | with resource name | ./internal/plugin/plugin_test.go |
| Test GetConfig of main plugins/SR-IoV | without resource name | ./internal/plugin/plugin_test.go |
| Test definition changes check | detect annotation change | ./controllers/multinicnetwork_test.go |
| Test definition changes check | detect config change | ./controllers/multinicnetwork_test.go |
| Test definition changes check | detect no change | ./controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition | ./controllers/multinicnetwork_test.go |
| Test deploying MultiNicNetwork | successfully create/delete network attachment definition on new namespace | ./controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change from no status | ./controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on compute results | ./controllers/multinicnetwork_test.go |
| Test multinicnetwork status change check | detect change on simple status | ./controllers/multinicnetwork_test.go |
| Unsync IPPool Test | All new | ./controllers/ippool_test.go |
| Unsync IPPool Test | Already synced | ./controllers/ippool_test.go |
| Unsync IPPool Test | Assignment on one interface is missing | ./controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending | ./controllers/ippool_test.go |
| Unsync IPPool Test | Deleted pods pending and new pod assigned the same index | ./controllers/ippool_test.go |
| Unsync IPPool Test | Duplicated allocation | ./controllers/ippool_test.go |
| Unsync IPPool Test | Same pod different IP | ./controllers/ippool_test.go |
| Unsync IPPool Test | Should all clean | ./controllers/ippool_test.go |
