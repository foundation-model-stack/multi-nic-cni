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

type InterfaceInfoType struct {
	InterfaceName string `json:"interfaceName"`
	NetAddress    string `json:"netAddress"`
	HostIP        string `json:"hostIP"`
	Vendor        string `json:"vendor"`
	Product       string `json:"product"`
	PciAddress    string `json:"pciAddress"`
}

func (i InterfaceInfoType) Equal(cmp InterfaceInfoType) bool {
	return i.InterfaceName == cmp.InterfaceName && i.NetAddress == cmp.NetAddress && i.HostIP == cmp.HostIP
}

// HostInterfaceSpec defines the desired state of HostInterface
type HostInterfaceSpec struct {
	HostName   string              `json:"hostName"`
	Interfaces []InterfaceInfoType `json:"interfaces"`
}

// HostInterfaceStatus defines the observed state of HostInterface
type HostInterfaceStatus struct {
	Stat LinkStat `json:"stat"`
}

type LinkStat struct {
	InterfaceName string `json:"interfaceName"`
	TxRate        int    `json:"txRate"`
	RxRate        int    `json:"rxRate"`
	TxDropRate    int    `json:"txDropRate"`
	RxDropRate    int    `json:"rxDropRate"`
	LastTx        int    `json:"lastTx"`
	LastRx        int    `json:"lastRx"`
	LastTxDrop    int    `json:"lastTxDrop"`
	LastRxDrop    int    `json:"lastRxDrop"`
	LastTimeStamp int64  `json:"lastTimestamp"`
	UsedCount     int    `json:"count"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// HostInterface is the Schema for the hostinterfaces API
type HostInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostInterfaceSpec   `json:"spec,omitempty"`
	Status HostInterfaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HostInterfaceList contains a list of HostInterface
type HostInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostInterface{}, &HostInterfaceList{})
}
