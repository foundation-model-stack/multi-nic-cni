# Multi-NIC CNI Infrastructure Preparation on IBM Cloud

## Table of Contents
- [Introduction](#introduction)
- [How to Install Terraform](#how-to-install-terraform)
- [IBM Cloud Infrastructure Setup with Terraform](#ibm-cloud-infrastructure-setup-with-terraform)
- [References](#references)

## Introduction
This guide describes how to prepare IBM Cloud infrastructure for Multi-NIC CNI using Terraform scripts. The process includes creating security groups, subnets, and attaching them to worker nodes, as well as configuring required rules for pod networking and the Multi-NIC CNI daemon port.

## How to Install Terraform

Follow these steps to install Terraform using the official method:

1. **Go to the Terraform Downloads Page:**  
   [https://www.terraform.io/downloads.html](https://www.terraform.io/downloads.html)

2. **Download the appropriate package** for your operating system (Windows, macOS, or Linux).

3. **Unzip the downloaded file** and move the `terraform` binary to a directory included in your system's `PATH`.
   - On Linux/macOS:
     ```bash
     unzip terraform_<VERSION>_<OS>_amd64.zip
     sudo mv terraform /usr/local/bin/
     ```
   - On Windows:
     - Unzip the file.
     - Move `terraform.exe` to a directory in your `PATH` (e.g., `C:\Windows\System32` or another directory you prefer).
     - You may need to add the directory to your system's PATH environment variable.

4. **Verify the installation:**
   ```bash
   terraform version
   ```

For more details, see the [official Terraform installation guide](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli).

## IBM Cloud Infrastructure Setup with Terraform

The infrastructure setup is divided into the following steps, using the provided Terraform scripts:

---

### 1. Install Terraform

- Install Terraform version `>= 0.1.3`.

---

### 2. Prepare and Configure Variables

- Copy the template and edit your configuration:
  ```bash
  cp terraform.tfvars.template terraform.tfvars
  ```
- Edit `terraform.tfvars` to set your IBM Cloud API key, VPC name, resource group, region, zone, number of subnets, and the main interface security group name.

---

### 3. Initialize Terraform

- Initialize the working directory:
  ```bash
  terraform init
  ```

---

### 4. Apply Terraform to Create Resources

- **(Recommended)** First, create only the subnets:
  ```bash
  terraform apply -var-file=terraform.tfvars -target="ibm_is_subnet.subnets"
  ```
- Then, apply all resources:
  ```bash
  terraform apply -var-file=terraform.tfvars
  ```

> **Why are these steps separated?**
> - Creating subnets first ensures that their IDs and network details are available for any subsequent resources that reference them (such as network interfaces, security group rules, or VM attachments).
> - This reduces the risk of dependency errors or race conditions, making the provisioning process more reliable—especially in cloud environments where resource creation order matters.
> - After the subnets exist, all other resources that depend on them can be safely created with the full apply.

---

**What the Terraform scripts do:**
- Create a security group with inbound/outbound rules to allow internal security group communication and pod subnet (`podnet`, default: 192.168.0.0/16).
- Add a daemon port (default: `11000`) rule to the main interface security group.
- Create the specified number of subnets in the chosen zone.
- Attach the created subnets to the worker nodes (VSIs) listed in your configuration.

## Secondary interface attachment via Machine API

To enable secondary interface attachment via the Machine API Operator, there is a need to build a custom machine-api-controller for IBM Cloud provider to support specifying secondary networks in [IBMCloudMachineProviderSpec](https://pkg.go.dev/github.com/openshift/machine-api-provider-ibmcloud/pkg/apis/ibmcloudprovider/v1#IBMCloudMachineProviderSpec).

To apply the custom controller, please check [Machine API Hacking Guide](https://github.com/openshift/machine-api-operator/blob/main/docs/dev/hacking-guide.md#machine-api---hacking-guide).

---

## References
- [Terraform Installation Guide](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli)
- [IBM Cloud Terraform Provider Documentation](https://registry.terraform.io/providers/IBM-Cloud/ibm/latest/docs)
- [Multi-NIC CNI Documentation](https://github.com/foundation-model-stack/multi-nic-cni)# Multi-NIC CNI Infrastructure Preparation on IBM Cloud

