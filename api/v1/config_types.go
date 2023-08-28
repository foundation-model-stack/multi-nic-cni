/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type DaemonSpec struct {
	NodeSelector    map[string]string           `json:"nodeSelector,omitempty"`
	Image           string                      `json:"image"`
	ImagePullSecret string                      `json:"imagePullSecretName,omitempty"`
	ImagePullPolicy string                      `json:"imagePullPolicy,omitempty"`
	SecurityContext *corev1.SecurityContext     `json:"securityContext,omitempty"`
	Env             []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	HostPathMounts  []HostPathMount             `json:"mounts,omitempty"`
	DaemonPort      int                         `json:"port"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations     []corev1.Toleration         `json:"tolerations,omitempty" protobuf:"bytes,22,opt,name=tolerations"`
}

type HostPathMount struct {
	Name        string `json:"name"`
	PodCNIPath  string `json:"podpath"`
	HostCNIPath string `json:"hostpath"`
}

// ConfigSpec defines the desired state of Config
type ConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CNIType                string     `json:"cniType"`
	IPAMType               string     `json:"ipamType"`
	Daemon                 DaemonSpec `json:"daemon"`
	JoinPath               string     `json:"joinPath"`
	InterfacePath          string     `json:"getInterfacePath"`
	AddRoutePath           string     `json:"addRoutePath,omitempty"`
	DeleteRoutePath        string     `json:"deleteRoutePath,omitempty"`
	UrgentReconcileSeconds int        `json:"urgentReconcileSeconds,omitempty"`
	NormalReconcileMinutes int        `json:"normalReconcileMinutes,omitempty"`
	LongReconcileMinutes   int        `json:"longReconcileMinutes,omitempty"`
	ContextTimeoutMinutes  int        `json:"contextTimeoutMinutes,omitempty"`
	LogLevel               int        `json:"logLevel,omitempty"`
}

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Config is the Schema for the configs API
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
