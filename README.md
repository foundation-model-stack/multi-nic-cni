> Documents and source codes for the deprecated domain `cogadvisor.io` are moved to [cogadvisor-net branch](https://github.com/foundation-model-stack/multi-nic-cni/tree/cogadvisor-net)
> 
- [Multi-NIC CNI](#multi-nic-cni)
  - [MultiNicNetwork](#multinicnetwork)
  - [Usage](#usage)
      - [Requirements](#requirements)
      - [Quick Installation](#quick-installation)
        - [by manifests with kubectl](#by-manifests-with-kubectl)
        - [by bundle with operator-sdk](#by-bundle-with-operator-sdk)
      - [Deploy MultiNicNetwork resource](#deploy-multinicnetwork-resource)
      - [Check connections](#check-connections)
        - [installed by bundle with operator-sdk](#installed-by-bundle-with-operator-sdk)
# Multi-NIC CNI
Attaching secondary network interfaces that is linked to different network interfaces on host (NIC) to pod provides benefits of network segmentation and top-up network bandwidth in the containerization system. 

Multi-NIC CNI is the CNI plugin operating on top of [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). However, unlike Multus, instead of defining and handling each secondary network interface one by one, this CNI automatically discovers all available secondary interfaces and handles them as a NIC pool.
With this manner, it can provide the following benefits.

i) **Common secondary network definition**: User can manage only one network definition for multiple secondary interfaces with a common CNI main plugin such as ipvlan, macvlan, and sr-iov. 

ii) **Common NAT-bypassing network solution**: All secondary NICs on each host can be assigned with non-conflict CIDR and non-conflict L3 routing configuration that can omit an overlay networking overhead. Particularyly, the CNI is built-in with L3 IPVLAN solution composing of the following functionalities.
  1) **Interface-host-devision CIDR Computation**: compute allocating CIDR range for each host and each interface from a single global subnet with the number of bits for hosts and for interface. 
  2) **L3 Host Route Configuration**: configure L3 routes (next hop via dev) in host route table according to the computed CIDR.
  3) **Distributed IP Allocation Management**: manage IP allocation/deallocation distributedly via the communication between CNI program and daemon at each host.

[read more](./document/multi-nic-ipam.md) 

iii) **Policy-based secondary network attachment**: Instead of statically set the desired host's master interface name one by one, user can define a policy on attaching multiple secondary network interfaces such as specifying only the number of desired interfaces, filtering only highspeed NICs. 

[read more](./document/policy.md)

![](./document/img/commonstack.png)

The Multi-NIC CNI architecture can be found [here](./document/architecture.md).

## MultiNicNetwork
The Multi-NIC operator operates over a custom resource named *MultiNicNetwork* defined by users.
This definition will define a Pod global subnet, common network definition (main CNI and IPAM plugin), and attachment policy. 
After deploying *MultiNicNetwork*, *NetworkAttachmentDefinition* with the same name will be automatically configured and created respectively.

```yaml
# network.yaml
apiVersion: multinic.fms.io/v1
kind: MultiNicNetwork
metadata:
  name: multi-nic-sample
spec:
  subnet: "192.168.0.0/16"
  ipam: |
    {
      "type": "multi-nic-ipam",
      "hostBlock": 6, 
      "interfaceBlock": 2,
      "vlanMode": "l3"
    }
  multiNICIPAM: true
  plugin:
    cniVersion: "0.3.0"
    type: ipvlan
    args: 
      mode: l3
  attachPolicy:
    strategy: none
  namespaces:
  - default
```

Argument|Description|Value|Remarks
---|---|---|---
subnet|cluster-wide subnet for all hosts and pods|CIDR range|currently support only v4
hostBlock|number of address bits for host indexing| int (n) | the number of assignable host = 2^n
ipam|ipam plugin config| string | ipam can be single-NIC IPAM (e.g., whereabouts, VPC-native IPAM) or multi-NIC IPAM (e.g., [Multi-NIC IPAM Plugin](document/multi-nic-ipam.md#ipam-configuration))
multiNicIPAM| indicator of ipam type | bool | **true** if ipam returns multiple IPs from *masters* key of NetworkAttachmentDefinition config at once, **false** if ipam returns only single IP from static config in ipam block
plugin|main plugin config|[NetConf](https://pkg.go.dev/github.com/containernetworking/cni/pkg/types#NetConf) + plugin-specific arguments | main plugin integration must implement [Plugin](./plugin/plugin.go) with GetConfig function
attachPolicy|attachment policy|policy|[strategy](document/policy.md) with corresponding arguments to select host NICs to be master of secondary interfaces on Pod
namespaces|list of namespaces to apply the network definitions (i.e., to create NetworkAttachmentDefinition resource)|[]string|apply to all namespace if not specified. new item can be added to the list by `kubectl edit` to create new NetworkAttachmentDefinition. the created NetworkAttachmentDefinition must be deleted manually if needed.


## Usage
#### Requirements
- Secondary interfaces attached to worker nodes, check terraform script [here](./terraform/)
- Multus CNI installation; compatible with networkAttachmentDefinition and pod annotation in multus-cni v3.8
- For IPVLAN L3 CNI, the following configurations are additionally required
  - enable allowing IP spoofing for each attached interface
  - set security group to allow IPs in the target container subnet
  - IPVLAN support (kernel version >= 4.2)
#### Quick Installation
For **Openshift**, assign privileged security context to multi-nic-cni-operator-controller-manager service account first
```bash
oc adm policy add-scc-to-user privileged system:serviceaccount:multi-nic-cni-operator-system:multi-nic-cni-operator-controller-manager
```
##### by manifests with kubectl
  ```bash
  kubectl apply -f deploy/
  ```
##### by bundle with operator-sdk
  ```bash
  operator-sdk run bundle ghcr.io/foundation-model-stack/multi-nic-cni-bundle:v1.0.2
  kubectl apply -f deploy/1_config.yaml
  ```
#### Deploy MultiNicNetwork resource
1. Prepare `network.yaml` as shown in the [example](#multinicnetwork)
    
2. Deploy 
   ```bash
   kubectl apply -f network.yaml
   ```
   After deployment, the operator will create *NetworkAttachmentDefinition* of [Multus CNI](multus) from *MultiNicNetwork* as well as dependent resource such as *SriovNetworkNodePolicy*, *SriovNetwork* for sriov plugin.
3. To attach additional interfaces, annotate the pod with the network name
    ```yaml
    metadata:
      annotations:
        k8s.v1.cni.cncf.io/networks: multi-nic-sample
    ```

#### Check connections
1. Deploy concheck driver
    ```bash
    kubectl create -f connection-check/concheck.yaml
    ```
2. Check log
   ```bash
    kubectl logs job/multi-nic-concheck
    ```
    expected log:
    ```bash
      ###########################################
      ## Connection Check: multinic-ipvlanl3
      ###########################################
      FROM                           TO                              CONNECTED/TOTAL IPs                          BANDWIDTHs
      multi-nic-n7zf6-worker-2-dbjpg multi-nic-n7zf6-worker-2-zt5l5  2/2             [192.168.0.65 192.168.64.65] [ 6.10Gbits/sec 10.2Gbits/sec]
      multi-nic-n7zf6-worker-2-zt5l5 multi-nic-n7zf6-worker-2-dbjpg  2/2             [192.168.0.1 192.168.64.1]   [ 7.81Gbits/sec 12.4Gbits/sec]
      ###########################################
    ```
3. Clean up
   ```bash
    kubectl delete pod -n default --selector multi-nic-concheck
    kubectl delete job -n default --selector multi-nic-concheck
    kubectl delete -f connection-check/concheck.yaml
    ```

#### Clean up
##### installed by manifests with kubectl
  ```bash
  kubectl delete -f deploy/
  ```
##### installed by bundle with operator-sdk
  ```bash
  operator-sdk cleanup multi-nic-cni-operator
  ```

