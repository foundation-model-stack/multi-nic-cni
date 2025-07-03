# Alpha 1.2 Channel

## v1.2.7
- support macvlan plugin
- code enhancements:
    * refactor code structure (add internal packages)
    * upgrade test to ginkgo V2
    * generate measurable test coverage results
    * improve test coverage to 60%
- fixes:
    * correct sample multinicnetwork for macvlan+whereabouts IPAM
    * handle error from ghw.PCI call

## v1.2.6

- upgrade go version
  * controller: GO 1.22
  * daemon, CNI: GO 1.23
- remove kube-rbac-proxy
- add make `set_version` target to simplify release steps
- update concept image, user and contributing guide
- rewrite the highlighted features and add demo and references
- fix bugs: 
    * [sample-concheck make error](https://github.com/foundation-model-stack/multi-nic-cni/pull/235)
    * [failed to load netconf: post fail: Post "http://localhost:11000/select": EOF](https://github.com/foundation-model-stack/multi-nic-cni/issues/240)

## v1.2.5

- support multiple resource names defined in NicClusterPolicy for Mellanox Host Device use case
- remove unnecessary selection policy call when network devices have already selected by the device plugin