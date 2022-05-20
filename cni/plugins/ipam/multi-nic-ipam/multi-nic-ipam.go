/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"os"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
)

const (
	IPVLAN_MODE_CONFIG      = "l3"
	DEFAULT_SUBNET          = "172.30.0.0/16"
	DEFAULT_HOST_BLOCK      = 8
	DEFAULT_INTERFACE_BLOCK = 2
)


// The top-level network config - IPAM plugins are passed the full configuration
// of the calling plugin, not just the IPAM section.

type Net struct {
	types.NetConf
	Subnet         string      `json:"subnet"`
	MasterNetAddrs []string    `json:"masterNets"`
	Masters        []string	   `json:"masters"`
	IPAM           *IPAMConfig `json:"ipam"`
}


type IPAMConfig struct {
	Name           string
	Type           string         `json:"type"`
	DaemonIP       string         `json:"daemonIP"`
	DaemonPort     int            `json:"daemonPort"`
	HostBlock      int            `json:"hostBlock"`
	InterfaceBlock int            `json:"interfaceBlock"`
	ExcludeCIDRs   []string       `json:"excludeCIDRs"`
	Routes         []*types.Route `json:"routes"`
	DNS            types.DNS      `json:"dns"`
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("multi-nic-ipam"))
}

func getNetAddress(v *net.IPNet) string {
	blockSize := strings.Split(v.String(), "/")[1]
	ip := v.IP.Mask(v.Mask).String()
	return ip + "/" + blockSize
}

// return container interface name from net address
func getInterfaceNameFromNetAddress(targetNet string) string {
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				if v.IP.To4() != nil {
					ifaceNet := getNetAddress(v)
					if ifaceNet == targetNet {
						return i.Name
					}
				}
			}
		}
	}
	return ""
}

func getPodInfo(cniArgs string) (string, string) {
	splits := strings.Split(cniArgs, ";")
	var podName, podNamespace string
	for _, split := range splits {
		if strings.HasPrefix(split, "K8S_POD_NAME=") {
			podName = strings.TrimPrefix(split, "K8S_POD_NAME=")
		}
		if strings.HasPrefix(split, "K8S_POD_NAMESPACE=") {
			podNamespace = strings.TrimPrefix(split, "K8S_POD_NAMESPACE=")
		}
	}
	return podName, podNamespace
}

func LoadIPAMConfig(bytes []byte) (*IPAMConfig, string, error) {
	n := Net{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return nil, "", err
	}
	if n.IPAM == nil {
		return nil, "", fmt.Errorf("IPAM config missing 'ipam' key")
	}
	n.IPAM.Name = n.Name
	return n.IPAM, n.CNIVersion, nil

}


func loadNetConf(bytes []byte) (*Net, string, error) {
	n := &Net{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}


func cmdCheck(args *skel.CmdArgs) error {
	// Get PrevResult from stdin... store in RawPrevResult
	n, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	// Parse previous result.
	if n.RawPrevResult == nil {
		return fmt.Errorf("required prevResult missing")
	}

	if err := version.ParsePrevResult(&n.NetConf); err != nil {
		return err
	}

	result, err := current.NewResultFromResult(n.PrevResult)
	if err != nil {
		return err
	}

	if len(result.IPs) == 0 {
		return fmt.Errorf("no ip allocated")
	}

	for _, ips := range result.IPs {
		_, subnetNet, err := net.ParseCIDR(n.Subnet)
		if err != nil {
			return fmt.Errorf("cannot parse subnet %s", n.Subnet)
		}
		if !subnetNet.Contains(ips.Address.IP) {
			return fmt.Errorf("allocated ip %s is not in designated subnet %s", ips.Address.IP, n.Subnet)
		} 
	}

	return nil
}


func cmdAdd(args *skel.CmdArgs) error {
	n, confVersion, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	var result *current.Result
	haveResult := false
	// Parse previous result
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

	if !haveResult {
		ipamConf, _, err := LoadIPAMConfig(args.StdinData)

		// do nothing
		if len(n.Masters) == 0 {
			return types.PrintResult(result, confVersion)
		}
		
		// multi-nic-cni IPAM
		hostName, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get host name")
		}
		podName, podNamespace := getPodInfo(args.Args)

		ipResponses, err := RequestIP(ipamConf.DaemonIP, ipamConf.DaemonPort, podName, podNamespace, hostName, ipamConf.Name, n.Masters)

		if err != nil {
			return fmt.Errorf("failed to request ip %v", err)
		}

		for index, master := range n.Masters {
			// find match master information and add
			for _, ipResponse := range ipResponses {
				if ipResponse.InterfaceName == master{
					vlanPodCIDR := fmt.Sprintf("%s/%s", ipResponse.IPAddress, ipResponse.VLANBlockSize)
					ipVal, reservedIP, err := net.ParseCIDR(vlanPodCIDR)
					reservedIP.IP = ipVal
					if err != nil {
						return fmt.Errorf("failed to parse IP: %s: %v", ipResponse.IPAddress, err)
					}
					ipConf := &current.IPConfig{
						Address:   *reservedIP,
						Interface: current.Int(index),
					}
					result.IPs = append(result.IPs, ipConf)
					break
				}
			}
		}
		result.DNS = ipamConf.DNS
		result.Routes = ipamConf.Routes
	}

	return types.PrintResult(result, confVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	
	if args.Netns == "" {
		return nil
	}

	ipamConf, _, err := LoadIPAMConfig(args.StdinData)
	if err != nil {
		return fmt.Errorf("fail to load ipam conf")
	}
	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get host name")
	}
	podName, podNamespace := getPodInfo(args.Args)

	Deallocate(ipamConf.DaemonPort, podName, podNamespace, hostName, ipamConf.Name)

	return nil
}
