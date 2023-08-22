/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/vishvananda/netlink"

	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

const (
	IPVLAN_MODE_CONFIG      = "l3"
	DEFAULT_SUBNET          = "172.30.0.0/16"
	DEFAULT_HOST_BLOCK      = 8
	DEFAULT_INTERFACE_BLOCK = 2
	logFilePath             = "/var/log/multi-nic-cni.log"
)

// NetConf defines general config for multi-nic-cni
type NetConf struct {
	types.NetConf
	MainPlugin     map[string]interface{} `json:"plugin"`
	Subnet         string                 `json:"subnet"`
	MasterNetAddrs []string               `json:"masterNets"`
	DeviceIDs      []string               `json:"deviceIDs"`
	Masters        []string               `json:"masters"`
	IsMultiNICIPAM bool                   `json:"multiNICIPAM,omitempty"`
	DaemonIP       string                 `json:"daemonIP"`
	DaemonPort     int                    `json:"daemonPort"`
	Args           *struct {
		NicSet *NicArgs `json:"cni,omitempty"`
	} `json:"args"`
}

// NicArgs defines additional specification in pod annotation
type NicArgs struct {
	NumOfInterfaces int      `json:"nics,omitempty"`
	InterfaceNames  []string `json:"masters,omitempty"`
	Target          string   `json:"target,omitempty"`
	DevClass        string   `json:"class,omitempty"`
}

func main() {
	utils.InitializeLogger(logFilePath)
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("multi-nic"))
}

func cmdAdd(args *skel.CmdArgs) error {
	// load general NetConf and get deviceType
	n, deviceType, err := loadConf(args)
	if err != nil {
		return fmt.Errorf("failed to load netconf: %v", err)
	}
	utils.Logger.Debug(fmt.Sprintf("Received an ADD request for: conf=%v", n))

	// open specified network namespace
	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("failed to open netns %q: %v", args.Netns, err))
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()

	// request ipam
	var result *current.Result
	haveResult := false
	// parse previous result
	if n.NetConf.RawPrevResult != nil {
		if err = version.ParsePrevResult(&n.NetConf); err != nil {
			utils.Logger.Debug(fmt.Sprintf("could not parse prevResult: %v", err))
			return fmt.Errorf("could not parse prevResult: %v", err)
		}

		result, err = current.NewResultFromResult(n.NetConf.PrevResult)
		if err != nil {
			utils.Logger.Debug(fmt.Sprintf("could not convert result to current version: %v", err))
			return fmt.Errorf("could not convert result to current version: %v", err)
		}

		if len(result.IPs) > 0 {
			haveResult = true
		}

	} else {
		result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
	}

	if !haveResult && n.IsMultiNICIPAM {
		utils.Logger.Debug(fmt.Sprintf("use multi-nic-ipam: %s", n.IPAM.Type))
		// run the IPAM plugin and get back the config to apply
		injectedStdIn := injectMaster(args.StdinData, n.MasterNetAddrs, n.Masters, n.DeviceIDs)
		r, err := ipam.ExecAdd(n.IPAM.Type, injectedStdIn)
		if err != nil {
			return fmt.Errorf("IPAM ExecAdd: %v, %s", err, string(injectedStdIn))
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
	var multiPathRoutes map[string][]*netlink.NexthopInfo
	switch deviceType {
	case "ipvlan":
		confBytesArray, multiPathRoutes, err = loadIPVANConf(args.StdinData, args.IfName, n, result.IPs)
	case "sriov":
		confBytesArray, multiPathRoutes, err = loadSRIOVConf(args.StdinData, args.IfName, n, result.IPs)
	case "aws-ipvlan":
		confBytesArray, multiPathRoutes, err = loadAWSCNIConf(args.StdinData, args.IfName, n, result.IPs)
	case "host-device":
		confBytesArray, multiPathRoutes, err = loadHostDeviceConf(args.StdinData, args.IfName, n, result.IPs)
	default:
		err = fmt.Errorf("unsupported device type: %s", deviceType)
	}
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("Fail loading %v: %v", string(args.StdinData), err))
		return err
	}
	if len(confBytesArray) == 0 {
		utils.Logger.Debug(fmt.Sprintf("zero config: %v (%d)", string(args.StdinData), len(n.Masters)))
		return fmt.Errorf("zero config %v", n)
	}

	ips := []*current.IPConfig{}
	for index, confBytes := range confBytesArray {
		command := "ADD"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		utils.Logger.Debug(fmt.Sprintf("Exec %s %s: %s", command, ifName, string(confBytes)))
		executeResult, err := execPlugin(deviceType, command, confBytes, args, ifName, true)
		if err != nil {
			utils.Logger.Debug(fmt.Sprintf("Fail execPlugin %v: %v", string(confBytes), err))
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
			addMultiPathRoutes(link, multiPathRoutes)
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
	utils.Logger.Debug(fmt.Sprintf("Result: %v", result))
	return types.PrintResult(result, n.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}
	ips := []*current.IPConfig{}
	n, deviceType, err := loadConf(args)
	if err != nil && n == nil {
		utils.Logger.Debug(fmt.Sprintf("fail to load conf: %v", err))
		return nil
	}
	utils.Logger.Debug(fmt.Sprintf("Received an DEL request for: conf=%v", n))
	// On chained invocation, IPAM block can be empty
	if n.IPAM.Type != "" {
		injectedStdIn := injectMaster(args.StdinData, n.MasterNetAddrs, n.Masters, n.DeviceIDs)
		if n.IPAM.Type != "multi-nic-ipam" {
			err = ipam.ExecDel(n.IPAM.Type, injectedStdIn)
			utils.Logger.Debug(fmt.Sprintf("Failed ipam.ExecDel %s: %v", err, string(injectedStdIn)))
		} else {
			r, err := ipam.ExecDelWithResult(n.IPAM.Type, injectedStdIn)
			if err != nil {
				utils.Logger.Debug(fmt.Sprintf("Failed ipam.ExecDel %s: %v", err, string(injectedStdIn)))
			} else {
				if r != nil {
					executeResult, err := current.NewResultFromResult(r)
					if err != nil {
						utils.Logger.Debug(fmt.Sprintf("Failed to parse result %v: %v", r, err))
					}
					ips = executeResult.IPs
				} else {
					utils.Logger.Debug(fmt.Sprintf("No previous result: %v", r))
				}
			}
		}
	}

	// get device config and apply
	confBytesArray := [][]byte{}
	var multiPathRoutes map[string][]*netlink.NexthopInfo
	switch deviceType {
	case "ipvlan":
		confBytesArray, multiPathRoutes, err = loadIPVANConf(args.StdinData, args.IfName, n, ips)
	case "sriov":
		confBytesArray, multiPathRoutes, err = loadSRIOVConf(args.StdinData, args.IfName, n, ips)
	case "aws-ipvlan":
		confBytesArray, multiPathRoutes, err = loadAWSCNIConf(args.StdinData, args.IfName, n, ips)
	case "host-device":
		confBytesArray, multiPathRoutes, err = loadHostDeviceConf(args.StdinData, args.IfName, n, ips)
	default:
		err = fmt.Errorf("unsupported device type: %s", deviceType)
	}
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("Fail loading %v: %v", string(args.StdinData), err))
	}
	if len(confBytesArray) == 0 {
		utils.Logger.Debug(fmt.Sprintf("zero config on cmdDel: %v (%d)", string(args.StdinData), len(n.Masters)))
	}

	// open specified network namespace
	netns, netNsErr := ns.GetNS(args.Netns)
	if netNsErr != nil {
		utils.Logger.Debug(fmt.Sprintf("failed to open netns %q: %v", args.Netns, err))
	} else {
		defer netns.Close()
	}

	for index, confBytes := range confBytesArray {
		command := "DEL"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		if netNsErr == nil {
			err = netns.Do(func(_ ns.NetNS) error {
				link, err := net.InterfaceByName(ifName)
				if err != nil {
					return err
				}
				delMultiPathRoutes(link, multiPathRoutes)
				return nil
			})
		}

		utils.Logger.Debug(fmt.Sprintf("Exec %s %s: %s", command, ifName, string(confBytes)))
		_, err := execPlugin(deviceType, command, confBytes, args, ifName, false)
		if err != nil {
			utils.Logger.Debug(fmt.Sprintf("Fail execPlugin %v: %v", string(confBytes), err))
		}
	}
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	if args.Netns == "" {
		return nil
	}

	n, deviceType, err := loadConf(args)
	if err != nil && n == nil {
		utils.Logger.Debug(fmt.Sprintf("fail to load conf: %v", err))
		return nil
	}

	var result *current.Result
	// parse previous result
	if n.NetConf.RawPrevResult != nil {
		if err = version.ParsePrevResult(&n.NetConf); err != nil {
			utils.Logger.Debug(fmt.Sprintf("could not parse prevResult: %v", err))
		} else {
			result, err = current.NewResultFromResult(n.NetConf.PrevResult)
			if err != nil {
				utils.Logger.Debug(fmt.Sprintf("could not convert result to current version: %v", err))
				result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
			}
		}
	}
	if result == nil {
		result = &current.Result{CNIVersion: current.ImplementedSpecVersion}
	}

	// get device config and apply
	confBytesArray := [][]byte{}
	switch deviceType {
	case "ipvlan":
		confBytesArray, _, err = loadIPVANConf(args.StdinData, args.IfName, n, result.IPs)
	case "sriov":
		confBytesArray, _, err = loadSRIOVConf(args.StdinData, args.IfName, n, result.IPs)
	case "aws-ipvlan":
		confBytesArray, _, err = loadAWSCNIConf(args.StdinData, args.IfName, n, result.IPs)
	case "host-device":
		confBytesArray, _, err = loadHostDeviceConf(args.StdinData, args.IfName, n, result.IPs)
	default:
		err = fmt.Errorf("unsupported device type: %s", deviceType)
	}
	if err != nil {
		utils.Logger.Debug(fmt.Sprintf("Fail loading %v: %v", string(args.StdinData), err))
	}
	if len(confBytesArray) == 0 {
		utils.Logger.Debug(fmt.Sprintf("zero config on cmdCheck: %v (%d)", string(args.StdinData), len(n.Masters)))
	}

	for index, confBytes := range confBytesArray {
		command := "CHECK"
		ifName := fmt.Sprintf("%s-%d", args.IfName, index)
		utils.Logger.Debug(fmt.Sprintf("Exec %s %s: %s", command, ifName, string(confBytes)))
		_, err := execPlugin(deviceType, command, confBytes, args, ifName, false)
		if err != nil {
			utils.Logger.Debug(fmt.Sprintf("Fail execPlugin %v: %v", string(confBytes), err))
		}
	}
	return nil
}

// loadConf unmarshal NetConf and return with dev type
func loadConf(args *skel.CmdArgs) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(args.StdinData, n); err != nil {
		return nil, "", err
	}
	deviceType := n.MainPlugin["type"].(string)
	if n.Subnet == "" {
		n.Subnet = DEFAULT_SUBNET
	}
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

	return n, deviceType, nil
}
