

https://github.com/openshift/sriov-network-operator/blob/6bf39e30d2119d1b883e7855fbcccaf298de09ad/api/v1/sriovnetwork_types.go#L75
```go
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
```

ADD/DEL/CHECK
https://github.com/k8snetworkplumbingwg/sriov-cni/blob/v2.1.0/cmd/sriov/main.go

LoadConf
https://github.com/openshift/sriov-cni/blob/dfbc68063bb549910a5440d7c80e45a2519d12cc/pkg/config/config.go#L20

NetConfg
```go
// NetConf extends types.NetConf for sriov-cni
type NetConf struct {
	types.NetConf
	OrigVfState   VfState // Stores the original VF state as it was prior to any operations done during cmdAdd flow
	DPDKMode      bool
	Master        string
	MAC           string
	Vlan          *int   `json:"vlan"`
	VlanQoS       *int   `json:"vlanQoS"`
	DeviceID      string `json:"deviceID"` // PCI address of a VF in valid sysfs format
	VFID          int
	ContIFNames   string // VF names after in the container; used during deletion
	MinTxRate     *int   `json:"min_tx_rate"`          // Mbps, 0 = disable rate limiting
	MaxTxRate     *int   `json:"max_tx_rate"`          // Mbps, 0 = disable rate limiting
	SpoofChk      string `json:"spoofchk,omitempty"`   // on|off
	Trust         string `json:"trust,omitempty"`      // on|off
	LinkState     string `json:"link_state,omitempty"` // auto|enable|disable
	RuntimeConfig struct {
		Mac string `json:"mac,omitempty"`
	} `json:"runtimeConfig,omitempty"`
}
```