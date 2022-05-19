# Attachment Policy
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
However, pod can be further annotated to apply only a subset of secondary interfaces with a specific number or name list.

For example, 
- attach only one secondary interface
```yaml
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
When `devClass` strategy is, the Multi-NIC daemon will be additionally aware of class argument specifed in the pod annotation as a filter.
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
apiVersion: net.cogadvisor.io/v1
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
