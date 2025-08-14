/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type PluginConfig struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Subnet         string   `json:"subnet"`
	MasterNetAddrs []string `json:"masterNets"`
	HostBlock      int      `json:"hostBlock"`
	InterfaceBlock int      `json:"interfaceBlock"`
	ExcludeCIDRs   []string `json:"excludeCIDRs,omitempty"`
	VlanMode       string   `json:"vlanMode,omitempty"`
}

type HostInterfaceInfo struct {
	HostIndex     int    `json:"hostIndex"`
	HostName      string `json:"hostName"`
	InterfaceName string `json:"interfaceName"`
	HostIP        string `json:"hostIP"`
	PodCIDR       string `json:"podCIDR"`
	IPPool        string `json:"ippool,omitempty"`
}

type CIDREntry struct {
	NetAddress     string              `json:"netAddress"`
	InterfaceIndex int                 `json:"interfaceIndex"`
	VlanCIDR       string              `json:"vlanCIDR"`
	Hosts          []HostInterfaceInfo `json:"hosts"`
}

// CIDRSpec defines the desired state of CIDR
type CIDRSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Config PluginConfig `json:"config"`
	CIDRs  []CIDREntry  `json:"cidr"`
}

// CIDRStatus defines the observed state of CIDR
type CIDRStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// CIDR is the Schema for the cidrs API
type CIDR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CIDRSpec   `json:"spec,omitempty"`
	Status CIDRStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CIDRList contains a list of CIDR
type CIDRList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CIDR `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CIDR{}, &CIDRList{})
}
