/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"

	"github.com/containernetworking/cni/pkg/types"
	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
)

const (
	MACVLAN_TYPE = "macvlan"
)

type MACVLANPlugin struct {
	Log logr.Logger
}

type MACVLANTypeNetConf struct {
	types.NetConf
	Mode string `json:"mode"`
	MTU  int    `json:"mtu"`
}

func (p *MACVLANPlugin) Init(config *rest.Config, logger logr.Logger) error {
	return nil
}

func (p *MACVLANPlugin) GetConfig(net netcogadvisoriov1.MultiNicNetwork, hifList map[string]netcogadvisoriov1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &MACVLANTypeNetConf{}
	argBytes, _ := json.Marshal(args)
	json.Unmarshal(argBytes, conf)
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = MACVLAN_TYPE
	val, err := getInt(args, "mtu")
	if err == nil {
		conf.MTU = val
	}
	confBytes, _ := json.Marshal(conf)
	return string(confBytes), make(map[string]string), nil
}

func (p *MACVLANPlugin) CleanUp(net netcogadvisoriov1.MultiNicNetwork) error {
	return nil
}
