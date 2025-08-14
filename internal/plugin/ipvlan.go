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
	IPVLAN_TYPE = "ipvlan"
)

type IPVLANPlugin struct {
}

type IPVLANTypeNetConf struct {
	types.NetConf
	Master string `json:"master"`
	Mode   string `json:"mode"`
	MTU    int    `json:"mtu"`
}

func (p *IPVLANPlugin) Init(config *rest.Config) error {
	return nil
}

func (p *IPVLANPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &IPVLANTypeNetConf{}
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = IPVLAN_TYPE
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

func (p *IPVLANPlugin) CleanUp(net multinicv1.MultiNicNetwork) error {
	return nil
}
