# Multi-NIC CNI Infrastructure Preparation on AWS

## Table of Contents
- [Introduction](#introduction)
- [AWS Infrastructure Setup with CloudFormation](#aws-infrastructure-setup-with-cloudformation)
- [Auto-scaling with Machine API Operator](#auto-scaling-with-machine-api-operator)
- [References](#references)

## Introduction
This guide describes how to prepare AWS infrastructure for Multi-NIC CNI using CloudFormation templates. The process includes setting up the VPC, EKS cluster, secondary subnets for Multi-NIC, and configuring nodegroups. It also covers considerations for auto-scaling with the Machine API Operator.

## AWS Infrastructure Setup with CloudFormation
The infrastructure setup is divided into three main steps, each with a corresponding CloudFormation template:

---

### 1. VPC-EKS infrastructure (`1_vpc-eks.yaml`)
  - create VPC
  - create 1 public subnet for VPC and its associated route table
  - create IGW, NATGW, Bastion Host
  - create EKS cluster
  - create 1 private subnet for EKS and its associated route table

**Usage:**
```sh
aws cloudformation create-stack \
  --stack-name my-vpc-eks \
  --template-body file://1_vpc-eks.yaml \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameters ParameterKey=VpcCidr,ParameterValue=10.0.0.0/16 \
               ParameterKey=AvailabilityZones,ParameterValue='["us-west-2a","us-west-2b"]' \
               ParameterKey=PublicSubnetAz1Cidr,ParameterValue=10.0.0.0/24 \
               ParameterKey=PublicSubnetAz2Cidr,ParameterValue=10.0.1.0/24 \
               ParameterKey=PrivateSubnetAz1Cidr,ParameterValue=10.0.2.0/24 \
               ParameterKey=PrivateSubnetAz2Cidr,ParameterValue=10.0.3.0/24
```
*Outputs from this step (VPC ID, subnet IDs, etc.) are required for the next steps.*

---

### 2. Multi-NIC secondary subnets (`2_multinic-subnet-sg.yaml`)
  > refer to created VPC id, and defined AZ and VPC cidr in 1.
  - create 2 subnets for the existing EKS cluster on specific AZ
  - create security group for secondary interfaces (using defined VPCCidr, and PodCidr)

**Usage:**
```sh
aws cloudformation create-stack \
  --stack-name multinic-subnet-sg \
  --template-body file://2_multinic-subnet-sg.yaml \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameters ParameterKey=VpcId,ParameterValue=<VPC_ID_FROM_STEP_1> \
               ParameterKey=VpcCidr,ParameterValue=10.0.0.0/16 \
               ParameterKey=AvailabilityZones,ParameterValue='["us-west-2a","us-west-2b"]' \
               ParameterKey=PodCidr,ParameterValue=192.168.0.0/16 \
               ParameterKey=MultusSubnet1Az1Cidr,ParameterValue=10.0.4.0/24 \
               ParameterKey=MultusSubnet1Az2Cidr,ParameterValue=10.0.5.0/24 \
               ParameterKey=MultusSubnet2Az1Cidr,ParameterValue=10.0.6.0/24 \
               ParameterKey=MultusSubnet2Az2Cidr,ParameterValue=10.0.7.0/24
```
*Use the VPC ID and AZs from step 1. Outputs (subnet IDs, security group) are needed for the next step.*

---

### 3. Nodegroup (`3_nodegroup.yaml`)
  > refer to EKS cluster name, VPC id, VPC cidr, private subnet created by 1. and subnets, security group created by 2.
  - add rules to node security group (including *NodeSecurityGroupDaemonPortIngress* for Multi-NIC CNI using defined DaemonPort)
  - put attaching the created private subnet and MultiNIC subnets as well as the security groups to lambda function as a primary interface and secondary interfaces respectively

**Usage:**
```sh
aws cloudformation create-stack \
  --stack-name multinic-nodegroup \
  --template-body file://3_nodegroup.yaml \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameters ParameterKey=ClusterName,ParameterValue=<EKS_CLUSTER_NAME> \
               ParameterKey=NodeGroupName,ParameterValue=multinic-nodegroup \
               ParameterKey=VpcId,ParameterValue=<VPC_ID_FROM_STEP_1> \
               ParameterKey=Subnets,ParameterValue='<PRIVATE_SUBNET_IDS_FROM_STEP_1>' \
               ParameterKey=MultiNICSubnets,ParameterValue='<MULTINIC_SUBNET_IDS_FROM_STEP_2>' \
               ParameterKey=MultiNICSecurityGroups,ParameterValue='<MULTINIC_SG_ID_FROM_STEP_2>' \
               ParameterKey=DaemonPort,ParameterValue=11000
```
*Replace placeholders with actual values from previous stack outputs.*

---

> [!TIP]
> - Deploy the stacks in order: `1_vpc-eks.yaml` → `2_multinic-subnet-sg.yaml` → `3_nodegroup.yaml`.
> - After each stack, note the outputs (IDs, subnets, security groups) for use in the next step.

## Secondary interface attachment via Machine API

To enable secondary interface attachment via the Machine API Operator, there is a need to build a custom machine-api-controller for AWS provider to support specifying secondary networks in [AWSMachineProviderConfig](https://pkg.go.dev/github.com/openshift/api/machine/v1beta1#AWSMachineProviderConfig).

To apply the custom controller, please check [Machine API Hacking Guide](https://github.com/openshift/machine-api-operator/blob/main/docs/dev/hacking-guide.md#machine-api---hacking-guide).

### Machine API Operator Resources
- [Launch instance function](https://github.com/openshift/machine-api-provider-aws/blob/main/pkg/actuators/machine/instances.go#L282)
- [Example machine manifest](https://github.com/openshift/machine-api-operator/blob/master/docs/examples/machine.yaml)
- [AWSMachineProviderConfig](https://github.com/openshift/api/blob/master/machine/v1beta1/types_awsprovider.go)
- [AWS parameters](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/parameters-section-structure.html)
- [awsmachine](https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/a87fc104303fab3a7ce0004f62126ff91da09096/api/v1beta1/awsmachine_types.go#L47)

## References
- [AWS CloudFormation Documentation](https://docs.aws.amazon.com/cloudformation/index.html)
- [Amazon EKS Documentation](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html)
- [Multi-NIC CNI Documentation](https://github.com/foundation-model-stack/multi-nic-cni)




