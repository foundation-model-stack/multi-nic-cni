# Beta Channel

## v1.1.0

**Major feature update:**

[Multi-cloud support](../user_guide/multi-cloud.md)

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
