/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
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

// SrIoVResource defines prefix and resource of resource from sriov plugin
type SrIoVResource struct {
	Prefix       string
	ResourceName string
}

// GetAnnotation returns prefix/resourceName
func (r SrIoVResource) GetAnnotation() string {
	return r.Prefix + "/" + r.ResourceName
}

// Valid checks if resourceName has value.
func (r SrIoVResource) Valid() bool {
	return r.ResourceName != ""
}

// GetSrIoVResource returns SrIoVResource from item in resourceList
func GetSrIoVResource(resourceMap map[string]interface{}) *SrIoVResource {
	if resourcePrefix, ok := resourceMap["resourcePrefix"]; ok {
		if resourceName, ok := resourceMap["resourceName"]; ok {
			return &SrIoVResource{
				Prefix:       resourcePrefix.(string),
				ResourceName: resourceName.(string),
			}
		} else {
			return &SrIoVResource{
				Prefix:       DEFAULT_MELLANOX_PREFIX,
				ResourceName: resourceName.(string),
			}
		}
	}
	return nil
}

func GetSrIoVResourcesFromSrIoVPlugin(sriovPlugin *DevicePluginSpec) (rs []SrIoVResource, err error) {
	configStr := *sriovPlugin.Config
	var config map[string]interface{}
	err = json.Unmarshal([]byte(configStr), &config)
	if err == nil {
		if resourceListStr, ok := config["resourceList"]; ok {
			if resourceList, ok := resourceListStr.([]interface{}); ok {
				for _, resource := range resourceList {
					if resourceMap, ok := resource.(map[string]interface{}); ok {
						if r := GetSrIoVResource(resourceMap); r != nil {
							rs = append(rs, *r)
						}
					}
				}
			}
		}
	}
	return rs, err
}

// GetCombinedResourceNames returns string joining SrIoVResource annotation list with comma
func GetCombinedResourceNames(rs []SrIoVResource) string {
	annotaions := []string{}
	for _, r := range rs {
		if r.Valid() {
			annotaions = append(annotaions, r.GetAnnotation())
		}
	}
	return strings.Join(annotaions, ",")
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
	rs := p.GetSrIoVResources()
	resourceAnnotation := GetCombinedResourceNames(rs)
	if resourceAnnotation == "" {
		msg := "failed to get resource annotation from sriov plugin config"
		vars.NetworkLog.V(2).Info(msg)
		return "", annotation, errors.New(msg)
	}
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
	annotation[RESOURCE_ANNOTATION] = resourceAnnotation
	return string(confBytes), annotation, nil
}

// return list of SrIoVResource found in SriovDevicePlugin
func (p *MellanoxPlugin) GetSrIoVResources() (rs []SrIoVResource) {
	policy, err := p.getPolicy()
	if err != nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("failed to get policy: %v", err))
	}
	if err != nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("failed to read sriov plugin config: %v", err))
		return rs
	}
	sriovPlugin := policy.Spec.SriovDevicePlugin
	if sriovPlugin == nil || sriovPlugin.Config == nil {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("no sriov device plugin config set in %s", policy.Name))
		return rs
	}
	rs, err = GetSrIoVResourcesFromSrIoVPlugin(sriovPlugin)
	if err != nil || len(rs) == 0 {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("no readable value from sriov config (%s): %v", *sriovPlugin.Config, err))
	}
	return rs
}

func (p *MellanoxPlugin) getPolicy() (*NicClusterPolicy, error) {
	policy := &NicClusterPolicy{}
	err := p.MellanoxNicClusterPolicyHandler.GetFirst(metav1.NamespaceAll, policy)
	if err != nil {
		return nil, err
	}
	return policy, err
}

func (p *MellanoxPlugin) CleanUp(net multinicv1.MultiNicNetwork) error {
	return nil
}
