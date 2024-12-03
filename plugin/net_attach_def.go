/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	CNI_VERSION                = "0.3.0"
	NET_ATTACH_DEF_API_VERSION = "k8s.cni.cncf.io/v1"
	NET_ATTACH_DEF_RESOURCE    = "network-attachment-definitions.v1.k8s.cni.cncf.io"
	NET_ATTACH_DEF_KIND        = "NetworkAttachmentDefinition"
	StatusesKey                = "k8s.v1.cni.cncf.io/networks-status"
)

// ////////////////////////////////////////
// NetworkAttachmentDefinition
// reference: github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1
func NewNetworkAttachmentDefinition(metaObj metav1.ObjectMeta, spec NetworkAttachmentDefinitionSpec) NetworkAttachmentDefinition {
	return NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: NET_ATTACH_DEF_API_VERSION,
			Kind:       NET_ATTACH_DEF_KIND,
		},
		ObjectMeta: metaObj,
		Spec:       spec,
	}
}

type NetworkAttachmentDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkAttachmentDefinitionSpec `json:"spec"`
}

type NetworkAttachmentDefinitionSpec struct {
	Config string `json:"config"`
}

func (def *NetworkAttachmentDefinition) GetName() string {
	return def.ObjectMeta.GetName()
}
func (def *NetworkAttachmentDefinition) GetNameSpace() string {
	return def.ObjectMeta.GetNamespace()
}

//////////////////////////////////////////

// NetConf defines general config for multi-nic-cni
type NetConf struct {
	types.NetConf
	MainPlugin     interface{} `json:"plugin"`
	Subnet         string      `json:"subnet"`
	MasterNetAddrs []string    `json:"masterNets"`
	DeviceIDs      []string    `json:"deviceIDs,omitempty"`
	IsMultiNICIPAM bool        `json:"multiNICIPAM,omitempty"`
	DaemonIP       string      `json:"daemonIP"`
	DaemonPort     int         `json:"daemonPort"`
}

type NetworkStatus struct {
	Name      string    `json:"name"`
	Interface string    `json:"interface,omitempty"`
	IPs       []string  `json:"ips,omitempty"`
	Mac       string    `json:"mac,omitempty"`
	Default   bool      `json:"default,omitempty"`
	DNS       types.DNS `json:"dns,omitempty"`
}

type NetAttachDefHandler struct {
	*DynamicHandler
	*kubernetes.Clientset
}

func GetNetAttachDefHandler(config *rest.Config) (*NetAttachDefHandler, error) {
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	netattachdef, _ := schema.ParseResourceArg(NET_ATTACH_DEF_RESOURCE)
	clientset, _ := kubernetes.NewForConfig(config)
	handler := &DynamicHandler{
		DYN: dyn,
		GVR: *netattachdef,
	}
	return &NetAttachDefHandler{
		DynamicHandler: handler,
		Clientset:      clientset,
	}, nil
}

func CheckDefChanged(def, existingDef *NetworkAttachmentDefinition) bool {
	if def.Spec.Config != existingDef.Spec.Config || len(def.Annotations) != len(existingDef.Annotations) {
		return true
	}
	for k, v := range existingDef.Annotations {
		if def.Annotations[k] != v {
			return true
		}
	}
	return false
}

func (h *NetAttachDefHandler) CreateOrUpdate(net *multinicv1.MultiNicNetwork, pluginStr string, annotations map[string]string) error {
	defs, err := h.generate(net, pluginStr, annotations)
	if err != nil {
		return err
	}
	errMsg := ""
	for _, def := range defs {
		errMsg = h.createOrUpdate(def, errMsg)
	}
	if errMsg != "" {
		vars.NetworkLog.V(2).Info(errMsg)
	}
	return nil
}

func (h *NetAttachDefHandler) CreateOrUpdateOnNamespace(ns string, net *multinicv1.MultiNicNetwork, pluginStr string, annotations map[string]string) error {
	def, err := NetToDef(ns, net, pluginStr, annotations)
	if err != nil {
		return err
	}
	errMsg := h.createOrUpdate(def, "")
	if errMsg != "" {
		vars.NetworkLog.V(2).Info(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// createOrUpdate creates new NetworkAttachmentDefinition resource if not exists, otherwise update
func (h *NetAttachDefHandler) createOrUpdate(def *NetworkAttachmentDefinition, errMsg string) string {
	name := def.GetName()
	namespace := def.GetNamespace()
	result := &NetworkAttachmentDefinition{}
	if h.IsExist(name, namespace) {
		existingDef, _ := h.Get(name, namespace)
		def.ObjectMeta = existingDef.ObjectMeta
		if CheckDefChanged(def, existingDef) {
			if namespace == "default" {
				vars.NetworkLog.V(2).Info(fmt.Sprintf("Update net-attach-def %s", def.Name))
			}
			err := h.DynamicHandler.Update(namespace, def, result)
			if err != nil {
				errMsg = fmt.Sprintf("%s\n%s: %v", errMsg, namespace, err)
			}
		}
	} else {
		err := h.DynamicHandler.Create(namespace, def, result)
		if err != nil {
			errMsg = fmt.Sprintf("%s\n%s: %v", errMsg, namespace, err)
		}
	}
	return errMsg
}

// getNamespace returns all available namespaces if .Spec.Namespaces not specified
func (h *NetAttachDefHandler) getNamespace(net *multinicv1.MultiNicNetwork) ([]string, error) {
	namespaces := net.Spec.Namespaces
	if len(namespaces) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		namespaceList, err := h.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, ns := range namespaceList.Items {
				if ns.Status.Phase == v1.NamespaceActive {
					namespaces = append(namespaces, ns.Name)
				}
			}
		} else {
			return namespaces, err
		}
	}
	return namespaces, nil
}

// NetToDef generates net-attach-def from multinicnetwork on specific namespace called by generate function
func NetToDef(namespace string, net *multinicv1.MultiNicNetwork, pluginStr string, annotations map[string]string) (*NetworkAttachmentDefinition, error) {
	name := net.GetName()
	config := &NetConf{
		NetConf: types.NetConf{
			CNIVersion: CNI_VERSION,
			Name:       name,
			Type:       vars.TargetCNI,
		},
		Subnet:         net.Spec.Subnet,
		MasterNetAddrs: net.Spec.MasterNetAddrs,
		IsMultiNICIPAM: net.Spec.IsMultiNICIPAM,
		DaemonPort:     vars.DaemonPort,
	}
	var ipamObj map[string]interface{}
	configBytes, _ := json.Marshal(config)
	configStr := string(configBytes)
	err := json.Unmarshal([]byte(net.Spec.IPAM), &ipamObj)
	if err != nil {
		return nil, err
	}
	ipamBytes, _ := json.Marshal(ipamObj)
	pluginValue := fmt.Sprintf("\"plugin\":%s", pluginStr)
	ipamValue := fmt.Sprintf("\"ipam\":%s", string(ipamBytes))
	configStr = strings.ReplaceAll(configStr, "\"ipam\":{}", ipamValue)
	configStr = strings.ReplaceAll(configStr, "\"plugin\":null", pluginValue)
	metaObj := GetMetaObject(name, namespace, annotations)
	spec := NetworkAttachmentDefinitionSpec{
		Config: configStr,
	}
	netattachdef := NewNetworkAttachmentDefinition(metaObj, spec)
	return &netattachdef, nil
}

// generate initializes NetworkAttachmentDefinition objects from MultiNicNetwork and unmarshal plugin
func (h *NetAttachDefHandler) generate(net *multinicv1.MultiNicNetwork, pluginStr string, annotations map[string]string) ([]*NetworkAttachmentDefinition, error) {
	defs := []*NetworkAttachmentDefinition{}
	namespaces, err := h.getNamespace(net)
	if err != nil {
		return defs, err
	}
	vars.NetworkLog.V(2).Info(fmt.Sprintf("generate net-attach-def config on %d namespaces", len(namespaces)))
	for _, ns := range namespaces {
		def, err := NetToDef(ns, net, pluginStr, annotations)
		if err != nil {
			return defs, err
		}
		defs = append(defs, def)
	}
	return defs, nil
}

func (h *NetAttachDefHandler) DeleteNets(net *multinicv1.MultiNicNetwork) error {
	namespaces, err := h.getNamespace(net)
	for _, ns := range namespaces {
		nsErr := h.Delete(net.GetName(), ns)
		if nsErr != nil {
			err = nsErr
		}
	}
	return err
}

// Get returns NetworkAttachmentDefinition object from name and namespace
func (h *NetAttachDefHandler) Get(name string, namespace string) (*NetworkAttachmentDefinition, error) {
	result := &NetworkAttachmentDefinition{}
	err := h.DynamicHandler.Get(name, namespace, result)
	return result, err
}

// IsExist checks if the NetworkAttachmentDefinition exist from name and namespace
func (h *NetAttachDefHandler) IsExist(name string, namespace string) bool {
	_, err := h.Get(name, namespace)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			vars.NetworkLog.V(2).Info(fmt.Sprintf("Not exist: %v", err))
		}
		return false
	}
	return true
}

// Delete deletes NetworkAttachmentDefinition from name and namespace
func (h *NetAttachDefHandler) Delete(name string, namespace string) error {
	return h.DynamicHandler.Delete(name, namespace)
}
