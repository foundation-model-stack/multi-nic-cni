/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MultiNicNetworkSpec defines the desired state of MultiNicNetwork
// MasterNetAddrs is network addresses of NIC members in the pool
// Subnet is global subnet, default: 172.30.0.0/16
// IPAM is ipam specification
// MainPlugin is plugin specification
// Policy is general policy of the pool
type MultiNicNetworkSpec struct {
	MasterNetAddrs []string         `json:"masterNets,omitempty"`
	Subnet         string           `json:"subnet,omitempty"`
	IPAM           string           `json:"ipam"`
	IsMultiNICIPAM bool             `json:"multiNICIPAM,omitempty"`
	MainPlugin     PluginSpec       `json:"plugin"`
	Policy         AttachmentPolicy `json:"attachPolicy,omitempty"`
	Namespaces     []string         `json:"namespaces,omitempty"`
}

// reference: github.com/containernetworking/cni/pkg/types
type PluginSpec struct {
	CNIVersion   string            `json:"cniVersion"`
	Type         string            `json:"type"`
	Capabilities map[string]bool   `json:"capabilities,omitempty"`
	DNS          DNS               `json:"dns,omitempty"`
	CNIArgs      map[string]string `json:"args,omitempty"`
}

// reference: github.com/containernetworking/cni/pkg/types
type DNS struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	Search      []string `json:"search,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// AssignmentPolicy defines the policy to select the NICs from the pool
// Strategy is one of None, CostOpt, PerfOpt, QoSClass
// Target is target bandwidth in a format (d+)Gbps, (d+)Mbps, (d+)Kbps
// required for CostOpt and PerfOpt
type AttachmentPolicy struct {
	Strategy string `json:"strategy"`
	Target   string `json:"target,omitempty"`
}

// MultiNicNetworkStatus defines the observed state of MultiNicNetwork
type MultiNicNetworkStatus struct {
	// Definitions lists NetworkAttachmentDefinition created by the MultiNicNetwork
	Definitions []string `json:"defs"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// MultiNicNetwork is the Schema for the multinicnetworks API
type MultiNicNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiNicNetworkSpec   `json:"spec,omitempty"`
	Status MultiNicNetworkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MultiNicNetworkList contains a list of MultiNicNetwork
type MultiNicNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiNicNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MultiNicNetwork{}, &MultiNicNetworkList{})
}
