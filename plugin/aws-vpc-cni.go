/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
)

const (
	AWS_VPC_CNI = "aws-vpc-cni"
)

type AwsVpcCNIPlugin struct {
	Log logr.Logger
}

type EnforcingMode string

const (
	EnforcingModeStrict   EnforcingMode = "strict"
	EnforcingModeStandard EnforcingMode = "standard"
)

type AwsVpcCNITypeNetConf struct {
	types.NetConf

	// VethPrefix is the prefix to use when constructing the host-side
	// veth device name. It should be no more than four characters, and
	// defaults to 'eni'.
	VethPrefix string `json:"vethPrefix"`

	// MTU for eth0
	MTU string `json:"mtu"`

	// PodSGEnforcingMode is the enforcing mode for Security groups for pods feature
	PodSGEnforcingMode EnforcingMode `json:"podSGEnforcingMode"`

	// Interface inside container to create
	IfName string `json:"ifName"`

	//MTU for Egress v4 interface
	EgressMTU int `json:"egressMtu"`

	Enabled string `json:"enabled"`

	RandomizeSNAT string `json:"randomizeSNAT"`

	// IP to use as SNAT target
	NodeIP net.IP `json:"nodeIP"`

	PluginLogFile        string `json:"pluginLogFile"`
	PluginLogLevel       string `json:"pluginLogLevel"`
	EgressPluginLogFile  string `json:"egressPluginLogFile"`
	EgressPluginLogLevel string `json:"egressPluginLogLevel"`
}

func (p *AwsVpcCNIPlugin) Init(config *rest.Config, logger logr.Logger) error {
	return nil
}

func (p *AwsVpcCNIPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &AwsVpcCNITypeNetConf{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, conf)
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = AWS_VPC_CNI
	val, err := getInt(args, "egressMtu")
	if err == nil {
		conf.EgressMTU = val
	}
	confBytes, err := json.Marshal(conf)
	if err != nil {
		return "", make(map[string]string), err
	}
	return string(confBytes), make(map[string]string), nil
}

func (p *AwsVpcCNIPlugin) CleanUp(net multinicv1.MultiNicNetwork) error {
	return nil
}
