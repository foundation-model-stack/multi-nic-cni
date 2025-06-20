# Multi-NIC CNI Infrastructure Support

## Table of Contents
- [AWS Infrastructure](#aws-infrastructure)
- [IBM Cloud Infrastructure](#ibm-cloud-infrastructure)
- [Bare Metal Infrastructure](#bare-metal-infrastructure)

Multi-NIC CNI supports three different infrastructure configurations, each with its own specific features and requirements.

## AWS Infrastructure

### Overview
- Based on modified AWS CloudFormation templates for Multus
- Supports EKS (Elastic Kubernetes Service) clusters
- Enables multiple network interfaces through secondary subnets
- Integrates with Machine API Operator for auto-scaling

### Key Components
- VPC-EKS infrastructure
- Multi-NIC secondary subnets
- Nodegroup configuration
- Security group management

### Features
- VPC and EKS cluster setup
- Secondary subnet creation and management
- Security group configuration
- Machine API Operator integration
- Auto-scaling support

### Requirements
- AWS CloudFormation templates
- EKS cluster
- VPC configuration
- Machine API Operator
- Appropriate IAM permissions

### Preparation
For detailed instructions on preparing the AWS infrastructure, please refer to [this guideline](./aws)

## IBM Cloud Infrastructure

### Overview
- Implemented using Terraform
- Supports IBM Cloud VPC infrastructure
- Enables multiple network interfaces through dynamic subnet creation

### Key Components
- Security group management with specific rules for:
  - Internal security group communication
  - Pod subnet (default: 192.168.0.0/16)
  - Daemon port (default: 11000)
- Dynamic subnet creation and attachment to VSIs
- VPC integration
- Resource group support

### Features
- Terraform-based infrastructure management
- Security group configuration with specific rules
- Dynamic subnet creation and management
- VPC and resource group integration
- Pod network support

### Requirements
- Terraform (version >= 0.1.3)
- IBM Cloud VPC infrastructure
- Resource group configuration
- VPC and zone configuration
- Worker node access

### Preparation
For detailed instructions on preparing the IBM Cloud infrastructure, please refer to [this guideline](./ibmcloud)

## Bare Metal Infrastructure

### Overview
- Supports physical network interfaces on bare metal servers
- Tested and supported on Red Hat OpenShift platform only
- Enables pods to use multiple physical network interfaces, including VLAN interfaces
- Ideal for high-performance computing and network-intensive workloads

### Key Components
- Physical network interface support
- VLAN interface support
- Multiple network interface types
- Direct hardware access
- Custom network configuration
- OpenShift integration

### Features
- Direct access to physical and VLAN network interfaces
- Support for various network interface types
- High-performance networking capabilities
- Custom network interface configuration

### Requirements
- Red Hat OpenShift platform
- Physical network interfaces
- Appropriate driver support
- Network interface configuration
- Hardware compatibility

### Preparation
For detailed instructions on preparing the bare metal infrastructure, please refer to [this guideline](./bare-metal)



