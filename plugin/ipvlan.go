/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/go-logr/logr"
	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"k8s.io/client-go/rest"
)

const (
	IPVLAN_TYPE = "ipvlan"
)

type IPVLANPlugin struct {
	Log logr.Logger
}

type IPVLANTypeNetConf struct {
	types.NetConf
	Master string `json:"master"`
	Mode   string `json:"mode"`
	MTU    int    `json:"mtu"`
}

func (p *IPVLANPlugin) Init(config *rest.Config, logger logr.Logger) error {
	return nil
}

func (p *IPVLANPlugin) GetConfig(net netcogadvisoriov1.MultiNicNetwork, hifList map[string]netcogadvisoriov1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &IPVLANTypeNetConf{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, conf)
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = IPVLAN_TYPE
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

func (p *IPVLANPlugin) CleanUp(net netcogadvisoriov1.MultiNicNetwork) error {
	return nil
}
