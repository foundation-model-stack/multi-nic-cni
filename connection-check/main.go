/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"log"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DEFAULT_NAMESPACE = "default"
	STREAMS_PER_IP    = 5
)

func getConfig() *rest.Config {
	var config *rest.Config
	var err error
	presentKube, ok := os.LookupEnv("KUBECONFIG_FILE")
	if !ok && presentKube != "" {
		log.Println("InCluster Config")
		config, err = rest.InClusterConfig()
	} else {
		log.Printf("Config %s", presentKube)
		config, err = clientcmd.BuildConfigFromFlags("", presentKube)
	}
	if err != nil {
		log.Printf("Config Error: %v", err)
	}
	return config
}

func main() {
	config := getConfig()
	cidrHandler := NewCIDRHandler(config)
	iperfHanlder := NewIperfHandler(config)
	// list all cidr resource
	cidrMap := cidrHandler.List(metav1.ListOptions{})
	for cidrName, cidr := range cidrMap {
		podCIDRsMap := cidrHandler.GetPodCIDRsMap(cidr)
		totalCount := 0

		// create iperf server pod for listed hosts with cidr multi-nic-network
		for host, cidrMap := range podCIDRsMap {
			numberOfInterface := len(cidrMap)
			_, err := iperfHanlder.CreateServerPod(DEFAULT_NAMESPACE, cidrName, host, numberOfInterface)
			if err != nil {
				log.Printf("Cannot create server pod for %s, %s: %v", cidrName, host, err)
			} else {
				totalCount = totalCount + 1
			}
		}
		log.Printf("%d/%d servers successfully created", totalCount, len(podCIDRsMap))
		// wait for server ready
		var ipMaps map[string][]string
		var primaryIPMap map[string]string
		var serversReady bool
		for {
			primaryIPMap, ipMaps, serversReady = iperfHanlder.CheckServers(DEFAULT_NAMESPACE, cidrName, totalCount)
			time.Sleep(5 * time.Second)
			if serversReady {
				break
			}
		}

		// create iperf client job for listed related hosts with cidr multi-nic-network
		totalCount = 0
		for host, _ := range podCIDRsMap {
			job, err := iperfHanlder.CreateClientJob(DEFAULT_NAMESPACE, cidrName, host, primaryIPMap, ipMaps)

			if err != nil {
				log.Printf("Cannot create client job for %s, %s: %v", cidrName, host, err)
			} else {
				totalCount = totalCount + 1
			}
			for {
				time.Sleep(5 * time.Second)
				clientReady := iperfHanlder.CheckClient(job)
				if clientReady {
					break
				}
			}
		}
		log.Printf("%d/%d clients successfully finished", totalCount, len(podCIDRsMap))
		iperfHanlder.ReadResult(DEFAULT_NAMESPACE, cidrName, ipMaps)
		log.Printf("%s checked", cidrName)
	}
}
