# Unit Test

Test | Description | File 
---|---|---
| Test Allocation | anomaly allocate after some allocations | /usr/local/build/daemon/src/main_test.go |
| Test Allocation | anomaly allocate from begining | /usr/local/build/daemon/src/main_test.go |
| Test Allocation | normaly allocate | /usr/local/build/daemon/src/main_test.go |
| Test Allocator | find next available index with exclude range over consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator | find next available index with exclude range over non-consecutive and then consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator | find next available index with exclude range over non-consecutive order | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator | find simple next available index | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Allocator | force expired | /usr/local/build/daemon/src/allocator/allocator_test.go |
| Test Get Interfaces | get interfaces | /usr/local/build/daemon/src/main_test.go |
| Test L3Config Add/Delete | apply/delete l3config | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Get resource map | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Init NumaAwareSelector from sysfs | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | Init NumaAwareSelector from topology file | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select all nic | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select nic by NumaAwareSelector (topology) | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select nic by dev class | /usr/local/build/daemon/src/main_test.go |
| Test NIC Select | select one nic | /usr/local/build/daemon/src/main_test.go |
| Test Path | Set RT path | /usr/local/build/daemon/src/router/router_test.go |
| Test RT Table | Add then delete new table. | /usr/local/build/daemon/src/router/router_test.go |
| Test RT Table | Get local RT | /usr/local/build/daemon/src/router/router_test.go |
