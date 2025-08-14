/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// SriovNetworkNodeState represents the node state for SR-IOV networks
type SriovNetworkNodeState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodeStateSpec   `json:"spec,omitempty"`
	Status SriovNetworkNodeStateStatus `json:"status,omitempty"`
}

// SriovNetworkNodeStateSpec defines the desired state of SriovNetworkNodeState
type SriovNetworkNodeStateSpec struct {
	// NodeSelector selects the nodes to be configured
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// SriovNetworkNodeStateStatus defines the observed state of SriovNetworkNodeState
type SriovNetworkNodeStateStatus struct {
	// Interfaces represents the list of SR-IOV capable interfaces on the node
	Interfaces []InterfaceExt `json:"interfaces,omitempty"`
}

// InterfaceExt represents an SR-IOV interface with extended information
type InterfaceExt struct {
	// Name is the name of the interface
	Name string `json:"name,omitempty"`
	// PciAddress is the PCI address of the interface
	PciAddress string `json:"pciAddress,omitempty"`
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
