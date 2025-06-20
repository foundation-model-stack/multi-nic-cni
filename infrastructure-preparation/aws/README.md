# Enabling Multi-NIC CNI on IBM Cloud
## Infrastructure by AWS CloudFormation
The provided template is modified from [AWS Cloud Formation Template for Multus](https://github.com/aws-samples/eks-install-guide-for-multus/blob/main/cfn/templates/infra/README.md). There are three templates as follows:
- 1. VPC-EKS infrastructure
  - create VPC
  - create 1 public subnet for VPC and its associated route table 
  - create IGW, NATGW, Bastion Host
  - create EKS cluster
  - create 1 private subnet for EKS and its associated route table
- 2. Multi-NIC secondary subnets 
    > refer to created VPC id, and defined AZ and VPC cidr in 1.
  - create 2 subnets for the existing EKS cluster on specific AZ
  - create security group for secondary interfaces (using defined VPCCidr, and PodCidr)
- 3. Nodegroup 
    >refer to EKS cluster name, VPC id, VPC cidr, private subnet created by 1. and subnets, security group created by 2.
  - add rules to node security group (including *NodeSecurityGroupDaemonPortIngress* for Multi-NIC CNI using defined DaemonPort)
  - put attaching the created private subnet and MultiNIC subnets as well as the security groups to lampda function as a primary interface and secondary interfaces respectively.

## Auto-scaling by Machine API Operator
To attach secondary interfaces with machine API operator, there is still a need to modify AWSMachineProviderConfig and update machine-api-controller for AWS provider to support.

references:
- [Launch instance function](https://github.com/openshift/machine-api-provider-aws/blob/main/pkg/actuators/machine/instances.go#L282)
- [Example machine manifest](https://github.com/openshift/machine-api-operator/blob/master/docs/examples/machine.yaml)
- [AWSMachineProviderConfig](https://github.com/openshift/api/blob/master/machine/v1beta1/types_awsprovider.go)
- [AWS parameters](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/parameters-section-structure.html)
- [awsmachine](https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/a87fc104303fab3a7ce0004f62126ff91da09096/api/v1beta1/awsmachine_types.go#L47)



