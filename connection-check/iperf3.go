/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DEFAULT_LABEL_NAME         = "multi-nic-concheck"
	DEFAULT_SERVER_LABEL_VALUE = "server"
	DEFAULT_CLIENT_LABEL_VALUE = "client"
	NETWORK_ANNOTATION         = "k8s.v1.cni.cncf.io/networks"
	NETWORK_STATUS_ANNOTATION  = "k8s.v1.cni.cncf.io/network-status"

	IPERF_IMAGE = "networkstatic/iperf3"
)

const (
	BANDWIDTH_KEY = "bits/sec"
	ERROR_KEY     = "Bad file descriptor"
)

type NetworkStatus struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
}

type IperfHandler struct {
	Clientset *kubernetes.Clientset
}

func NewIperfHandler(config *rest.Config) *IperfHandler {
	clientset, _ := kubernetes.NewForConfig(config)
	handler := &IperfHandler{
		Clientset: clientset,
	}
	return handler
}
func (h *IperfHandler) getName(cidrName string, hostName string, labelValue string) string {
	return fmt.Sprintf("%s-%s-%s", cidrName, hostName, labelValue)
}

func (h *IperfHandler) getLabelValue(cidrName string, labelValue string) string {
	return fmt.Sprintf("%s-%s", cidrName, labelValue)
}

func (h *IperfHandler) getMetaObject(namespace string, cidrName string, hostName string, labelValue string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      h.getName(cidrName, hostName, labelValue),
		Namespace: namespace,
		Labels: map[string]string{
			DEFAULT_LABEL_NAME: h.getLabelValue(cidrName, labelValue),
		},
		Annotations: map[string]string{
			NETWORK_ANNOTATION: cidrName,
		},
	}
}

func (h *IperfHandler) CreateServerPod(namespace string, cidrName string, hostName string) (*v1.Pod, error) {
	var period int64
	period = 0
	container := v1.Container{
		Name:            DEFAULT_SERVER_LABEL_VALUE,
		Image:           IPERF_IMAGE,
		ImagePullPolicy: v1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{"echo started; iperf3 -s"},
	}
	pod := &v1.Pod{
		ObjectMeta: h.getMetaObject(namespace, cidrName, hostName, DEFAULT_SERVER_LABEL_VALUE),
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				container,
			},
			NodeName:                      hostName,
			TerminationGracePeriodSeconds: &period,
		},
	}
	return h.Clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

func (h *IperfHandler) CheckServers(namespace string, cidrName string, totalCount int) (map[string][]string, bool) {
	serverIPsMap := make(map[string][]string)
	labels := fmt.Sprintf("%s=%s", DEFAULT_LABEL_NAME, h.getLabelValue(cidrName, DEFAULT_SERVER_LABEL_VALUE))
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	podList, err := h.Clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil || len(podList.Items) < totalCount {
		log.Printf("some server is not available %d/%d or error %v", len(podList.Items), totalCount, err)
		return serverIPsMap, false
	}
	for _, pod := range podList.Items {
		// pod is running
		if pod.Status.Phase != v1.PodRunning {
			log.Printf("%s: %s", pod.GetName(), pod.Status.Phase)
			return serverIPsMap, false
		}
		lines, err := h.getLog(pod)
		if err != nil || len(lines) == 0 {
			log.Printf("%s wait for server to run (%v): %v", pod.GetName(), lines, err)
			return serverIPsMap, false
		}
		// annotation exists
		annotations := pod.ObjectMeta.Annotations
		if netstatus, exist := annotations[NETWORK_STATUS_ANNOTATION]; exist {
			var statusObj []NetworkStatus
			err := json.Unmarshal([]byte(netstatus), &statusObj)
			if err != nil {
				log.Printf("cannot unmarshal %s: %v", netstatus, err)
				return serverIPsMap, false
			}
			for _, status := range statusObj {
				if status.Name == fmt.Sprintf("%s/%s", namespace, cidrName) {
					serverIPsMap[pod.Spec.NodeName] = status.IPs
				}
			}
		} else {
			log.Printf("%s is not annotated", pod.GetName())
			return serverIPsMap, false
		}
	}
	return serverIPsMap, true
}

func (h *IperfHandler) generateCommand(hostName string, ipMap map[string][]string) string {
	allIPs := []string{}
	for targetHost, ips := range ipMap {
		if targetHost == hostName {
			continue
		}
		allIPs = append(allIPs, ips...)
	}
	cmd := ""
	for _, clientIP := range allIPs {
		cmd = fmt.Sprintf("%s echo -n %s,;iperf3 -c %s -n 1 --connect-timeout 10s|grep %s -m 1|awk '{ print $7$8 }';sleep 1;echo '';", cmd, clientIP, clientIP, BANDWIDTH_KEY)
	}
	// log.Printf("%s", cmd)
	return cmd
}

func (h *IperfHandler) CreateClientJob(namespace string, cidrName string, hostName string, ipMap map[string][]string) (*batchv1.Job, error) {
	container := v1.Container{
		Name:            DEFAULT_SERVER_LABEL_VALUE,
		Image:           IPERF_IMAGE,
		ImagePullPolicy: v1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{h.generateCommand(hostName, ipMap)},
	}
	var period int64
	period = 0
	job := &batchv1.Job{
		ObjectMeta: h.getMetaObject(namespace, cidrName, hostName, DEFAULT_CLIENT_LABEL_VALUE),
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: h.getMetaObject(namespace, cidrName, hostName, DEFAULT_CLIENT_LABEL_VALUE),
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						container,
					},
					NodeName:                      hostName,
					TerminationGracePeriodSeconds: &period,
					RestartPolicy:                 v1.RestartPolicyNever,
				},
			},
		},
	}
	return h.Clientset.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
}
func (h *IperfHandler) CheckClient(job *batchv1.Job) bool {
	jobName := job.GetName()
	getJob, err := h.Clientset.BatchV1().Jobs(job.GetNamespace()).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Cannot get job %s", jobName)
		return false
	}
	if getJob.Status.Active > 0 {
		log.Printf("Some job is still running: %s", jobName)
		return false
	}
	if getJob.Status.Succeeded <= 0 {
		log.Printf("Job failed: %s, %d", jobName, getJob.Status.Succeeded)
		return false
	}
	return true
}

func (h *IperfHandler) CheckClients(namespace string, cidrName string, totalCount int) ([]batchv1.Job, bool) {
	jobs := []batchv1.Job{}
	labels := fmt.Sprintf("%s=%s", DEFAULT_LABEL_NAME, h.getLabelValue(cidrName, DEFAULT_CLIENT_LABEL_VALUE))
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	jobList, err := h.Clientset.BatchV1().Jobs(namespace).List(context.TODO(), listOptions)
	if err != nil || len(jobList.Items) < totalCount {
		log.Printf("some client is not created %d/%d or error %v", len(jobList.Items), totalCount, err)
		return jobs, false
	}
	for _, job := range jobList.Items {
		// job is running
		if job.Status.Active > 0 {
			log.Printf("Some job is still running: %s", job.GetName())
			return jobs, false
		} else {
			if job.Status.Succeeded > 0 {
				jobs = append(jobs, job)
			} else {
				log.Printf("%s failed", job.GetName())
				return jobs, false
			}
		}
	}
	return jobs, true
}

func (h *IperfHandler) getLog(pod v1.Pod) ([]string, error) {
	podlog := h.Clientset.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), &v1.PodLogOptions{})
	podLogs, err := podlog.Stream(context.TODO())
	if err != nil {
		log.Printf("cannot read stream")
		return []string{}, err
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		log.Printf("cannot copy buffer")
		return []string{}, err
	}
	lines := strings.Split(buf.String(), "\n")
	return lines, nil
}

func (h *IperfHandler) ReadResult(namespace string, cidrName string, ipMap map[string][]string) map[string]map[string]string {
	fmt.Println("###########################################")
	fmt.Printf("## Connection Check: %s\n", cidrName)
	fmt.Println("###########################################")

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	fmt.Fprintln(w, "FROM\tTO\t\tCONNECTED/TOTAL\tIPs\tBANDWIDTHs")
	result := make(map[string]map[string]string)
	labels := fmt.Sprintf("%s=%s", DEFAULT_LABEL_NAME, h.getLabelValue(cidrName, DEFAULT_CLIENT_LABEL_VALUE))
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	podList, _ := h.Clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	for _, pod := range podList.Items {
		hostName := pod.Spec.NodeName
		result[hostName] = make(map[string]string)
		lines, err := h.getLog(pod)
		if err != nil {
			log.Printf("cannot read log")
		} else {
			for _, line := range lines {
				values := strings.Split(line, ",")
				if len(values) == 0 {
					continue
				}
				if len(values) > 1 {
					result[hostName][values[0]] = values[1]
				} else {
					result[hostName][values[0]] = ERROR_KEY
				}

			}
		}

		for targetHost, ips := range ipMap {
			if targetHost == hostName {
				continue
			}
			failCount := 0
			bps := "["
			for _, ip := range ips {
				if val, exist := result[hostName][ip]; exist {
					if strings.Contains(val, ERROR_KEY) || val == "" {
						failCount += 1
					} else {
						bps = fmt.Sprintf("%s %s", bps, val)
					}
				}
			}
			bps += "]"
			total := len(ips)
			fmt.Fprintf(w, "%s\t%s\t\t%d/%d\t%v\t%s\n", hostName, targetHost, total-failCount, total, ips, bps)
		}
	}
	w.Flush()
	fmt.Println("###########################################")
	return result
}
