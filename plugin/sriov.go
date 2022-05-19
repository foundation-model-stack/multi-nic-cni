/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	SRIOV_RESOURCE_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
	SRIOV_RESOURCE_PREFIX     = "openshift.io"
	SRIOV_NAMESPACE           = "openshift-sriov-network-operator"
	SRIOV_IB_KEY              = "ib"
	SRIOV_TYPE                = "sriov"
	SRIOV_DEFAULT_NUMVFS      = 1
	SRIOV_NETWORK_SUBFIX      = "-net"

	SRIOV_NETWORK_RESOURCE    = "sriovnetworks.v1.sriovnetwork.openshift.io"
	SRIOV_POLICY_RESOURCE     = "sriovnetworknodepolicies.v1.sriovnetwork.openshift.io"
	SRIOV_NODE_STATE_RESOURCE = "sriovnetworknodestates.v1.sriovnetwork.openshift.io"
)

var SRIOV_NODE_SELECTOR map[string]string = map[string]string{
	"feature.node.kubernetes.io/network-sriov.capable": "true",
}

var SRIOV_MANIFEST_PATH string = "/template/cni-config"

const (
	RESOURCE_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
)

type SriovPlugin struct {
	Log                           logr.Logger
	SriovNetworkHandler           *DynamicHandler
	SriovNetworkNodePolicyHandler *DynamicHandler
	SriovNetworkNodeStateHandler  *DynamicHandler
}

func (p *SriovPlugin) Init(config *rest.Config) error {
	dyn, err := dynamic.NewForConfig(config)
	sriovnetwork, _ := schema.ParseResourceArg(SRIOV_NETWORK_RESOURCE)
	p.SriovNetworkHandler = &DynamicHandler{
		DYN: dyn,
		GVR: *sriovnetwork,
	}
	sriovpolicy, _ := schema.ParseResourceArg(SRIOV_POLICY_RESOURCE)
	p.SriovNetworkNodePolicyHandler = &DynamicHandler{
		DYN: dyn,
		GVR: *sriovpolicy,
	}
	sriovstate, _ := schema.ParseResourceArg(SRIOV_NODE_STATE_RESOURCE)
	p.SriovNetworkNodeStateHandler = &DynamicHandler{
		DYN: dyn,
		GVR: *sriovstate,
	}
	return err
}

func (p *SriovPlugin) GetConfig(net netcogadvisoriov1.MultiNicNetwork, hifList map[string]netcogadvisoriov1.HostInterface) (string, map[string]string, error) {
	annotation := make(map[string]string)
	name := net.GetName()
	namespace := net.GetNamespace()
	resourceName := p.ValidateResourceName(name) // default name
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	rootDevices := p.getRootDevices(net, hifList)
	// get resource, create new SriovNetworkNodePolicies if resource is not pre-defined
	// TO-DO: check configmap to verify pre-defined resourceName is valid
	resourceName = p.getResource(name, args, resourceName, rootDevices)
	var raw *unstructured.Unstructured
	var err error
	// create sriov network
	// TO-DO: support SriovIBNetwork
	sriovnet, err := p.createSriovNetwork(name, namespace, args, resourceName)
	if err != nil {
		return "", annotation, err
	}
	raw, err = sriovnet.RenderNetAttDef()
	if err != nil {
		return "", annotation, err
	}
	config := raw.Object["spec"].(map[string]interface{})["config"].(string)
	annotation[SRIOV_RESOURCE_ANNOTATION] = SRIOV_RESOURCE_PREFIX + "/" + resourceName
	return config, annotation, nil
}

func (p *SriovPlugin) getRootDevices(net netcogadvisoriov1.MultiNicNetwork, hifList map[string]netcogadvisoriov1.HostInterface) []string {
	masterNets := net.Spec.MasterNetAddrs

	netAddrHash := make(map[string]string)
	if len(masterNets) == 0 {
		for _, hif := range hifList {
			for _, iface := range hif.Spec.Interfaces {
				netAddrHash[iface.NetAddress] = ""
			}
		}
	} else {
		for _, netAddr := range masterNets {
			netAddrHash[netAddr] = ""
		}
	}

	for hostName, hif := range hifList {
		state := &SriovNetworkNodeState{}
		err := p.SriovNetworkNodeStateHandler.Get(hostName, SRIOV_NAMESPACE, state)
		if err != nil {
			continue
		}
		sriovIfaces := state.Status.Interfaces
		if sriovIfaces == nil {
			// no sriov interface
			continue
		}
		nameDeviceMap := make(map[string]string)
		for _, sriovIface := range sriovIfaces {
			nameDeviceMap[sriovIface.Name] = sriovIface.PciAddress
		}

		for _, iface := range hif.Spec.Interfaces {
			if _, exist := netAddrHash[iface.NetAddress]; exist {
				// target interface
				netAddrHash[iface.NetAddress] = nameDeviceMap[iface.InterfaceName]
			}
		}
		p.Log.Info(fmt.Sprintf("host: %s, deviceMap: %v", hostName, nameDeviceMap))
	}
	p.Log.Info(fmt.Sprintf("hif: %d, netAddrHash: %v", len(hifList), netAddrHash))
	rootDevices := []string{}
	for _, val := range netAddrHash {
		if val != "" {
			rootDevices = append(rootDevices, val)
		}
	}
	return rootDevices
}

func (p *SriovPlugin) getResource(name string, args map[string]string, resourceName string, rootDevices []string) string {
	spec := &SriovNetworkNodePolicySpec{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, spec)
	if spec.ResourceName == "" {
		// create new resource
		val, err := getInt(args, "priority")
		if err == nil {
			spec.Priority = val
		}
		val, err = getInt(args, "mtu")
		if err == nil {
			spec.Mtu = val
		}
		val, err = getInt(args, "numVfs")
		if err == nil {
			spec.NumVfs = val
		} else {
			spec.NumVfs = SRIOV_DEFAULT_NUMVFS
		}
		bVal, err := getBoolean(args, "isRdma")
		if err == nil {
			spec.IsRdma = bVal
		}
		bVal, err = getBoolean(args, "needVhostNet")
		if err == nil {
			spec.NeedVhostNet = bVal
		}
		spec.ResourceName = resourceName
		spec.NodeSelector = SRIOV_NODE_SELECTOR
		spec.NicSelector = SriovNetworkNicSelector{
			RootDevices: rootDevices,
		}
		metaObj := GetMetaObject(name, SRIOV_NAMESPACE, make(map[string]string))
		policy := NewSriovNetworkNodePolicy(metaObj, *spec)
		result := &SriovNetworkNodePolicy{}
		err = p.SriovNetworkNodePolicyHandler.Create(SRIOV_NAMESPACE, policy, result)
		if err != nil {
			p.Log.Info(fmt.Sprintf("Policy: %v", policy.Spec))
			p.Log.Info(fmt.Sprintf("Failed to create policy %s: %v", name, err))
		} else {
			p.Log.Info(fmt.Sprintf("Create new SriovNetworkNodePolicy: %s", name))
		}
		return resourceName
	}
	p.Log.Info(fmt.Sprintf("Use existing resource %s", spec.ResourceName))
	return spec.ResourceName
}

func (p *SriovPlugin) createSriovNetwork(name string, namespace string, args map[string]string, resourceName string) (*SriovNetwork, error) {
	spec := &SriovNetworkSpec{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, spec)
	spec.NetworkNamespace = namespace
	spec.ResourceName = resourceName
	val, err := getInt(args, "vlan")
	if err == nil {
		spec.Vlan = val
	}
	val, err = getInt(args, "vlanQoS")
	if err == nil {
		spec.VlanQoS = val
	}
	val, err = getInt(args, "minTxRate")
	if err == nil {
		spec.MinTxRate = &val
	}
	val, err = getInt(args, "maxTxRate")
	if err == nil {
		spec.MaxTxRate = &val
	}
	netName := p.SriovnetworkName(name)
	metaObj := GetMetaObject(netName, SRIOV_NAMESPACE, make(map[string]string))
	sriovnet := NewSrioNetwork(metaObj, *spec)
	result := &SriovNetwork{}

	err = p.SriovNetworkHandler.Create(SRIOV_NAMESPACE, sriovnet, result)
	return result, err
}

func (p *SriovPlugin) ValidateResourceName(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

func (p *SriovPlugin) SriovnetworkName(name string) string {
	return name + SRIOV_NETWORK_SUBFIX
}

func (p *SriovPlugin) CleanUp(net netcogadvisoriov1.MultiNicNetwork) error {
	name := net.GetName()
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	nodeSpec := &SriovNetworkNodePolicySpec{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, nodeSpec)
	var policyerr error
	if nodeSpec.ResourceName == "" {
		// multi-nic-defined resource
		policyerr = p.SriovNetworkNodePolicyHandler.Delete(name, SRIOV_NAMESPACE)
	}
	netName := p.SriovnetworkName(name)
	err := p.SriovNetworkHandler.Delete(netName, SRIOV_NAMESPACE)
	if policyerr != nil || err != nil {
		return fmt.Errorf("%v,%v", policyerr, err)
	}
	return nil
}
