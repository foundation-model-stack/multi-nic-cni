# User Guide
## Requirements
- Secondary interfaces attached to worker nodes, check terraform script [here](https://github.com/foundation-model-stack/multi-nic-cni/tree/main/terraform).
- Multus CNI installation; compatible with networkAttachmentDefinition and pod annotation in multus-cni v3.8
- For IPVLAN L3 CNI, the following configurations are additionally required
    * enable allowing IP spoofing for each attached interface
    * set security group to allow IPs in the target container subnet
    * IPVLAN support (kernel version >= 4.2)
## Quick Installation
**by OperatorHub**

- Kubernetes with OLM:
    * check [multi-nic-cni-operator on OperatorHub.io](https://operatorhub.io/operator/multi-nic-cni-operator)
    ![](../img/k8s-operatorhub.png)
- Openshift Container Platform:
    * Search for `multi-nic-cni-operator` in OperatorHub
    ![](../img/openshift-operatorhub.png)

**by manifests with kubectl**

```bash
kubectl apply -f deploy/
```
**by bundle with operator-sdk**

```bash
operator-sdk run bundle ghcr.io/foundation-model-stack/multi-nic-cni-bundle:v1.0.5
```
## Deploy MultiNicNetwork resource

**MultiNicNetwork CR**

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
ipam|ipam plugin config| string | ipam can be single-NIC IPAM (e.g., whereabouts, VPC-native IPAM) or multi-NIC IPAM (e.g., [Multi-NIC IPAM Plugin](../concept/multi-nic-ipam.md#ipam-configuration))
multiNicIPAM| indicator of ipam type | bool | **true** if ipam returns multiple IPs from *masters* key of NetworkAttachmentDefinition config at once, **false** if ipam returns only single IP from static config in ipam block
plugin|main plugin config|[NetConf](https://pkg.go.dev/github.com/containernetworking/cni/pkg/types#NetConf) + plugin-specific arguments | main plugin integration must implement [Plugin](https://github.com/foundation-model-stack/multi-nic-cni/blob/main/plugin/plugin.go) with GetConfig function
attachPolicy|attachment policy|policy|[strategy](../concept/policy.md) with corresponding arguments to select host NICs to be master of secondary interfaces on Pod
namespaces|list of namespaces to apply the network definitions (i.e., to create NetworkAttachmentDefinition resource)|[]string|apply to all namespace if not specified. new item can be added to the list by `kubectl edit` to create new NetworkAttachmentDefinition. the created NetworkAttachmentDefinition must be deleted manually if needed.

1. Prepare `network.yaml` as shown in the [example](#multinicnetwork)
    
2. Deploy the network definition.

        kubectl apply -f network.yaml

    After deployment, the operator will create *NetworkAttachmentDefinition* of [Multus CNI](multus) from *MultiNicNetwork* as well as dependent resource such as *SriovNetworkNodePolicy*, *SriovNetwork* for sriov plugin.

3. Annotate the pod with the network name to attach additional interfaces. 

        metadata:
        annotations:
            k8s.v1.cni.cncf.io/networks: multi-nic-sample

## Check connections
1. Deploy concheck driver
        
        kubectl create -f connection-check/concheck.yaml

2. Check log

        kubectl logs job/multi-nic-concheck

     *expected output:*
     
         ###########################################
         ## Connection Check: multinic-ipvlanl3
         ###########################################
         FROM                           TO                              CONNECTED/TOTAL IPs                          BANDWIDTHs
         multi-nic-n7zf6-worker-2-dbjpg multi-nic-n7zf6-worker-2-zt5l5  2/2             [192.168.0.65 192.168.64.65] [ 6.10Gbits/sec 10.2Gbits/sec]
         multi-nic-n7zf6-worker-2-zt5l5 multi-nic-n7zf6-worker-2-dbjpg  2/2             [192.168.0.1 192.168.64.1]   [ 7.81Gbits/sec 12.4Gbits/sec]
         ###########################################

3. Clean up

        kubectl delete pod -n default --selector multi-nic-concheck
        kubectl delete job -n default --selector multi-nic-concheck
        kubectl delete -f connection-check/concheck.yaml

## Clean up
**installed by manifests with kubectl**
```
kubectl delete -f deploy/
```
**installed by bundle with operator-sdk**
```
operator-sdk cleanup multi-nic-cni-operator
```
