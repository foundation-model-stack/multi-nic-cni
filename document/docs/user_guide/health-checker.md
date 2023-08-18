# Troubleshooting with Multi-NIC CNI Health Checker

|StatusCode |Message        |Potential causes                           |Actions|
|---        |---            |---                                        |---|
|200        |Success        |-                                          |-|
|400        |NetworkNotFound|NetworkAttachmentDefinition is not created.| Check multinicnetwork CR whether <br>the `.spec.namespaces` is limited to specific list.<br> If included your namespace, check [controller log](./troubleshooting.md#get-controller-log).|
|401        |PluginNotFound |CNI binary file is not available.| Check [CNI binary file](./troubleshooting.md#cni-binary-not-exist).|
|500        |ConfigFailure  |Configuration input is not valid.| Check `NetworkAttachmentDefinition.spec.config`<br> comparing with full error message from [/status response](#check-full-error-message). 
|501        |PluginNotSupport|CNI plugin is not supported <br>in the running environment.|Check [requirements of CNI](https://github.com/foundation-model-stack/multi-nic-cni#requirements).
|600        |NetNSFailured  |The agent process faces restriction<br>to open a new network Linux namespace <br>(e.g., privileges, namespace limited)|Check full error message from [/status response](#check-full-error-message). 
|601        |IPAMFailure|IPAM CNI fails to assign an IP|Check [IPAM CNI log](./troubleshooting.md#ipam-execadd-failed) and full error message from [/status response](#check-full-error-message). 
|602        |PluginExecFailure|(depends on failed CNI)|Check [CNI log](./troubleshooting.md#ipam-execadd-failed) and full error message from [/status response](#check-full-error-message). 
|603        |PartialFailure|Only some network device is not healthy|Identify failed network address from [/status response](#check-full-error-message).<br> Check [connectivity failure](./troubleshooting.md#ping-failed). 
|700        |DaemonConnectionFailure|Daemon pod is not running <br>or fails to response.|Check if [multi-nicd is deployed]((#hostinterface-not-created) ).<br> If yes, check [multi-nicd log](#get-multi-nicd-log).
|999        |Unknown|No health agent running by [taint](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)<br>or checker failed to get response.|If agent is normally running, <br>check full error message from [/status response](#check-full-error-message).

### Check full error message

```bash
# port-forward checker pod on one terminal
checker=$(kubectl get po -n multi-nic-cni-operator|grep multi-nic-cni-health-checker|awk '{ print $1 }')
kubectl port-forward ${checker} -n multi-nic-cni-operator 8080:8080

# request status on specific node
export FAILED_NODE= # failed node name
curl "localhost:8080/status?host=${FAILED_NODE}"
```