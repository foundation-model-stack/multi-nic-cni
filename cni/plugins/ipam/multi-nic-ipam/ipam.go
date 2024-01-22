/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bytes"
	"errors"
)

const (
	ALLOCATE_PATH   = "allocate"
	DEALLOCATE_PATH = "deallocate"

	DEFAULT_DAEMON_PORT = 11000

	DEFAULT_DAEMON_IP = "localhost"
)

type IPRequest struct {
	PodName          string   `json:"pod"`
	PodNamespace     string   `json:"namespace"`
	HostName         string   `json:"host"`
	NetAttachDefName string   `json:"def"`
	InterfaceNames   []string `json:"masters"`
}

type IPResponse struct {
	InterfaceName string `json:"interface"`
	IPAddress     string `json:"ip"`
	VLANBlockSize string `json:"block"`
}

func RequestIP(daemonIP string, daemonPort int, podName string, podNamespace string, hostName string, defName string, masters []string) ([]IPResponse, error) {
	var response []IPResponse
	if daemonPort == 0 {
		daemonPort = DEFAULT_DAEMON_PORT
	}
	if daemonIP == "" {
		daemonIP = DEFAULT_DAEMON_IP
	}
	address := fmt.Sprintf("http://%s:%d/%s", daemonIP, daemonPort, ALLOCATE_PATH)
	request := IPRequest{
		PodName:          podName,
		PodNamespace:     podNamespace,
		HostName:         hostName,
		NetAttachDefName: defName,
		InterfaceNames:   masters,
	}

	jsonReq, err := json.Marshal(request)

	if err != nil {
		return response, fmt.Errorf("marshal fail: %v", err)
	} else {
		client := http.Client{
			Timeout: 2 * time.Minute,
		}
		defer client.CloseIdleConnections()
		res, err := client.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			return response, fmt.Errorf("post fail: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return response, errors.New(res.Status)
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return response, fmt.Errorf("read body: %v", err)
		}
		err = json.Unmarshal(body, &response)
		if err == nil && len(response) == 0 {
			return response, fmt.Errorf("response nothing")
		}
		return response, err
	}
}

func Deallocate(daemonPort int, podName string, podNamespace string, hostName string, defName string) ([]IPResponse, error) {
	var response []IPResponse
	if daemonPort == 0 {
		daemonPort = DEFAULT_DAEMON_PORT
	}

	address := fmt.Sprintf("http://localhost:%d/%s", daemonPort, DEALLOCATE_PATH)
	request := IPRequest{
		PodName:          podName,
		PodNamespace:     podNamespace,
		HostName:         hostName,
		NetAttachDefName: defName,
	}

	jsonReq, err := json.Marshal(request)

	if err != nil {
		return response, fmt.Errorf("marshal fail: %v", err)
	} else {
		client := http.Client{
			Timeout: 2 * time.Minute,
		}
		defer client.CloseIdleConnections()
		res, err := client.Post(address, "application/json; charset=utf-8", bytes.NewBuffer(jsonReq))
		if err != nil {
			return response, fmt.Errorf("post fail: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return response, errors.New(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return response, fmt.Errorf("read body: %v", err)
		}
		err = json.Unmarshal(body, &response)
		if err == nil && len(response) == 0 {
			return response, fmt.Errorf("response nothing")
		}
		return response, err
	}
}
