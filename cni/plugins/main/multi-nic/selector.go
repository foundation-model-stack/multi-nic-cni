/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */
 
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"bytes"
	"errors"
)

const (
	NIC_SELECT_PATH     = "select"
	DEFAULT_DAEMON_PORT = 11000
	DEFAULT_DAEMON_IP = "localhost"
)

type NICSelectRequest struct {
	PodName          string   `json:"pod"`
	PodNamespace     string   `json:"namespace"`
	HostName         string   `json:"host"`
	NetAttachDefName string   `json:"def"`
	MasterNetAddrs []string   `json:"masterNets"`
	NicSet           NicArgs  `json:"args"`
}

type NICSelectResponse struct {
	DeviceIDs      []string   `json:"deviceIDs"`
	Masters 	   []string   `json:"masters"`
}

func selectNICs(daemonIP string, daemonPort int, podName string, podNamespace string, hostName string, defName string, nicSet NicArgs, masterNets []string) (NICSelectResponse, error) {
	var response NICSelectResponse
	if daemonPort == 0 {
		daemonPort = DEFAULT_DAEMON_PORT
	}
	if daemonIP == "" {
		daemonIP = DEFAULT_DAEMON_IP
	}
	address := fmt.Sprintf("http://%s:%d/%s", daemonIP, daemonPort, NIC_SELECT_PATH)
	request := NICSelectRequest{
		PodName:          podName,
		PodNamespace:     podNamespace,
		HostName:         hostName,
		NetAttachDefName: defName,
		MasterNetAddrs:   masterNets,
		NicSet:           nicSet,
	}

	jsonReq, err := json.Marshal(request)

	if err != nil {
		return response, errors.New(fmt.Sprintf("Marshal fail: %v", err))
	} else {
		res, err := http.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			return response, errors.New(fmt.Sprintf("Post fail: %v", err))
		}
		if res.StatusCode != http.StatusOK {
			return response, errors.New(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return response, errors.New(fmt.Sprintf("Read body: %v", err))
		}
		err = json.Unmarshal(body, &response)
		if err == nil && len(response.Masters) == 0 {
			return response, fmt.Errorf("Response nothing")
		}
		return response, err
	}
}

