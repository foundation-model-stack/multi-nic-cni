# Unit Test

Test | Description | File 
---|---|---
| Join | empty greet ack | /usr/local/build/daemon/src/main_test.go |
| Join | empty join | /usr/local/build/daemon/src/main_test.go |
| Test Allocation | anomaly allocate after some allocations | /usr/local/build/daemon/src/main_test.go |
| Test Allocation | anomaly allocate from begining | /usr/local/build/daemon/src/main_test.go |
| Test Allocation | normaly allocate | /usr/local/build/daemon/src/main_test.go |
| Test Allocator/Allocate/FindAvailableIndex | excludes consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/FindAvailableIndex | excludes non-consecutive and then consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/FindAvailableIndex | excludes non-consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/FindAvailableIndex | no excludes | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/allocateIP | first allocation | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/allocateIP | no interface name | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/allocateIP | no ippool | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/allocateIP | reuse allocation | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/allocateIP | second allocation | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getAddressByIndex | first index | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getAddressByIndex | shifted index | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getAddressByIndex | zero index | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getExcludeRanges | empty | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getExcludeRanges | inner exclude | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getExcludeRanges | multiple inner excludes | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Allocate/getExcludeRanges | outer exclude | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/Deallocate | force expired | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/with IPPool | Allocate-DeallocateIP | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator/with IPPool | CleanHangingAllocation | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Get Interfaces | get interfaces | /usr/local/build/daemon/src/main_test.go |
| Test L3Config Add/Delete | apply/delete l3config | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Get resource map | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Init NumaAwareSelector from sysfs | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Init NumaAwareSelector from topology file | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select all nic | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select nic by NumaAwareSelector (topology) | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select nic by dev class | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select one nic | /usr/local/build/daemon/src/main_test.go |
| Test Route Add/Delete | add/delete route | /usr/local/build/daemon/src/main_test.go |
| Test Route/API/Add/DeleteRoute | invalid request | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/API/Add/DeleteRoute | valid request | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/API/ApplyL3Config/DeleteL3Config | not-existing delete request | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/API/ApplyL3Config/DeleteL3Config | valid request | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/RT Path | Get local RT | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/RT Path | Set RT path | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/Table/getRoutesFromL3Config | not-existing table | /usr/local/build/daemon/src/router/router_test.go |
| Test Route/Table/getRoutesFromL3Config | valid table | /usr/local/build/daemon/src/router/router_test.go |
| Test VF/PF Interface Mapping/Interface matching logic | should match VF interface to corresponding PF in IPPool | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/Interface matching logic | should match regular interface directly | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/Interface matching logic | should not match when VF maps to different PF | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/getPFInterfaceName function | should return PF name when single PF found | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/getPFInterfaceName function | should return original name when physfn directory doesn't exist | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/isVF function | should return false for non-existent interface | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/isVF function | should return false for regular interface (non-VF) | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test VF/PF Interface Mapping/isVF function | should return true for VF interface | /usr/local/build/daemon/src/allocator/allocator_test.go |
