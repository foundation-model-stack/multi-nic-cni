# Alpha Channel

![](../img/alpha-release.png)

## v1.2.1

**Improvements:**

* Unmanaged HostNetworkInterface for IP-less network device 
    - zero host block/zero interface block

            apiVersion: multinic.fms.io/v1
            kind: MultiNicNetwork
            metadata:
                name: multinic-unmanaged
            spec:
                ipam: |
                    {
                    "type": "multi-nic-ipam",
                    "hostBlock": 0, 
                    "interfaceBlock": 0,
                    "vlanMode": "l2"
                    }
                multiNICIPAM: true
                plugin:
                    cniVersion: "0.3.0"
                    type: ipvlan
                    args: 
                        mode: l2

    - specify static cidr of each host

            apiVersion: multinic.fms.io/v1
            kind: HostInterface
            metadata:
                name: node-1
                labels:
                  unmanaged: true
            hostName: node-1
            interfaces:
            -   hostIP: ""
                interfaceName: eth1
                netAddress: 192.168.0.0/24
            -   hostIP: ""
                interfaceName: eth2
                netAddress: 192.168.1.0/24
                
* Multi-gateway route configuration support

        apiVersion: multinic.fms.io/v1
        kind: MultiNicNetwork
        metadata:
            name: multinic-multi-gateway
        spec:
            ipam: |
                {
                "type": "multi-nic-ipam",
                ...
                "routes": [{"dst": "10.0.0.0/24","gw": "1.1.1.1"}, {"dst": "10.0.0.0/24","gw": "2.2.2.2"}]
                }
            multiNICIPAM: true

    The above definition will generate the following route on pod:
    
    `10.0.0.0/24 nexthop via 1.1.1.1 nexthop via 2.2.2.2`