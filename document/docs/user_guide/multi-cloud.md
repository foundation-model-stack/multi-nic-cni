# Multi-NIC CNI on Multi-Cloud

Multi-NIC CNI Features|IBM Cloud|Bare Metal|AWS|Azure (tentative)
---|---|---|---|---
Single definition for multiple attachments<br>- dynamic interface discovery<br>- policy-based NIC selection|&check;|&check;|&check;|&check;
CIDR/IP management|&check;|*|*|&check;
L3 Route configuration|&check;|X|X|&check;

> **&check;:** beneficial<br>**\*:** optional (e.g., replacable by whereabout, aws-vpc-cni IPAM)<br>**X:** non-beneficial as using L2

## Operator Installation
Multi-NIC CNI Operator for supporting multi-cloud is under developing from v1.1.0. The pre-release bundle is available on [alpha](../release/alpha.md) and [beta](../release/beta.md) channel on Openshift OKD OperatorHub.

Please check latest [release](../release/index.md).

## MultiNICNetwork Deployment

### IBM Cloud, Azure

- [IPVLAN L3](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/ipvlanl3.yaml)

        kubectl apply -f config/samples/multinicnetwork/ipvlanl3.yaml
        
### BareMetal
- [IPVLAN L2 with whereabout IPAM](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/ipvlanl2.yaml)

        kubectl apply -f config/samples/multinicnetwork/ipvlanl2.yaml

- [SR-IoV with Multi-NIC IPAM](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/sriov.yaml) ( from v1.2.0 )

        kubectl apply -f config/samples/multinicnetwork/sriov.yaml

- [Mellanox Host Device without IPAM](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/mellanox_hostdevice.yaml) ( from v1.2.0 )

        kubectl apply -f config/samples/multinicnetwork/mellanox_hostdevice.yaml

- [IPVLAN L2 with unmanaged HostInterface](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/ipvlanl2_unmanaged.yaml) ( from v1.2.1 )

        kubectl apply -f config/samples/multinicnetwork/ipvlanl2_unmanaged.yaml

### AWS
- [IPVLAN L2 with AWS-VPC-connecting IPAM](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/config/samples/multinicnetwork/awsipvlan.yaml) ( from v1.1.0 )

        kubectl apply -f config/samples/multinicnetwork/awsipvlan.yaml

## Connection Check
see [check connection](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/README.md#check-connections).