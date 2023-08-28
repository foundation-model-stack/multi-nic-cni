/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/containernetworking/plugins/pkg/utils"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

const (
	logFilePath = "/var/log/multi-nic-aws-ipvlan.log"
)

// NetConf defines general config for multi-nic-cni
type NetConf struct {
	types.NetConf
	PrimaryIP string `json:"primaryIP"`
	PodIP     string `json:"podIP"`
	Master    string `json:"master"`
	Mode      string `json:"mode"`
	MTU       int    `json:"mtu"`
}

type IPVLANTypeNetConf struct {
	types.NetConf
	IPAM   IPAMConfig `json:"ipam"`
	Master string     `json:"master"`
	Mode   string     `json:"mode"`
	MTU    int        `json:"mtu"`
}

// static IPAM
type Address struct {
	AddressStr string `json:"address"`
	Gateway    net.IP `json:"gateway,omitempty"`
	Address    net.IPNet
	Version    string
}

type IPAMConfig struct {
	Name      string
	Type      string    `json:"type"`
	Addresses []Address `json:"addresses,omitempty"`
}

func main() {
	utils.InitializeLogger(logFilePath)
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("multi-nic"))
}

func cmdAdd(args *skel.CmdArgs) error {
	// load general NetConf and get deviceType
	n, ipvlanConfig, err := loadConf(args, false)
	if err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}
	utils.Logger.Debug(fmt.Sprintf("Received an ADD request for: conf=%v", n))
	_, err = AssignIP(n.PrimaryIP, n.PodIP)
	if err != nil {
		return fmt.Errorf("Failed to AssignIP: %v", err)
	}
	// call ipvlan
	command := "ADD"
	executeResult, err := execIPVLAN(command, ipvlanConfig, args, args.IfName, true)
	defer func() {
		if err != nil {
			unassignErr := UnassignIP(n.PrimaryIP, n.PodIP)
			if unassignErr != nil {
				utils.Logger.Debug(fmt.Sprintf("Unassign %s from %s failed: %v", n.PodIP, n.PrimaryIP, unassignErr))
			}
		}
	}()
	if err != nil {
		return err
	}
	return types.PrintResult(executeResult, n.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}
	n, ipvlanConfig, err := loadConf(args, false)
	if err != nil {
		return fmt.Errorf("fail to load conf: %v", err)
	}
	utils.Logger.Debug(fmt.Sprintf("Received an DEL request for: conf=%v", n))
	err = UnassignIP(n.PrimaryIP, n.PodIP)
	if err != nil {
		return fmt.Errorf("Failed to UnassignIP: %v", err)
	}
	// call ipvlan
	command := "DEL"
	_, err = execIPVLAN(command, ipvlanConfig, args, args.IfName, false)
	if err != nil {
		return err
	}
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}
	_, ipvlanConfig, err := loadConf(args, false)
	if err != nil {
		return fmt.Errorf("fail to load conf: %v", err)
	}
	// call ipvlan
	command := "CHECK"
	_, err = execIPVLAN(command, ipvlanConfig, args, args.IfName, false)
	if err != nil {
		return err
	}
	return nil
}

// loadConf unmarshal NetConf and return with dev type
func loadConf(args *skel.CmdArgs, check bool) (*NetConf, *IPVLANTypeNetConf, error) {
	n := &NetConf{}
	if err := json.Unmarshal(args.StdinData, n); err != nil {
		return nil, nil, err
	}
	ipvlanConfig := &IPVLANTypeNetConf{}
	if err := json.Unmarshal(args.StdinData, ipvlanConfig); err != nil {
		return nil, nil, err
	}
	return n, ipvlanConfig, nil
}

var defaultExec = &invoke.DefaultExec{
	RawExec: &invoke.RawExec{Stderr: os.Stderr},
}

func execIPVLAN(command string, ipvlanConfig *IPVLANTypeNetConf, args *skel.CmdArgs, ifName string, isAdd bool) (*current.Result, error) {
	confBytes, err := json.Marshal(ipvlanConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ipvlanConfig: %v", err)
	}
	cniPath := os.Getenv("CNI_PATH")
	singleNicArgs := &invoke.Args{
		Command:       command,
		ContainerID:   args.ContainerID,
		NetNS:         args.Netns,
		IfName:        ifName,
		PluginArgsStr: args.Args,
		Path:          cniPath,
	}
	paths := filepath.SplitList(cniPath)
	pluginPath, err := defaultExec.FindInPath("ipvlan", paths)
	if err != nil {
		return nil, err
	}

	if isAdd {
		r, err := invoke.ExecPluginWithResult(context.TODO(), pluginPath, confBytes, singleNicArgs, defaultExec)
		if err != nil {
			return nil, err
		}
		return current.NewResultFromResult(r)
	} else {
		err = invoke.ExecPluginWithoutResult(context.TODO(), pluginPath, confBytes, singleNicArgs, defaultExec)
		return nil, err
	}
}
