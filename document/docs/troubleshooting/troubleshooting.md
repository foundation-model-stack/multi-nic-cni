# Manual Troubleshooting (Common Issues)

** Please first confirm feature supports on each multi-nic-cni release version from [here](../release/index.md). **

<!-- TOC tocDepth:2..3 chapterDepth:3..6 -->

- [Issues](#issues)
    - [Multi-NIC CNI Controller gets OOMKilled](#multi-nic-cni-controller-gets-oomkilled)
    - [HostInterface not created](#hostinterface-not-created)
    - [No secondary interfaces in HostInterface](#no-secondary-interfaces-in-hostinterface)
    - [Pod failed to start](#pod-failed-to-start)
    - [Pod failed to start (Summary Table)](#pod-failed-to-start-summary-table)
    - [Ping failed](#ping-failed)
    - [TCP/UDP communication failed.](#tcpudp-communication-failed)
- [Actions](#actions)
    - [Controller configuration](#controller-configuration)
    - [Daemon configuration](#daemon-configuration)
    - [List in-use pods](#list-in-use-pods)
    - [Get CNI log (available after v1.0.3)](#get-cni-log-available-after-v103)
    - [Get Controller log](#get-controller-log)
    - [Get multi-nicd log](#get-multi-nicd-log)
    - [Deploy multi-nicd config](#deploy-multi-nicd-config)
    - [Set security groups](#set-security-groups)
    - [Add secondary interfaces](#add-secondary-interfaces)
    - [Restart controller](#restart-controller)
    - [Restart multi-nicd](#restart-multi-nicd)
    - [Check host secondary interfaces](#check-host-secondary-interfaces)
    - [Update daemon pod to use latest version](#update-daemon-pod-to-use-latest-version)
    - [Update controller to use latest version](#update-controller-to-use-latest-version)
    - [Safe upgrade Multi-NIC CNI operator](#safe-upgrade-multi-nic-cni-operator)
    - [Customize Multi-NIC CNI controller of operator](#customize-multi-nic-cni-controller-of-operator)

<!-- /TOC -->

## Issues
There are commonly three steps of issue: at pod creation, simple ICMP (ping) communication, TCP/UDP communication. The most complicated one is at pod creation. 

Before start troubleshooting, set common variables for reference simplicity.
```bash
export FAILED_POD= # pod that fails to run
export FAILED_POD_NAMESPACE= # namespace where the failed pod is supposed to run
export FAILED_NODE= # node where pod is deployed
export FAILED_NODE_IP = # IP of FAILED_NODE
export MULTI_NIC_NAMESPACE= # namespace where multi-nic cni operator is deployed, default=multi-nic-cni-operator
```

### Multi-NIC CNI Controller gets OOMKilled

This is expected issue in a large cluster where the controller requires large amount of member to operate. Please adjust the resource limit in the controller deployment. For the case of installing via operator hub or operator bundle, please check the step to modify the deployment in [Customize Multi-NIC CNI controller of operator](#customize-multi-nic-cni-controller-of-operator).


### HostInterface not created
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

        To tolerate the [taint](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration), add the tolerate manually to the multi-nicd DaemonSet.

            kubectl edit $(kubectl get po -owide -A|grep multi-nicd\
                |grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')

- Other cases, check [controller log](#get-controller-log)

### No secondary interfaces in HostInterface

The HostInterface is created but there is no interface listed in the custom resource.
There are two common root causes.

1. Communication between controller and multi-nicd is blocked. 
      
      * Check whether the controller can communicate with multi-nicd:

            kubectl logs --selector control-plane=controller-manager \
              -n $MULTI_NIC_NAMESPACE -c manager| grep Join| grep $FAILED_NODE_IP

        - If no line shown up and the full [controller log](#get-controller-log) printing `Fail to create hostinterface ... cannot update interfaces: Get "<node IP>/interface": dial tcp <node IP>:11000: i/o timeout`, check [set required security group rules](#set-security-groups)

2. Network interfaces are not configured as expected.
      * Check [multi-nicd log](#get-multi-nicd-log).
    
        - If getting `cannot list address on <SECONDARY INTERFACE>`, please confirm whether IPv4 address on the host. 
        - Otherwise, please refer to [check interfaces at node's host network](#check-host-secondary-interfaces).

### Pod failed to start

**Issue:**
Pod stays pending in `ContainerCreating` status.
Get more information from `describe`
```bash
kubectl describe $FAILED_POD -n $FAILED_POD_NAMESPACE
```

Find the following keyword from `FailedCreatePodSandBox`:

* [Network not found](#network-not-found)
* [CNI binary not found](#cni-binary-not-found)
* [IPAM ExecAdd: failed](#ipam-execadd-failed)
* [No available IP address](#no-available-ip-address)
* [IPAM plugin returned missing IP config](#ipam-plugin-returned-missing-ip-config)
* [zero config](#zero-config)

#### Pod failed to start (Summary Table)
For those who are familar to action command (e.g., list multinic CRs, list daemon pods), you may troubleshoot with the summary table:

> - Investigate source of issue from top to bottom
> - *X* refers to no relevance
> - If the issue cannot be solved by configuration (multinicnetwork, annotation, host network, config.multinic) and last patch of [controller](#update-daemon-pod-to-use-latest-version) and [multi-nicd](#update-daemon-pod-to-use-latest-version), please report the [issue](https://github.com/foundation-model-stack/multi-nic-cni/issues) with the corresponding log. 
> - *The solved bug on CNI binary requires node restart.

Potential source of Issue|Network not found|CNI binary not found|- IPAM ExecAdd: failed <br>- IPAM plugin returned missing IP config|zero config|Fail execPlugin
---|---|---|---|---|---
**multinicnetwork definition/annotation**|- annotation missing/mismatch<br>- multinicnetwork wrong configured|X|- IPAM wrong configured<br>- `masters` multinicnetwork spec missing (> 1 multinicnetwork)|**non-IP host:**<br>- no master name provided via multi-config or annotation|X
**host network**|X|X|X|**L3:**<br>- daemon communication blocked<br>**All:**<br>- interface missing<br>|X
**controller**|- net-attach-def not created|- daemon not created due to wrong configured (config.multinic)|**L3:**<br>- daemon/hostinterface not created<br>- CIDR/IPPool not created/unsynced|X|X
**daemon**<br>(multi-nicd)|X|X|**L3:**<br>- failed to discover hostinterface<br>- IP limit reach<br>**All cases:**<br>- hang on no-respond API server (should be fixed by [#172](https://github.com/foundation-model-stack/multi-nic-cni/pull/172))|X|X
**main CNI binary**<br>(multi-nic)|X|X|- *failed to clean up previous pod network (should be fixed by [#165](https://github.com/foundation-model-stack/multi-nic-cni/pull/165))|**host-device**<br>- *failed to clean up previous pod network (should be fixed by [#152](https://github.com/foundation-model-stack/multi-nic-cni/issues/152))|X
**ipam CNI binary**<br>(multi-nic-ipam)|X|X|- *failed to clean up previous ip allocation (should be fixed by [#104](https://github.com/foundation-model-stack/multi-nic-cni/pull/104))|X|X
**3rd-party CNI binary**|X|- binary missing|- 3rd-party IPAM failure|X|- 3rd-party main plugin failure


#### Network not found

```bash
kubectl get multinicnetwork # multinicnetwork resource created
kubectl get $FAILED_POD -n $FAILED_POD_NAMESPACE -oyaml|grep "k8s.v1.cni.cncf.io/networks" # pod annotation matched
kubectl get net-attach-def # network-attachment-definition created
```

If net-attach-def is missing (`No resources found in default namespace`), check [controller log](#get-controller-log) to see whether the failure comes from misconfiguration in multinicnetwork (Marshal failure) or network-attachment-definition creation request to API server.

#### CNI binary not found
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

#### IPAM ExecAdd: failed
This error occurs when CNI cannot execute Multi-NIC IPAM which can be caused by multiple reasons as follows.

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
    - Multi-nicd daemon pod has no response, [restart multi-nicd](#restart-multi-nicd) might help.

- other CNI plugin (such as aws-vpc-cni, sr-iov) failure, check each CNI log.
    - aws-vpc-cni: `/host/var/log/aws-routed-eni`

###### No available IP address
List corresponding Pod CIDR from HostInterface.
```bash
kubectl get HostInterface $FAILED_NODE -oyaml
```
Check ippools.multinic.fms.io of the corresponding pod CIDR whether the IP address actually reach the limit. If yes, consider changing the host block and interface block in `multinicnetworks.multinic.fms.io`.

#### IPAM plugin returned missing IP config

No IP address set from the multi-nic type IPAM without throwing an error. To troubleshoot, we need additional information from [IPAM CNI log](#get-cni-log-available-after-v103).

#### Zero config

Zero config occurs when CNI cannot generate configurations from the network-attachment-definition. To troubleshoot, we need additional information from [CNI log](#get-cni-log).

### Ping failed
**Issue:** Pods cannot ping each other.

If the CNI operates at Layer 2, please confirm whether the defined Pod CIDR is routable within your cluster.

If the CNI operates at Layer 3, check route status in `multinicnetworks.multinic.fms.io`.
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
**Issue:** Pods can ping each other but do not get response from TCP/UDP communication such as iPerf.

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
Available configurations on `config.multinic/multi-nicd`:

### Controller configuration 

These following controller configuration values will be applied on-the-fly (no need to restart the controller pod).

Configuration|Description|Default Value
---|---|---
.spec.logLevel|controller's verbose log level|4
.spec.urgentReconcileSeconds|time to requeue reconcile after instant failure in second unit|5 seconds
.spec.normalReconcileMinutes|time to requeue reconcile while waiting for initial configuration in minute unit|1 minute
.spec.longReconcileMinutes|time to requeue reconcile when sensing control traffic failure in minute unit|10 minutes
.spec.contextTimeoutMinutes|time out for API server call context in minute unit|2 minutes

#### Log Levels

Verbose Level | Information
---|---
1|- critical error (cannot create/update resource by k8s API) <br> - "Set Config" key <br> - set up log <br>- config error
2|- significant events/failures of multinicnetwork
3|- significant events/failures of cidr
4 (default)|- significant events/failures of hostinterface
5|- significant events/failures of ippools
6|- significant events/failures of route configurations 
7|- requeue <br> - get deleted resource <br> - debug pointers (e.g., start point of function call)


### Daemon configuration 

Configuration|Description|Type|Default Value
---|---|---|---
.spec.daemon.port|multi-nicd serving port|int|11000
.spec.daemon.mounts|additional host-path mount|HostPathMount|

```yaml
# HostPathMount
mounts:
- name: mountName
  podpath: path/on/pod
  hostpath: path/on/host
```

Additionally, the following common [apps/DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) configurations are also available under `.spec.daemon`.
- nodeSelector
- image
- imagePullSecret
- imagePullPolicy
- securityContext
- env
- envFrom
- resources
- tolerations

### List in-use pods

**modify '< MULTINICNETWORK NAME HERE >'** in the following command with your target.multinicnetwork name

```bash
kubectl get po -A -ojson| jq -r '.items[]|select(.metadata.annotations."k8s.v1.cni.cncf.io/networks"=="< MULTINICNETWORK NAME HERE >")|.metadata.namespace + " " + .metadata.name'
```

### Get CNI log (available after v1.0.3)
To make CNI log available on the daemon pod, you may mount the the host log path to the daemon pod:

- Run 

```bash
kubectl edit config.multinic multi-nicd
```

- Add the following mount items

```yaml
# config/multi-nicd
spec:
  daemon:
    mounts:
    ...
    - hostpath: /var/log/multi-nic-cni.log
      name: cni-log
      podpath: /host/var/log/multi-nic-cni.log
    - hostpath: /var/log/multi-nic-ipam.log
      name: ipam-log
      podpath: /host/var/log/multi-nic-ipam.log
    # For AWS-IPVLAN main plugin log also add the following lines:
    # - hostpath: /var/log/multi-nic-aws-ipvlan.log
    #   name: ipam-log
    #   podpath: /host/var/log/multi-nic-aws-ipvlan.log
```

Then, you can get CNI log from the following commands:

```bash
# default main plugin
kubectl exec $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')\
-- cat /host/var/log/multi-nic-cni.log

# multi-nic on aws main plugin
kubectl exec $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')\
-- cat /host/var/log/multi-nic-aws-ipvlan.log

# IPAM plugin
kubectl exec $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')\
-- cat /host/var/log/multi-nic-ipam.log
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

### Restart multi-nicd
```bash
kubectl delete po $(kubectl get po -owide -A|grep multi-nicd\
|grep $FAILED_NODE|awk '{printf "%s -n %s", $2, $1}')
```

### Check host secondary interfaces 
Log in to FAILED_NODE with `oc debug node/$FAILED_NODE` or using [nettools](https://github.com/jedrecord/nettools) with `hostNetwork: true`. If secondary interfaces do not exist at the host network or an IPv4 address has not been assigned, [add the secondary interfaces](#add-secondary-interfaces-at-nodes-host-network)

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

### Customize Multi-NIC CNI controller of operator
If the multi-nic-cni operator has been managed by the Operator Lifecycle Manager (olm)  (installed by operator-sdk run bundle or via operator hub), the modification to the controller deployment (multi-nic-cni controller pod) will be overriden by the olm. 

To modify the value such as resource request/limit to the controller pod, you need to edit the `.spec.install.spec.deployments` section in the ClusterServiceVersion (csv) resource of the multi-nic-cni operator. 

You can locate the csv resource of multi-nic-cni operator in your cluster from the following command.

```
kubectl get csv -l operators.coreos.com/multi-nic-cni-operator.multi-nic-cni-operator -A
```


*Before v1.0.5, the csv are created in all namespaces. You need to edit the csv in the namespace that the controller has been deployed. The modification of csv in the other namespace will not be applied.*


