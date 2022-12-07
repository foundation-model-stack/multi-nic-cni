# Troubleshooting 
Before start troubleshooting, set common variables for reference simplicity.
```bash
export FAILED_POD= # pod that fails to run
export FAILED_POD_NAMESPACE= # namespace where the failed pod is supposed to run
export FAILED_NODE= # node where pod is deployed
export FAILED_NODE_IP = # IP of FAILED_NODE
export MULTI_NIC_NAMESPACE= # namespace where multi-nic cni operator is deployed, default=openshift-operators
```
## Issues
### Pod failed to start
Pod stays pending in `ContainerCreating` status.
Get more information from `describe`
```bash
kubectl describe $FAILED_POD -n $FAILED_POD_NAMESPACE
```
- Error: `FailedCreatePodSandBox`
    * [CNI binary not found](#cni-binary-not-exist)
    * [IPAM ExecAdd failed](#ipam-execadd-failed)

##### CNI binary not exist
The binary file of CNI is not in the expected location read by Multus. The expected location can be found in Multus daemonset as below.

```bash
kubectl get ds $(kubectl get ds -A\
|grep multus|head -n 1|awk '{printf "%s -n %s", $2, $1}')  -ojson\
|jq .spec.template.spec.volumes
```

*Example output:*

```json
[
...
  {
    "hostPath": {
      "path": "/var/lib/cni/bin",
      "type": ""
    },
    "name": "cnibin"
  },
...
]
```

The expected location is in *hostPath* of *cnibin*.

- **missing multi-nic/multi-nic-ipam CNI**

  The CNI directory is probably mounted to a wrong location in the configs.multinic.fms.io CR. Modify mount path ( *hostpath* attribute ) in `spec.daemon.mounts` of *cnibin* to the target location above.

    kubectl edit config.multinic multi-nicd -n $MULTI_NIC_NAMESPACE

- **missing other CNI such as ipvlan**
  The missing CNI may not be supported. 

##### IPAM ExecAdd failed
- `failed to load netconf`
   
    The configuration cannot be loaded. This is delegated CNI (such as IPVLAN) issue. 
    Find more details from [CNI log](#get-cni-log).

- `"netx": address already in use`

    There are a couple of reasons to cause this issue such as IPPool is unsync due to unexpected removal (from operator reinstalltion) or modification of IPPool resource when some assigned pods are still running. IP Address is previously assigned to other pods. 
   
    This should be handled by [this commit](https://github.com/foundation-model-stack/multi-nic-cni/commit/4bbac4b8e8b6975b4c49f660df78dc9506e34a49). This commit will try assigning the next available address to prevent infinite failure assignment to the same already-in-use IP address.
    Try [updating to latest image of daemon](#update-daemon-pod-to-use-latest-version).

- `failed to request ip Response nothing` 
    - get more information from HostInterface CR: 

            kubectl get HostInterface $FAILED_NODE -oyaml

    - If hostinterfaces.multinic.fms.io "FAILED_NODE" not found, check [HostInterface is not be created.](#hostinterface-not-created) 
    - If no interfaces in `.spec.interfaces`, check [HostInterface does not show the secondary interfaces.](#no-secondary-interfaces-in-hostinterface)
    - Check whether it reaches CIDR block limit, confirm [no available IP address](#no-secondary-interfaces-in-hostinterface)
    - Other cases, find more details from [multi-nicd log](#get-multi-nicd-log)
- other CNI plugin (such as aws-vpc-cni, sr-iov) failure, check each CNI log.
    - aws-vpc-cni: `/host/var/log/aws-routed-eni`


###### HostInterface not created
There are a couple of reasons that the HostInterface is not created. First check the multi-nicd DaemonSet.
```bash
kubectl get ds multi-nicd -n $MULTI_NIC_NAMESPACE -oyaml
```
- *daemonsets.apps "multi-nicd" not found*
      - Check whether no config.multinic.fms.io deployed in the cluster.

            kubectl get config.multinic multi-nicd -n $MULTI_NIC_NAMESPACE

        If no config.multinic.fms.io deployed, see [Deploy multi-nicd](#deploy-multi-nicd-config)

      - The node has taint that the daemon is not tolerate. 

            kubectl get nodes $FAILED_NODE -o json|jq -r .spec.taints

        To tolerate the [taint]((https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)), add the tolerate manually to the multi-nicd DaemonSet.

            kubectl edit $(kubectl get po -owide -A|grep multi-nicd\
                |grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')

- Other cases, check [controller log](#get-controller-log)
###### No secondary interfaces in HostInterface

The HostInterface is created but there is no interface listed in the custom resource.

Check whether the controller can communicate with multi-nicd:

```bash
kubectl logs --selector control-plane=controller-manager \
  -n $MULTI_NIC_NAMESPACE -c manager| grep Join| grep $FAILED_NODE_IP
```

- If no line shown up and the full [controller log](#get-controller-log) keep printing `Fail to create hostinterface ... cannot update interfaces: Get "<node IP>/interface": dial tcp <node IP>:11000: i/o timeout`, check [set required security group rules](#set-security-groups)
- Other cases, [check interfaces at node's host network](#check-host-secondary-interfaces)

###### No available IP address
List corresponding Pod CIDR from HostInterface.
```bash
kubectl get HostInterface $FAILED_NODE -oyaml
```
Check ippools.multinic.fms.io of the corresponding pod CIDR whether the IP address actually reach the limit. If yes, consider changing the host block and interface block in `multinicnetworks.multinic.fms.io`.

### Ping failed
Check route status in multinicnetworks.multinic.fms.io.
```bash
kubectl get multinicnetwork.multinic.fms.io multinic-ipvlanl3 -o json\ 
| jq -r .status.routeStatus
```

- *WaitForRoutes*:  the new cidr is just recomputed and waiting for route update.
- *Failed*: some route cannot be applied, need attention. Check [multi-nicd log](#get-multi-nicd-log)
- *Unknown*: some daemon cannot be connected. 
- *N/A*: there is no L3 configuration applied. Check whether multinicnetwork.multinic.fms.io is defined with L3 mode and cidrs.multinic.fms.io is created. 

      kubectl get cidrs.multinic.fms.io

- *Success*: check [set required security group rules](#set-security-groups)
### TCP/UDP communication failed.
Check whether the multi-nicd detects the other host interfaces.
```bash
kubectl get po $(kubectl get po -owide -A|grep multi-nicd\
   |grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}') -o json\
   |jq -r .metadata.labels
```

The nubmer in `multi-nicd-join` should be equal to accumulated number of interfaces from each host in the same zone. 

[Check whether the host secondary interfaces between hosts are connected](#check-host-secondary-interfaces). 
If yes, try [restarting multi-nic-cni controller node](#restart-controller) to forcefully synchronize host interfaces.

## Actions

### Get CNI log
```bash
kubectl exec $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')\
-- cat /host/var/log/multi-nic-cni.log
```
### Get Controller log
```bash
kubectl logs --selector control-plane=controller-manager \
-n $MULTI_NIC_NAMESPACE -c manager
```
### Get multi-nicd log
```bash
kubectl logs $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')
```
### Deploy multi-nicd config
Restart the controller pod should create the multi-nicd config automatically.
```bash
kubectl delete po --selector control-plane=controller-manager \
-n $MULTI_NIC_NAMESPACE
```
If not, update the controller to the latest image and [restart the controller](#restart-multi-nic-cni-controller) (recommended).
Otherwise, deploy config manually.
```bash
kubectl create -f https://raw.githubusercontent.com/foundation-model-stack/multi-nic-cni/main/config/samples/config.yaml \
-n $MULTI_NIC_NAMESPACE
```
### Set security groups
There are four security group rules that must be opened for Multi-nic CNI.

1. outbound/inbound communication within the same security group
2. outbound/inbound communication of Pod networks
3. inbound multi-nicd serving TCP port (default: 11000)

### Add secondary interfaces
- Prepare secondary subnets with [required security group rules](#set-required-security-group-rules) and enable multiple source IPs from a single vNIC (i.e.g, enable IP spoofing on IBM Cloud) 
- Attach the secondary subnets to instance
      - manual attachment: follows Cloud provider instruction
      - by machine-api-operator: updates an image of machine api controller of the provider to support secondary interface on provider spec. 

        Check [example commit](https://github.com/openshift/machine-api-provider-ibmcloud/compare/main...sunya-ch:machine-api-provider-ibmcloud:multi-nic) in [modified controller](https://github.com/sunya-ch/machine-api-provider-ibmcloud/tree/multi-nic)

### Restart controller
```bash
kubectl delete --selector control-plane=controller-manager \
-n $MULTI_NIC_NAMESPACE
```

### Check host secondary interfaces 
Log in to FAILED_NODE with `oc debug node/$FAILED_NODE` or using [nettools](https://github.com/jedrecord/nettools) with `hostNetwork: true`. If secondary interfaces do not exist at the host network, [add the secondary interfaces](#add-secondary-interfaces-at-nodes-host-network)

### Update daemon pod to use latest version
1. Check whether the using `image` set with `latest` version tag and `imagePullPolicy: Always`

        kubectl get daemonset multi-nicd -o yaml -n $MULTI_NIC_NAMESPACE|grep image

    If not, modify `image` with the latest version tag and change `imagePullPolicy` to `Always`

        kubectl edit daemonset multi-nicd -n $MULTI_NIC_NAMESPACE

2. Delete the current multi-nicd pods with selector

        kubectl delete po --selector app=multi-nicd -n $MULTI_NIC_NAMESPACE

3. Check readiness  

        kubectl get po --selector app=multi-nicd -n $MULTI_NIC_NAMESPACE

### Update controller to use latest version
1. Check whether the using `image` set with `latest` version tag and `imagePullPolicy: Always`

        kubectl get deploy multi-nic-cni-operator-controller-manager -o yaml -n $MULTI_NIC_NAMESPACE|grep multi-nic-cni-controller -A 2|grep image

    If not, modify `image` with the latest version tag and change `imagePullPolicy` to `Always`

        kubectl edit deploy multi-nic-cni-operator-controller-manager -n $MULTI_NIC_NAMESPACE

2. Delete the current multi-nicd pods with selector

        kubectl delete po --selector control-plane=controller-manager -n $MULTI_NIC_NAMESPACE

3. Check readiness  

        kubectl get po --selector control-plane=controller-manager -n $MULTI_NIC_NAMESPACE

### Safe upgrade Multi-NIC CNI operator
- Before bundle version on Operator Hub to v1.0.2

      There are three significant changes:

       - Change API group from `net.cogadvisor.io` to `multinic.fms.io`. To check API group,

            kubectl get crd|grep multinicnetworks
            multinicnetworks.multinic.fms.io                                  2022-09-27T08:47:35Z

       - Change route configuration logic for handling fault tolerance issue. To check route configuration logic. Run `ip rule` in any worker host by running `oc debug node` or using [nettools](https://github.com/jedrecord/nettools) with `hostNetwork: true`. 

            > ip rule
            0:	from all lookup local
            32765:	from 192.168.0.0/16 lookup multinic-ipvlanl3
            32766:	from all lookup main
            32767:	from all lookup default
        
         If it shows similar rules as above, the route configuration logic is up-to-date.
        
      - Add multinicnetwork CR to show routeStatus. To check routeStatus key in multinicnetwork CR

            kubectl get multinicnetwork -o yaml|grep routeStatus
              routeStatus: Success

     If all changes are applied (up-to-date) in your current version, there is no need to stop the running workload to reinstall the operator. Check [update the daemon pods](#update-daemon-pod-to-use-latest-version) and [update the controller](#update-controller-to-use-latest-version) to get the image with latest minor updates and bug fixes.
    <br>

    Otherwise, check [live migration](https://github.com/foundation-model-stack/multi-nic-cni/tree/doc/live-migration)
