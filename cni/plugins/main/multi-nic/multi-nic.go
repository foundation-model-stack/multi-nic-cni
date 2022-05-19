/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package main

import (
	"os"
	"net"
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"


)

const (
	IPVLAN_MODE_CONFIG      = "l3"
	DEFAULT_SUBNET          = "172.30.0.0/16"
	DEFAULT_HOST_BLOCK      = 8
	DEFAULT_INTERFACE_BLOCK = 2
)


// NetConf defines general config for multi-nic-cni
type NetConf struct {
	types.NetConf
	MainPlugin     map[string]interface{}  `json:"plugin"`
	Subnet         string                  `json:"subnet"`
	MasterNetAddrs []string                `json:"masterNets"`
	DeviceIDs      []string	               `json:"deviceIDs"`
	Masters        []string	               `json:"masters"`
	IsMultiNICIPAM bool                    `json:"multiNICIPAM,omitempty"`
	DaemonIP       string                  `json:"daemonIP"`
	DaemonPort     int                     `json:"daemonPort"`
	Args *struct {
		NicSet *NicArgs `json:"cni,omitempty"`
	} `json:"args"`
}

// NicArgs defines additional specification in pod annotation
type NicArgs struct {
	NumOfInterfaces int `json:"nics,omitempty"`
	InterfaceNames  []string `json:"masters,omitempty"`
	Target          string `json:"target,omitempty"`
	DevClass        string `json:"class,omitempty"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("multi-nic"))
}


func cmdAdd(args *skel.CmdArgs) error {
	// load general NetConf and get deviceType
	n, deviceType, err := loadConf(args, false)
	if err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}

	// open specified network namespace
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// request ipam
	var result *current.Result
	haveResult := false
	// parse previous result
	if n.NetConf.RawPrevResult != nil {
		if err = version.ParsePrevResult(&n.NetConf); err != nil {
			return fmt.Errorf("could not parse prevResult: %v", err)
		}

		result, err = current.NewResultFromResult(n.NetConf.PrevResult)
		if err != nil {
			return fmt.Errorf("could not convert result to current version: %v", err)
		}

		if len(result.IPs) > 0 {
			haveResult = true
		}

	} else {
		result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
	}

	if !haveResult && n.IsMultiNICIPAM {
		// run the IPAM plugin and get back the config to apply
		injectedStdIn := injectMaster(args.StdinData, n.MasterNetAddrs, n.Masters, n.DeviceIDs)
		r, err := ipam.ExecAdd(n.IPAM.Type, injectedStdIn)
		if err != nil {
			return fmt.Errorf("IPAM ExecAdd: %v, %s",err, string(injectedStdIn))
		}

		// Invoke ipam del if err to avoid ip leak
		defer func() {
			if err != nil {
				ipam.ExecDel(n.IPAM.Type, args.StdinData)
			}
		}()

		// Convert whatever the IPAM result was into the current Result type
		result, err = current.NewResultFromResult(r)
		if err != nil {
			return err
		}

		if len(result.IPs) == 0 {
			return fmt.Errorf("IPAM plugin returned missing IP config %v", string(injectedStdIn))
		}
	}


	// get device config and apply
	confBytesArray := [][]byte{}
	switch deviceType {
	case "ipvlan":
		confBytesArray, err = loadIPVANConf(args.StdinData, args.IfName, n, result.IPs)

		if err != nil {
			return err
		}

		if len(confBytesArray) == 0 {
			return fmt.Errorf("zero config %v", n)
		}
	case "sriov":
		confBytesArray, err = loadSRIOVConf(args.StdinData, args.IfName, n, result.IPs)

		if err != nil {
			return err
		}

		if len(confBytesArray) == 0 {
			return fmt.Errorf("zero config %v", n)
		}
	default:
		return fmt.Errorf("unsupported device type: %s", deviceType)
	}

	ips := []*current.IPConfig{}
	for index, confBytes := range confBytesArray {
		command := "ADD"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		executeResult, err := execPlugin(deviceType, command, confBytes, args, ifName, true)
		if err != nil {
			return err
		}
		interfaceItem := &current.Interface{}
		interfaceItem.Name = ifName
		
		err = netns.Do(func(_ ns.NetNS) error {
			link, err := net.InterfaceByName(ifName)
			if err != nil {
				return err
			}
			interfaceItem.Mac = link.HardwareAddr.String()
			interfaceItem.Sandbox = netns.Path()
			return nil
		})
		result.Interfaces = append(result.Interfaces, interfaceItem)
		if len(executeResult.IPs) > 0 {
			ipConf := executeResult.IPs[0]
			ipConf.Interface = current.Int(index)
			ips = append(ips, ipConf)
		}
	}
	result.IPs = ips
	return types.PrintResult(result, n.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}

	n, deviceType, err := loadConf(args, false)
	if err != nil {
		return fmt.Errorf("fail to load conf: %v", err)
	}
	
	// On chained invocation, IPAM block can be empty
	if n.IPAM.Type != "" {
		injectedStdIn := injectMaster(args.StdinData, n.MasterNetAddrs, n.Masters, n.DeviceIDs)
		err = ipam.ExecDel(n.IPAM.Type, injectedStdIn)
		if err != nil {
			return err
		}
	}

	var result *current.Result
	// parse previous result
	if n.NetConf.RawPrevResult != nil {
		if err = version.ParsePrevResult(&n.NetConf); err != nil {
			return fmt.Errorf("could not parse prevResult: %v", err)
		}

		result, err = current.NewResultFromResult(n.NetConf.PrevResult)
		if err != nil {
			return fmt.Errorf("could not convert result to current version: %v", err)
		}
	} else {
		result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
	}

	// get device config and apply
	confBytesArray := [][]byte{}
	switch(deviceType) {
	case "ipvlan":
		confBytesArray, err = loadIPVANConf(args.StdinData, args.IfName, n, result.IPs)
		if err != nil {
			return err
		}
	case "sriov":
		confBytesArray, err = loadSRIOVConf(args.StdinData, args.IfName, n, result.IPs)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported device type: %s", deviceType)
	}

	for index, confBytes := range confBytesArray {
		command := "DEL"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		_, err := execPlugin(deviceType, command, confBytes, args, ifName, false)
		if err != nil {
			return fmt.Errorf("%s, %v", string(confBytes), err)
		}
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}

	n, deviceType, err := loadConf(args, true)
	if err != nil {
		return fmt.Errorf("fail to load conf")
	}
	
	var result *current.Result
	// parse previous result
	if n.NetConf.RawPrevResult != nil {
		if err = version.ParsePrevResult(&n.NetConf); err != nil {
			return fmt.Errorf("could not parse prevResult: %v", err)
		}

		result, err = current.NewResultFromResult(n.NetConf.PrevResult)
		if err != nil {
			return fmt.Errorf("could not convert result to current version: %v", err)
		}
	} else {
		result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
	}


	// get device config and apply
	confBytesArray := [][]byte{}
	switch(deviceType) {
	case "ipvlan":
		confBytesArray, err = loadIPVANConf(args.StdinData, args.IfName, n, result.IPs)
		if err != nil {
			return err
		}

	case "sriov":
		confBytesArray, err = loadSRIOVConf(args.StdinData, args.IfName, n, result.IPs)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported device type: %s", deviceType)
	}


	for index, confBytes := range confBytesArray {
		command := "CHECK"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		_, err := execPlugin(deviceType, command, confBytes, args, ifName, false)
		if err != nil {
			return err
		}
	}

	return nil
}


// loadConf unmarshal NetConf and return with dev type
func loadConf(args *skel.CmdArgs, check bool) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(args.StdinData, n); err != nil {
		return nil, "", err
	}
	deviceType := n.MainPlugin["type"].(string)
	if n.Subnet == "" {
		n.Subnet = DEFAULT_SUBNET
	}
	if !check {
		// select NICs
		hostName, err := os.Hostname()
		if err != nil {
			return n, deviceType, err
		}
		podName, podNamespace := getPodInfo(args.Args)
		
		nicSet := NicArgs{}
		// check if user defined number of interfaces or specific set of interface names in the annotation
		if n.Args != nil && n.Args.NicSet != nil {
			nicSet = *n.Args.NicSet
		}
		selectResponse, err := selectNICs(n.DaemonIP, n.DaemonPort, podName, podNamespace, hostName, n.Name, nicSet, n.MasterNetAddrs)
		if err != nil {
			return n, deviceType, err
		}
		n.Masters = selectResponse.Masters
		n.DeviceIDs = selectResponse.DeviceIDs
	}
	return n, deviceType, nil
}

