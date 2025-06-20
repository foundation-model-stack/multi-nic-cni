# OpenShift on Bare Metal Infrastructure Preparation

## Table of Contents
- [Steps to Install OpenShift with the Assisted Installer Web Console](#steps-to-install-openshift-with-the-assisted-installer-web-console)
- [Tenant Networking (NIC2) Configuration](#tenant-networking-nic2-configuration)
  - [Macvlan Configuration via NMState Operator](#macvlan-configuration-via-nmstate-operator)
  - [IPvlan Configuration via NMState Operator](#ipvlan-configuration-via-nmstate-operator)
- [Reference](#reference)

This guide describes how to install OpenShift using the Assisted Installer web console, which provides a user-friendly, browser-based approach for deploying OpenShift clusters.

## Steps to Install OpenShift with the Assisted Installer Web Console

1. **Access the Assisted Installer Web Console**
   - Go to the OpenShift Assisted Installer web console as described in the official documentation.

2. **Set Cluster Details**
   - Enter your cluster name, base domain, and other required details.

3. **Configure Networking and Hosts**
   - Configure static or dynamic networking as needed.
   - Add and configure hosts for your cluster.

4. **Upload Custom Manifests (Optional)**
   - In the web console, navigate to the custom manifests section.
   - Upload your YAML or JSON manifest files (such as network or MachineConfig customizations).
   - You can add, remove, or overwrite manifests before installation.

5. **Preinstallation Validations**
   - The installer will automatically validate your configuration and hosts before allowing installation to proceed.

6. **Begin Installation**
   - Once all validations pass, click **Begin installation** in the web console.
   - Monitor installation progress and host status directly from the UI.

7. **Post-Installation**
   - After installation, download the `kubeconfig` and credentials from the console.
   - Log in to the OpenShift web console and complete any postinstallation steps (such as configuring identity providers).

## Tenant Networking (NIC2) Configuration

In typical bare metal environments, **NIC1** is reserved for infrastructure networking, handling essential intra-cluster communication and management traffic. **NIC2**, on the other hand, is commonly designated for tenant networking. It serves as the primary interface for attaching additional networks (for example, using Multus), enabling advanced pod-level networking and supporting high-performance or isolated workloads.

### Macvlan Configuration via NMState Operator

The `configure-nic2-macvlan.yaml` file provides an example of configuring a macvlan network interface using the [NMState Operator](https://nmstate.io/). This configuration is useful for setting up advanced network topologies on bare metal nodes in OpenShift.

#### Key Features
- Defines a `NodeNetworkConfigurationPolicy` for a specific node (replace `<hostname>` with your node's hostname).
- Configures a bond interface (`tenant-bond`) in active-backup mode with static IPv4 addressing and custom MTU.
- Sets up two physical interfaces (`ens9f0np0` and `ens9f1np1`) with specific speed, ring buffer, and MAC address settings (MAC addresses are anonymized).

#### Usage
1. Ensure the NMState Operator is installed and running in your cluster.
2. Edit `configure-nic2-macvlan.yaml` and replace `<hostname>` with the actual hostname of your target node.
3. (Optional) Adjust interface names, IP addresses, or other parameters as needed for your environment.
4. Apply the configuration using `kubectl`:
   ```sh
   kubectl apply -f configure-nic2-macvlan.yaml
   ```

### IPvlan Configuration via NMState Operator

The `configure-nic2-ipvlan.yaml` file provides an example of configuring an ipvlan network interface using the NMState Operator. This configuration is useful for setting up advanced network topologies on bare metal nodes in OpenShift.

#### Key Features
- Defines a `NodeNetworkConfigurationPolicy` for a specific node (replace `<hostname>` with your node's hostname).
- Configures an ipvlan interface with static IPv4 addressing and custom MTU.
- Supports multiple parent interfaces and advanced network segmentation using VLANs (if needed).
- Allows for flexible attachment to physical interfaces and custom network policies.

#### Usage
1. Ensure the NMState Operator is installed and running in your cluster.
2. Edit `configure-nic2-ipvlan.yaml` and replace any placeholder values (such as `<hostname>`) with the actual values for your environment.
3. (Optional) Adjust interface names, IP addresses, or other parameters as needed for your environment.
4. Apply the configuration using `kubectl`:
   ```sh
   kubectl apply -f configure-nic2-ipvlan.yaml
   ```

## Reference
- For detailed instructions on installing OpenShift with the Assisted Installer, see: [Red Hat Assisted Installer for OpenShift - Installing with Web Console](https://docs.redhat.com/en/documentation/assisted_installer_for_openshift_container_platform/2025/html/installing_openshift_container_platform_with_the_assisted_installer/installing-with-ui)
- For more details on the NMState Operator and advanced network configuration, see: [NMState documentation](https://nmstate.io/)


