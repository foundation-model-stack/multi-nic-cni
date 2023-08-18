# v1.2.0 [alpha]

* Topology-aware NIC Selection
* RoCE GDR-support CNI (NVIDIA MOFED operator) - `mellanox`
    - Host-device CNI support
    - NICClusterPolicy aware

    ```yaml
    apiVersion: multinic.fms.io/v1
    kind: MultiNicNetwork
    metadata:
    name: multinic-mellanox-hostdevice
    spec:
    subnet: ""
    ipam: |
        {
            "type": "host-device-ipam"
        }
    multiNICIPAM: false
    plugin:
        cniVersion: "0.3.1"
        type: mellanox
    ```
    
* Unmanaged HostNetworkInterface for IP-less network device
* Multi-gateway route configuration support