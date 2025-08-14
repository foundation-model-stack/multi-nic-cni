/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HostInterfaceHandler handles HostInterface object
// - general handling: Get, List, Delete
type HostInterfaceHandler struct {
	*kubernetes.Clientset
	client.Client
	*SafeCache
	DaemonConnector
}

// NewHostInterfaceHandler
func NewHostInterfaceHandler(config *rest.Config, client client.Client) *HostInterfaceHandler {
	clientset, _ := kubernetes.NewForConfig(config)

	return &HostInterfaceHandler{
		Clientset: clientset,
		Client:    client,
		SafeCache: InitSafeCache(),
		DaemonConnector: DaemonConnector{
			Clientset: clientset,
		},
	}
}

// initHostInterface
func (h *HostInterfaceHandler) initHostInterface(hostName string, interfaces []multinicv1.InterfaceInfoType) *multinicv1.HostInterface {
	newHif := &multinicv1.HostInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostName,
		},
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   hostName,
			Interfaces: interfaces,
		},
	}
	return newHif
}

// CreateHostInterface creates new HostInterface from an interface list get from daemon pods
func (h *HostInterfaceHandler) CreateHostInterface(hostName string, interfaces []multinicv1.InterfaceInfoType) error {
	newHif := h.initHostInterface(hostName, interfaces)
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	return h.Client.Create(ctx, newHif)
}

// UpdateHostInterface updates HostInterface
func (h *HostInterfaceHandler) UpdateHostInterface(oldObj multinicv1.HostInterface, interfaces []multinicv1.InterfaceInfoType) (*multinicv1.HostInterface, error) {
	updateHif := &multinicv1.HostInterface{
		ObjectMeta: oldObj.ObjectMeta,
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   oldObj.Spec.HostName,
			Interfaces: interfaces,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	return updateHif, h.Client.Update(ctx, updateHif)
}

// GetHostInterface gets HostInterface from hostname
func (h *HostInterfaceHandler) GetHostInterface(name string) (*multinicv1.HostInterface, error) {
	instance := &multinicv1.HostInterface{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.Get(ctx, namespacedName, instance)
	return instance, err
}

// ListHostInterface returns a map from hostname to HostInterface
func (h *HostInterfaceHandler) ListHostInterface() (map[string]multinicv1.HostInterface, error) {
	hifList := &multinicv1.HostInterfaceList{}
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	err := h.Client.List(ctx, hifList)
	hifMap := make(map[string]multinicv1.HostInterface)
	if err == nil {
		for _, hif := range hifList.Items {
			name := hif.GetName()
			hifMap[name] = hif
		}
	}
	return hifMap, err
}

// DeleteHostInterface deletes HostInterface from hostname
func (h *HostInterfaceHandler) DeleteHostInterface(name string) error {
	instance, err := h.GetHostInterface(name)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
		defer cancel()
		err = h.Client.Delete(ctx, instance)
	}
	return err
}

func (h *HostInterfaceHandler) SetCache(key string, value multinicv1.HostInterface) {
	h.SafeCache.SetCache(key, value)
}

func (h *HostInterfaceHandler) GetCache(key string) (multinicv1.HostInterface, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return multinicv1.HostInterface{}, errors.New(vars.NotFoundError)
	}
	return value.(multinicv1.HostInterface), nil
}

func (h *HostInterfaceHandler) ListCache() map[string]multinicv1.HostInterface {
	snapshot := make(map[string]multinicv1.HostInterface)
	h.SafeCache.Lock()
	for key, value := range h.cache {
		snapshot[key] = value.(multinicv1.HostInterface)
	}
	h.SafeCache.Unlock()
	return snapshot
}

//////////////////////////////////////////////////////////////
// Connecting with daemon pod

// ipamJoin calls daemon to greet the existing hosts by referring to HostInterface list
func (h *HostInterfaceHandler) IpamJoin(daemon DaemonPod) error {
	podIP := daemon.HostIP
	if podIP == "" {
		// ip hasn't been assigned yet
		return nil
	}
	var hifs []multinicv1.InterfaceInfoType
	snapshot := h.ListCache()
	for _, hif := range snapshot {
		if hif.Spec.HostName == daemon.HostIP {
			continue
		}
		hifs = append(hifs, hif.Spec.Interfaces...)
	}
	hifLen := len(hifs)
	if lastLenStr, exists := daemon.Labels[vars.JoinLabelName]; exists {
		// already join
		lastLen, _ := strconv.ParseInt(lastLenStr, 10, 64)
		if hifLen == int(lastLen) {
			// join with the same number of hostinterfaces
			return nil
		}
	}
	podAddress := GetDaemonAddressByPod(daemon)
	vars.HifLog.V(4).Info(fmt.Sprintf("Join %s with %d hifs", daemon.HostIP, hifLen))
	err := h.DaemonConnector.Join(podAddress, hifs)
	if err != nil {
		return err
	}
	err = h.addLabel(daemon, vars.JoinLabelName, fmt.Sprintf("%d", hifLen))
	if err != nil {
		vars.HifLog.V(4).Info(fmt.Sprintf("Fail to add label to %s: %v", daemon.Name, err))
	}
	return nil
}

// addLabel labels daemon pod with node name to update HostInterface
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (h *HostInterfaceHandler) addLabel(daemon DaemonPod, labelName string, labelValue string) error {
	podName := daemon.Name
	namespace := daemon.Namespace
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  fmt.Sprintf("/metadata/labels/%s", labelName),
		Value: labelValue,
	}}
	payloadBytes, _ := json.Marshal(payload)

	_, err := h.Clientset.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	return err
}

func (h *HostInterfaceHandler) GetInfoAvailableSize() int {
	hostInterfaceSnapshot := h.ListCache()
	infoAvailableSize := 0
	for _, instance := range hostInterfaceSnapshot {
		if len(instance.Spec.Interfaces) > 0 {
			infoAvailableSize += 1
		}
	}
	return infoAvailableSize
}
