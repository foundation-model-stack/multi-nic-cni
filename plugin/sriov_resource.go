/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ////////////////////////////////////////
// SR-IOV-related resources
// reference: github.com/k8snetworkplumbingwg/sriov-network-operator/api/v1@bc40302
// ////////////////////////////////////////
const (
	SRIOV_API_VERSION     = "sriovnetwork.openshift.io/v1"
	SRIOV_NETWORK_KIND    = "SriovNetwork"
	SRIOV_POLICY_KIND     = "SriovNetworkNodePolicy"
	SRIOV_NODE_STATE_KIND = "SriovNetworkNodeState"
)

func NewSrioNetwork(metaObj metav1.ObjectMeta, spec SriovNetworkSpec) SriovNetwork {
	return SriovNetwork{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SRIOV_API_VERSION,
			Kind:       SRIOV_NETWORK_KIND,
		},
		ObjectMeta: metaObj,
		Spec:       spec,
	}
}

// SriovNetworkStatus defines the observed state of SriovNetwork
type SriovNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkSpec   `json:"spec,omitempty"`
	Status SriovNetworkStatus `json:"status,omitempty"`
}

// SriovNetworkSpec defines the desired state of SriovNetwork
type SriovNetworkSpec struct {
	// Namespace of the NetworkAttachmentDefinition custom resource
	NetworkNamespace string `json:"networkNamespace,omitempty"`
	// SRIOV Network device plugin endpoint resource name
	ResourceName string `json:"resourceName"`
	//Capabilities to be configured for this network.
	//Capabilities supported: (mac|ips), e.g. '{"mac": true}'
	Capabilities string `json:"capabilities,omitempty"`
	//IPAM configuration to be used for this network.
	IPAM string `json:"ipam,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4096
	// VLAN ID to assign for the VF. Defaults to 0.
	Vlan int `json:"vlan,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=7
	// VLAN QoS ID to assign for the VF. Defaults to 0.
	VlanQoS int `json:"vlanQoS,omitempty"`
	// VF spoof check, (on|off)
	// +kubebuilder:validation:Enum={"on","off"}
	SpoofChk string `json:"spoofChk,omitempty"`
	// VF trust mode (on|off)
	// +kubebuilder:validation:Enum={"on","off"}
	Trust string `json:"trust,omitempty"`
	// VF link state (enable|disable|auto)
	// +kubebuilder:validation:Enum={"auto","enable","disable"}
	LinkState string `json:"linkState,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Minimum tx rate, in Mbps, for the VF. Defaults to 0 (no rate limiting). min_tx_rate should be <= max_tx_rate.
	MinTxRate *int `json:"minTxRate,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Maximum tx rate, in Mbps, for the VF. Defaults to 0 (no rate limiting)
	MaxTxRate *int `json:"maxTxRate,omitempty"`
	// MetaPluginsConfig configuration to be used in order to chain metaplugins to the sriov interface returned
	// by the operator.
	MetaPluginsConfig string `json:"metaPlugins,omitempty"`
}

// SriovNetworkStatus defines the observed state of SriovNetwork
type SriovNetworkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

func NewSriovNetworkNodePolicy(metaObj metav1.ObjectMeta, spec SriovNetworkNodePolicySpec) SriovNetworkNodePolicy {
	return SriovNetworkNodePolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SRIOV_API_VERSION,
			Kind:       SRIOV_POLICY_KIND,
		},
		ObjectMeta: metaObj,
		Spec:       spec,
	}
}

// SriovNetworkNodePolicy is the Schema for the sriovnetworknodepolicies API
type SriovNetworkNodePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodePolicySpec   `json:"spec,omitempty"`
	Status SriovNetworkNodePolicyStatus `json:"status,omitempty"`
}

// SriovNetworkNodePolicySpec defines the desired state of SriovNetworkNodePolicy
type SriovNetworkNodePolicySpec struct {
	// SRIOV Network device plugin endpoint resource name
	ResourceName string `json:"resourceName"`
	// NodeSelector selects the nodes to be configured
	NodeSelector map[string]string `json:"nodeSelector"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=99
	// Priority of the policy, higher priority policies can override lower ones.
	Priority int `json:"priority,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// MTU of VF
	Mtu int `json:"mtu,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Number of VFs for each PF
	NumVfs int `json:"numVfs"`
	// NicSelector selects the NICs to be configured
	NicSelector SriovNetworkNicSelector `json:"nicSelector"`
	// +kubebuilder:validation:Enum=netdevice;vfio-pci
	// The driver type for configured VFs. Allowed value "netdevice", "vfio-pci". Defaults to netdevice.
	DeviceType string `json:"deviceType,omitempty"`
	// RDMA mode. Defaults to false.
	IsRdma bool `json:"isRdma,omitempty"`
	// mount vhost-net device. Defaults to false.
	NeedVhostNet bool `json:"needVhostNet,omitempty"`
	// +kubebuilder:validation:Enum=eth;ETH;ib;IB
	// NIC Link Type. Allowed value "eth", "ETH", "ib", and "IB".
	LinkType string `json:"linkType,omitempty"`
	// +kubebuilder:validation:Enum=legacy;switchdev
	// NIC Device Mode. Allowed value "legacy","switchdev".
	EswitchMode string `json:"eSwitchMode,omitempty"`
}

// SriovNetworkNodePolicyStatus defines the observed state of SriovNetworkNodePolicy
type SriovNetworkNodePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type SriovNetworkNicSelector struct {
	// The vendor hex code of SR-IoV device. Allowed value "8086", "15b3".
	Vendor string `json:"vendor,omitempty"`
	// The device hex code of SR-IoV device. Allowed value "0d58", "1572", "158b", "1013", "1015", "1017", "101b".
	DeviceID string `json:"deviceID,omitempty"`
	// PCI address of SR-IoV PF.
	RootDevices []string `json:"rootDevices,omitempty"`
	// Name of SR-IoV PF.
	PfNames []string `json:"pfNames,omitempty"`
	// Infrastructure Networking selection filter. Allowed value "openstack/NetworkID:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	NetFilter string `json:"netFilter,omitempty"`
}

// RenderNetAttDef renders a net-att-def for sriov CNI
func (cr *SriovNetwork) RenderNetAttDef() (*unstructured.Unstructured, error) {
	// render RawCNIConfig manifests
	data := MakeRenderData()
	data.Data["MetaPluginsConfigured"] = false
	data.Data["CniType"] = "sriov"
	data.Data["SriovNetworkName"] = cr.Name
	if cr.Spec.NetworkNamespace == "" {
		data.Data["SriovNetworkNamespace"] = cr.Namespace
	} else {
		data.Data["SriovNetworkNamespace"] = cr.Spec.NetworkNamespace
	}
	data.Data["SriovCniResourceName"] = os.Getenv("RESOURCE_PREFIX") + "/" + cr.Spec.ResourceName
	data.Data["SriovCniVlan"] = cr.Spec.Vlan

	if cr.Spec.VlanQoS <= 7 && cr.Spec.VlanQoS >= 0 {
		data.Data["VlanQoSConfigured"] = true
		data.Data["SriovCniVlanQoS"] = cr.Spec.VlanQoS
	} else {
		data.Data["VlanQoSConfigured"] = false
	}

	if cr.Spec.Capabilities == "" {
		data.Data["CapabilitiesConfigured"] = false
	} else {
		data.Data["CapabilitiesConfigured"] = true
		data.Data["SriovCniCapabilities"] = cr.Spec.Capabilities
	}

	data.Data["SpoofChkConfigured"] = true
	switch cr.Spec.SpoofChk {
	case "off":
		data.Data["SriovCniSpoofChk"] = "off"
	case "on":
		data.Data["SriovCniSpoofChk"] = "on"
	default:
		data.Data["SpoofChkConfigured"] = false
	}

	data.Data["TrustConfigured"] = true
	switch cr.Spec.Trust {
	case "on":
		data.Data["SriovCniTrust"] = "on"
	case "off":
		data.Data["SriovCniTrust"] = "off"
	default:
		data.Data["TrustConfigured"] = false
	}

	data.Data["StateConfigured"] = true
	switch cr.Spec.LinkState {
	case "enable":
		data.Data["SriovCniState"] = "enable"
	case "disable":
		data.Data["SriovCniState"] = "disable"
	case "auto":
		data.Data["SriovCniState"] = "auto"
	default:
		data.Data["StateConfigured"] = false
	}

	data.Data["MinTxRateConfigured"] = false
	if cr.Spec.MinTxRate != nil {
		if *cr.Spec.MinTxRate >= 0 {
			data.Data["MinTxRateConfigured"] = true
			data.Data["SriovCniMinTxRate"] = *cr.Spec.MinTxRate
		}
	}

	data.Data["MaxTxRateConfigured"] = false
	if cr.Spec.MaxTxRate != nil {
		if *cr.Spec.MaxTxRate >= 0 {
			data.Data["MaxTxRateConfigured"] = true
			data.Data["SriovCniMaxTxRate"] = *cr.Spec.MaxTxRate
		}
	}

	if cr.Spec.IPAM != "" {
		data.Data["SriovCniIpam"] = "\"ipam\":" + strings.Join(strings.Fields(cr.Spec.IPAM), "")
	} else {
		data.Data["SriovCniIpam"] = "\"ipam\":{}"
	}

	objs, err := RenderDir(SRIOV_MANIFEST_PATH, &data)
	if err != nil {
		return nil, err
	}
	return objs[0], nil
}

type RenderData struct {
	Funcs template.FuncMap
	Data  map[string]interface{}
}

func MakeRenderData() RenderData {
	return RenderData{
		Funcs: template.FuncMap{},
		Data:  map[string]interface{}{},
	}
}

// Functions available for all templates

// getOr returns the value of m[key] if it exists, fallback otherwise.
// As a special case, it also returns fallback if the value of m[key] is
// the empty string
func getOr(m map[string]interface{}, key string, fallback interface{}) interface{} {
	val, ok := m[key]
	if !ok {
		return fallback
	}

	s, ok := val.(string)
	if ok && s == "" {
		return fallback
	}

	return val
}

// isSet returns the value of m[key] if key exists, otherwise false
// Different from getOr because it will return zero values.
func isSet(m map[string]interface{}, key string) interface{} {
	val, ok := m[key]
	if !ok {
		return false
	}
	return val
}

// RenderTemplate reads, renders, and attempts to parse a yaml or
// json file representing one or more k8s api objects
func RenderTemplate(path string, d *RenderData) ([]*unstructured.Unstructured, error) {
	tmpl := template.New(path).Option("missingkey=error")
	if d.Funcs != nil {
		tmpl.Funcs(d.Funcs)
	}

	// Add universal functions
	tmpl.Funcs(template.FuncMap{"getOr": getOr, "isSet": isSet})
	tmpl.Funcs(sprig.TxtFuncMap())

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifest %s", path)
	}

	if _, err := tmpl.Parse(string(source)); err != nil {
		return nil, errors.Wrapf(err, "failed to parse manifest %s as template", path)
	}

	rendered := bytes.Buffer{}
	if err := tmpl.Execute(&rendered, d.Data); err != nil {
		return nil, errors.Wrapf(err, "failed to render manifest %s", path)
	}

	out := []*unstructured.Unstructured{}

	// special case - if the entire file is whitespace, skip
	if len(strings.TrimSpace(rendered.String())) == 0 {
		return out, nil
	}

	decoder := yaml.NewYAMLOrJSONDecoder(&rendered, 4096)
	for {
		u := unstructured.Unstructured{}
		if err := decoder.Decode(&u); err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrapf(err, "failed to unmarshal manifest %s", path)
		}
		out = append(out, &u)
	}

	return out, nil
}

// RenderDir will render all manifests in a directory, descending in to subdirectories
// It will perform template substitutions based on the data supplied by the RenderData
func RenderDir(manifestDir string, d *RenderData) ([]*unstructured.Unstructured, error) {
	out := []*unstructured.Unstructured{}

	if err := filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip non-manifest files
		if !(strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".json")) {
			return nil
		}

		objs, err := RenderTemplate(path, d)
		if err != nil {
			return err
		}
		out = append(out, objs...)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "error rendering manifests")
	}

	return out, nil
}

// SriovNetworkNodeState is the Schema for the sriovnetworknodestates API
type SriovNetworkNodeState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodeStateSpec   `json:"spec,omitempty"`
	Status SriovNetworkNodeStateStatus `json:"status,omitempty"`
}

// SriovNetworkNodeStateStatus defines the observed state of SriovNetworkNodeState
type SriovNetworkNodeStateStatus struct {
	Interfaces    InterfaceExts `json:"interfaces,omitempty"`
	SyncStatus    string        `json:"syncStatus,omitempty"`
	LastSyncError string        `json:"lastSyncError,omitempty"`
}

// SriovNetworkNodeStateSpec defines the desired state of SriovNetworkNodeState
type SriovNetworkNodeStateSpec struct {
	DpConfigVersion string          `json:"dpConfigVersion,omitempty"`
	Interfaces      SriovInterfaces `json:"interfaces,omitempty"`
}

type SriovInterface struct {
	PciAddress string    `json:"pciAddress"`
	NumVfs     int       `json:"numVfs,omitempty"`
	Mtu        int       `json:"mtu,omitempty"`
	Name       string    `json:"name,omitempty"`
	LinkType   string    `json:"linkType,omitempty"`
	VfGroups   []VfGroup `json:"vfGroups,omitempty"`
}

type VfGroup struct {
	ResourceName string `json:"resourceName,omitempty"`
	DeviceType   string `json:"deviceType,omitempty"`
	VfRange      string `json:"vfRange,omitempty"`
	PolicyName   string `json:"policyName,omitempty"`
}

type SriovInterfaces []SriovInterface

type InterfaceExt struct {
	Name       string            `json:"name,omitempty"`
	Mac        string            `json:"mac,omitempty"`
	Driver     string            `json:"driver,omitempty"`
	PciAddress string            `json:"pciAddress"`
	Vendor     string            `json:"vendor,omitempty"`
	DeviceID   string            `json:"deviceID,omitempty"`
	Mtu        int               `json:"mtu,omitempty"`
	NumVfs     int               `json:"numVfs,omitempty"`
	LinkSpeed  string            `json:"linkSpeed,omitempty"`
	LinkType   string            `json:"linkType,omitempty"`
	TotalVfs   int               `json:"totalvfs,omitempty"`
	VFs        []VirtualFunction `json:"Vfs,omitempty"`
}
type InterfaceExts []InterfaceExt

type VirtualFunction struct {
	Name       string `json:"name,omitempty"`
	Mac        string `json:"mac,omitempty"`
	Assigned   string `json:"assigned,omitempty"`
	Driver     string `json:"driver,omitempty"`
	PciAddress string `json:"pciAddress"`
	Vendor     string `json:"vendor,omitempty"`
	DeviceID   string `json:"deviceID,omitempty"`
	Vlan       int    `json:"Vlan,omitempty"`
	Mtu        int    `json:"mtu,omitempty"`
	VfID       int    `json:"vfID"`
}

/////////////////////////////////////////////
