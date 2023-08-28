/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin

import (
	"encoding/json"

	"github.com/containernetworking/cni/pkg/types"
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"k8s.io/client-go/rest"
)

const (
	AWS_IPVLAN_TYPE = "aws-ipvlan"
)

type AwsVpcCNIPlugin struct {
}

type AWSIPVLANNetConf struct {
	types.NetConf
	PrimaryIP map[string]interface{} `json:"primaryIP"`
	PodIP     string                 `json:"podIP"`
	Master    string                 `json:"master"`
	Mode      string                 `json:"mode"`
	MTU       int                    `json:"mtu"`
}

func (p *AwsVpcCNIPlugin) Init(config *rest.Config) error {
	return nil
}

func (p *AwsVpcCNIPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &AWSIPVLANNetConf{}
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = AWS_IPVLAN_TYPE
	var primaryIP map[string]interface{}
	err := json.Unmarshal([]byte(args["primaryIP"]), &primaryIP)
	if err == nil {
		conf.PrimaryIP = primaryIP
	}
	conf.PodIP = args["podIP"]
	conf.Master = args["master"]
	conf.Mode = args["mode"]
	val, err := getInt(args, "mtu")
	if err == nil {
		conf.MTU = val
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
