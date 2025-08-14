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
	MACVLAN_TYPE = "macvlan"
)

type MACVLANPlugin struct {
}

type MACVLANTypeNetConf struct {
	types.NetConf
	Master string `json:"master"` // Name of the master interfce (e.g., eth0)
	Mode   string `json:"mode"`   // Mod of the macvlan interface (e.g., bridge, private)
	MTU    int    `json:"mtu"`
}

func (p *MACVLANPlugin) Init(config *rest.Config) error {
	return nil
}

func (p *MACVLANPlugin) GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error) {
	spec := net.Spec.MainPlugin
	args := spec.CNIArgs
	conf := &MACVLANTypeNetConf{}
	conf.CNIVersion = net.Spec.MainPlugin.CNIVersion
	conf.Type = MACVLAN_TYPE
	conf.Master = args["master"] // Populate the master field
	conf.Mode = args["mode"]
	val, err := getInt(args, "mtu")
	if err == nil {
		conf.MTU = val
	}
	confBytes, _ := json.Marshal(conf)
	return string(confBytes), make(map[string]string), nil
}

func (p *MACVLANPlugin) CleanUp(net multinicv1.MultiNicNetwork) error {
	return nil
}
