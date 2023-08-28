/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/foundation-model-stack/multi-nic-cni/e2e-test/cni-stub/exec"
)

const (
	podNamePrefix = "pod"
	podNamespace  = "default"
	maxTry        = 5
	waitTime      = 1
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
		NicSet *exec.NicArgs `json:"cni,omitempty"`
	} `json:"args"`
}

func getConfig(daemonIP string, daemonPort int) string {
	masterNetsBytes, _ := json.Marshal(POOL_NETWORK_ADDRESSES)
	masterNets := string(masterNetsBytes)
	ipamValue := fmt.Sprintf(`"ipam": {
		"type": "multi-nic-ipam",
		"hostBlock": 8,
		"interfaceBlock": 2,
		"daemonIP": "%s",
		"daemonPort": %d
		},
		"multiNICIPAM": true,`, daemonIP, daemonPort)
	return fmt.Sprintf(`{ 
		"cniVersion": "0.3.0", 
		"name": "multi-nic-sample",
		"type": "multi-nic",
		"plugin": {
			"cniVersion": "0.3.0",
			"type": "ipvlan",
			"mode": "l3"
		},
		"vlanMode": "l3",
		%s
		"daemonIP": "%s",
		"daemonPort": %d,
		"subnet": "192.168.0.0/16",
		"masterNets": %s
		}`, ipamValue, daemonIP, daemonPort, masterNets)
}

var (
	DEFAULT_DAEMON_PORT             = 11000
	POOL_NETWORK_ADDRESSES []string = []string{"10.0.0.0/16", "10.1.0.0/16"}
)

func on_pod_create(startI, numOfPod int, hostName, daemonIP string, daemonPort int) error {
	conf := getConfig(daemonIP, daemonPort)
	n := &NetConf{}
	if err := json.Unmarshal([]byte(conf), n); err != nil {
		return err
	}
	for i := startI; i < startI+numOfPod; i++ {
		podName := fmt.Sprintf("%s-%s-%d", podNamePrefix, hostName, i)
		// select nic
		nicSet := exec.NicArgs{}
		tried := 1
		var selectResponse exec.NICSelectResponse
		var err error
		for {
			selectResponse, err = exec.SelectNICs(n.DaemonIP, n.DaemonPort, podName, podNamespace, hostName, n.Name, nicSet, n.MasterNetAddrs)
			if len(selectResponse.Masters) == 0 || err != nil {
				if tried > maxTry {
					return fmt.Errorf("Failed to select NIC: %d, %v", len(selectResponse.Masters), err)
				}
				tried += 1
			} else {
				break
			}
		}
		log.Printf("%s successfully selected NIC (%d tried)", podName, tried)
		n.Masters = selectResponse.Masters
		n.DeviceIDs = selectResponse.DeviceIDs

		tried = 1
		for {
			ipResponses, err := exec.RequestIP(n.DaemonIP, n.DaemonPort, podName, podNamespace, hostName, n.Name, n.Masters)
			if len(ipResponses) == 0 || err != nil {
				if tried > maxTry {
					return fmt.Errorf("Failed to allocate IP: %d, %v", len(ipResponses), err)
				}
				tried += 1
			} else {
				log.Printf("%s successfully allocate IP (%d tried)", podName, tried)
				break
			}
		}
	}
	return nil
}

func on_pod_delete(startI, numOfPod int, hostName, daemonIP string, daemonPort int) error {
	conf := getConfig(daemonIP, daemonPort)
	n := &NetConf{}
	if err := json.Unmarshal([]byte(conf), n); err != nil {
		return err
	}
	for i := startI; i < startI+numOfPod; i++ {
		podName := fmt.Sprintf("%s-%s-%d", podNamePrefix, hostName, i)
		exec.Deallocate(n.DaemonIP, n.DaemonPort, podName, podNamespace, hostName, n.Name)
		log.Printf("%s deallocated", podName)
	}
	return nil
}

var (
	Cmd        = flag.String("command", "add", "cni exec command (add, delete)")
	StartI     = flag.Int("start", 0, "number of pods")
	NumOfPod   = flag.Int("n", 1, "number of pods")
	HostName   = flag.String("host", "", "host name")
	DaemonIP   = flag.String("dip", "", "daemon IP")
	DaemonPort = flag.Int("dport", DEFAULT_DAEMON_PORT, "daemon port")
)

func main() {
	flag.Parse()
	var err error
	switch *Cmd {
	case "add":
		err = on_pod_create(*StartI, *NumOfPod, *HostName, *DaemonIP, *DaemonPort)
	case "delete":
		err = on_pod_delete(*StartI, *NumOfPod, *HostName, *DaemonIP, *DaemonPort)
	}
	if err != nil {
		log.Printf("CNIFailed (%s): %v", *Cmd, err)
	}
}
