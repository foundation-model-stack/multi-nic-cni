/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	HOST_DEVICE_TYPE                     = "host-device"
	MELLANOX_TYPE                        = "mellanox"
	DEFAULT_MELLANOX_PREFIX              = "nvidia.com"
	MELLANOX_NETWORK_RESOURCE            = "hostdevicenetworks.v1alpha1.mellanox.com"
	MELLANOX_NIC_CLUSTER_POLICY_RESOURCE = "nicclusterpolicies.v1alpha1.mellanox.com"
)

type MellanoxPlugin struct {
	MellanoxNetworkHandler          *DynamicHandler
	MellanoxNicClusterPolicyHandler *DynamicHandler
}

// origin: https://github.com/containernetworking/plugins/blob/283f200489b5ef8f0b6aadc09f751ab0c5145497/plugins/main/host-device/host-device.go#L45C1-L56C2
// template: https://github.com/Mellanox/network-operator/blob/ce089f067153ea73b4712f0ad905fea92a1cf453/manifests/state-hostdevice-network/0010-hostdevice-net-cr.yml#L11
type HostDeviceTypeNetConf struct {
	types.NetConf
}

func (p *MellanoxPlugin) Init(config *rest.Config) error {
	dyn, err := dynamic.NewForConfig(config)
	mellanoxNetwork, _ := schema.ParseResourceArg(MELLANOX_NETWORK_RESOURCE)
	p.MellanoxNetworkHandler = &DynamicHandler{
		DYN: dyn,
		GVR: *mellanoxNetwork,
	}
	nicClusterPolicy, _ := schema.ParseResourceArg(MELLANOX_NIC_CLUSTER_POLICY_RESOURCE)
	p.MellanoxNicClusterPolicyHandler = &DynamicHandler{
		DYN: dyn,
		GVR: *nicClusterPolicy,
	}
	return err
}

func (p *MellanoxPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	annotation := make(map[string]string)
	var err error
	// get resource from nicclusterpolicy
	resourceName := p.GetResourceName()
	if resourceName == "" {
		msg := "failed to get resource name from sriov plugin config"
		vars.NetworkLog.V(2).Info(msg)
		return "", annotation, fmt.Errorf(msg)
	}

	name := net.GetName()
	namespace := net.GetNamespace()
	netName := GetHolderNetName(name)
	hostdevicenetwork, err := p.createHostDeviceNetwork(net.Spec.IPAM, netName, namespace, resourceName)
	if err != nil {
		return "", annotation, err
	}
	vars.NetworkLog.V(2).Info(fmt.Sprintf("hostdevicenetwork %s created", hostdevicenetwork.Name))
	conf := HostDeviceTypeNetConf{}
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = HOST_DEVICE_TYPE
	conf.Name = net.Name
	err = json.Unmarshal([]byte(net.Spec.IPAM), &conf.IPAM)
	if err != nil {
		return "", annotation, err
	}
	confBytes, err := json.Marshal(conf)
	if err != nil {
		return "", annotation, err
	}
	annotation[RESOURCE_ANNOTATION] = resourceName
	return string(confBytes), annotation, nil
}

// return first resource name found in SriovDevicePlugin
func (p *MellanoxPlugin) GetResourceName() string {
	policy, err := p.getPolicy()
	if err != nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("failed to get policy: %v", err))
	}
	if err != nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("failed to read sriov plugin config: %v", err))
		return ""
	}
	sriovPlugin := policy.Spec.SriovDevicePlugin
	if sriovPlugin == nil || sriovPlugin.Config == nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("no sriov device plugin config set in %s", policy.Name))
		return ""
	}
	configStr := *sriovPlugin.Config
	var config map[string]interface{}
	err = json.Unmarshal([]byte(configStr), &config)
	if err == nil {
		if resourceListStr, ok := config["resourceList"]; ok {
			if resourceList, ok := resourceListStr.([]interface{}); ok {
				for _, resource := range resourceList {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						if resourcePrefix, ok := resourceMap["resourcePrefix"]; ok {
							if resourceName, ok := resourceMap["resourceName"]; ok {
								return resourcePrefix.(string) + "/" + resourceName.(string)
							} else {
								return DEFAULT_MELLANOX_PREFIX + "/" + resourceName.(string)
							}
						}
					}
				}
			}
		}
	}
	vars.NetworkLog.V(2).Info(fmt.Sprintf("cannot read value from sriov config: %v", err))
	return ""
}

func (p *MellanoxPlugin) getPolicy() (*NicClusterPolicy, error) {
	policy := &NicClusterPolicy{}
	err := p.MellanoxNicClusterPolicyHandler.GetFirst(metav1.NamespaceAll, policy)
	if err != nil {
		return nil, err
	}
	return policy, err
}

func (p *MellanoxPlugin) createHostDeviceNetwork(ipam string, name string, namespace string, resourceName string) (*HostDeviceNetwork, error) {
	spec := &HostDeviceNetworkSpec{}
	spec.NetworkNamespace = "default"
	spec.ResourceName = resourceName
	spec.IPAM = ipam
	metaObj := GetMetaObject(name, namespace, make(map[string]string))
	hostDeviceNet := NewHostDeviceNetwork(metaObj, *spec)
	result := &HostDeviceNetwork{}
	err := p.MellanoxNetworkHandler.Create(metav1.NamespaceAll, hostDeviceNet, result)
	if k8serrors.IsAlreadyExists(err) {
		return result, nil
	}
	return result, err
}

func (p *MellanoxPlugin) CleanUp(net multinicv1.MultiNicNetwork) error {
	netName := GetHolderNetName(net.Name)
	return p.MellanoxNetworkHandler.Delete(netName, metav1.NamespaceAll)
}
