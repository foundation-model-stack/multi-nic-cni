# Beta Channel

![](../img/beta-release.png)

## v1.2.7

**Highlights**

- support macvlan plugin
- refactor code structure (add internal packages)

**Maintenance**

- upgrade controller test to ginkgo V2
- generate measurable controller test coverage results
- improve controller test coverage to 60%

**Fix**

- correct sample multinicnetwork for macvlan+whereabouts IPAM
- handle error from ghw.PCI call


## v1.2.0 (deprecated)

**Major feature update:**

* [Topology-aware NIC Selection](../concept/policy.md#topology-strategy)
* RoCE GDR-support CNI (NVIDIA MOFED operator) - `mellanox`
    - Host-device CNI support
    - NICClusterPolicy aware

            apiVersion: multinic.fms.io/v1
            kind: MultiNicNetwork
            metadata:
            name: multinic-mellanox-hostdevice
            spec:
                ipam: |
                    {
                        "type": "host-device-ipam"
                    }
                multiNICIPAM: false
                plugin:
                    cniVersion: "0.3.1"
                    type: mellanox

---

## v1.1.0 (deprecated)

**Major feature update:**

[Multi-cloud support](../user_guide/index.md#additional-multinicnetwork-for-specific-cloud-infrastructure)

- AWS-support CNI
    - Provide `aws-ipvlan` working with Multi-NIC IPAM
    - Support using Host subnet for Pod subnet for ENA


            apiVersion: multinic.fms.io/v1
            kind: MultiNicNetwork
            metadata:
            name: multinic-aws-ipvlan
            spec:
                ipam: |
                    {
                    "type": "multi-nic-ipam",
                    "hostBlock": 8, 
                    "interfaceBlock": 2,
                    "vlanMode": "l2"
                    }
                multiNICIPAM: true
                plugin:
                    cniVersion: "0.3.0"
                    type: aws-ipvlan
                    args: 
                    mode: l2
