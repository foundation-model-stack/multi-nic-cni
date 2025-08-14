/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Allocation struct {
	Pod       string `json:"pod"`
	Namespace string `json:"namespace"`
	Index     int    `json:"index"`
	Address   string `json:"address"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IPPoolSpec defines the desired state of IPPool
type IPPoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of IPPool. Edit ippool_types.go to remove/update
	PodCIDR          string       `json:"podCIDR"`
	VlanCIDR         string       `json:"vlanCIDR"`
	NetAttachDefName string       `json:"netAttachDef"`
	HostName         string       `json:"hostName"`
	InterfaceName    string       `json:"interfaceName"`
	Excludes         []string     `json:"excludes"`
	Allocations      []Allocation `json:"allocations"`
}

// IPPoolStatus defines the observed state of IPPool
type IPPoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// IPPool is the Schema for the ippools API
type IPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPPoolSpec   `json:"spec,omitempty"`
	Status IPPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IPPoolList contains a list of IPPool
type IPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPPool{}, &IPPoolList{})
}
