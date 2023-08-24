# Policy-based secondary network attachment
To apply attachment policy, the key `attachPolicy` need to be specified in MultiNicNetwork and specific arguments can be added specific to Pod annotation (if needed).
```yaml
# MultiNicNetwork 
spec:
  attachPolicy:
    strategy: none|costOpt|perfOpt|devClass
```
Policy|Description|Status
---|---|---
none (default)|Apply all NICs in the pool|implemented
costOpt|provide target ideal bandwidth with minimum cost based on HostInterface spec and status|TODO
perfOpt|provide target ideal bandwidth with most available NICs set based on HostInterface spec and status |TODO
devClass|give preference for a specific class of NICs based on DeviceClass custom resource|implemented
topology|give priority to NIC based on Numa affnity of GPU allocation|implemented

Annotation (CNIArgs)|Description|Status
---|---|---
nics|fixed number of interfaces (none, DeviceClass strategy)|implemented
master|fixed interface names (none strategy)|implemented
target|overridden target bandwidth (CostOpt, PerfOpt strategy)|TODO
class|preferred device class (DeviceClass strategy)|implemented

#### None Strategy (none)
When `none` strategy is set or no strategy is set, the Multi-NIC daemon will basically attach all secondary interfaces listed in HostInterface custom resource to the Pod. 
```yaml
# MultiNicNetwork 
spec:
  attachPolicy:
    strategy: none
```

To limit the secondary interfaces, the list of subnets can be specified in `.spec.masterNets`.

```yaml
# MultiNicNetwork 
spec:
  masterNets:
  - 10.0.0.0/16
  - 10.1.0.0/16
```

However, pod can be further annotated to apply only a subset of secondary interfaces with a specific number or name list.

For example, 
- attach only one secondary interface
```yaml
# Pod
metadata:
  annotations:
      k8s.v1.cni.cncf.io/networks: |
          [{
            "name": "multi-nic-sample",
            "cni-args": {
                "nics": 1
            }
          }]
```
- attach with the secondary interface name eth1
```yaml
# Pod
metadata:
  annotations:
      k8s.v1.cni.cncf.io/networks: |
          [{
            "name": "multi-nic-sample",
            "cni-args": {
                "master": [eth1]
            }
          }]
```
If both arguments (nics and master) are applied at the same time, the master argument will be applied.
#### DeviceClass Strategy (devClass)
When `devClass` strategy is set, the Multi-NIC daemon will be additionally aware of class argument specifed in the pod annotation as a filter.

```yaml
# MultiNicNetwork 
spec:
  attachPolicy:
    strategy: devClass
```

```yaml
# Pod
metadata:
  annotations:
      k8s.v1.cni.cncf.io/networks: |
          [{
            "name": "multi-nic-sample",
            "cni-args": {
                "class": "highspeed"
                "nics": 1
            }
          }]
```
With the above annotation, one secondary interface that falls into highspeed class defined by DeviceClass will be attached to the Pod.

The DeviceClass resource is composed of a list of vendor and product identifiers as below example. 
```yaml
# DeviceClass example
apiVersion: multinic.fms.io/v1
kind: DeviceClass
metadata:
  name: highspeed
spec:
  ids:
  - vendor: "15b3"
    products: 
    - "1019"
  - vendor: "1d0f"
    products: 
    - "efa0"
    - "efa1"
```

#### Topology Strategy 

When `topology` strategy is set and the number of NICs to select is set lower than availability, Multi-NIC daemon will prioritize the network device by the weight of NUMA where it is located.  

```yaml
# MultiNicNetwork 
spec:
  attachPolicy:
    strategy: topology
```

```yaml
# Pod
metadata:
  annotations:
      k8s.v1.cni.cncf.io/networks: |
          [{
            "name": "multi-nic-sample",
            "cni-args": {
                "nics": 1
            }
          }]
```

Weight is the number of GPU devices located on the NUMA that is assigned to the pod by the nvidia device plugin. 

If no topology file is provided in `/var/run/nvidia-topologyd/virtualTopology.xml`, the daemon will parse the topology from `/sys/devices`. 

