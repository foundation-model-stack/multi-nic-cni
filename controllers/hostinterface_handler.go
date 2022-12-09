/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/go-logr/logr"
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
	Log logr.Logger
	*SafeCache
	DaemonConnector
}

// NewHostInterfaceHandler
func NewHostInterfaceHandler(config *rest.Config, client client.Client, logger logr.Logger) *HostInterfaceHandler {
	clientset, _ := kubernetes.NewForConfig(config)

	return &HostInterfaceHandler{
		Clientset: clientset,
		Client:    client,
		Log:       logger,
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
	return h.Client.Create(context.TODO(), newHif)
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
	return updateHif, h.Client.Update(context.TODO(), updateHif)
}

// GetHostInterface gets HostInterface from hostname
func (h *HostInterfaceHandler) GetHostInterface(name string) (*multinicv1.HostInterface, error) {
	instance := &multinicv1.HostInterface{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: metav1.NamespaceAll,
	}
	err := h.Client.Get(context.TODO(), namespacedName, instance)
	return instance, err
}

// ListHostInterface returns a map from hostname to HostInterface
func (h *HostInterfaceHandler) ListHostInterface() (map[string]multinicv1.HostInterface, error) {
	hifList := &multinicv1.HostInterfaceList{}
	err := h.Client.List(context.TODO(), hifList)
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
		err = h.Client.Delete(context.TODO(), instance)
	}
	h.SafeCache.UnsetCache(name)
	return err
}

func (h *HostInterfaceHandler) SetCache(key string, value multinicv1.HostInterface) {
	h.SafeCache.SetCache(key, value)
}

func (h *HostInterfaceHandler) GetCache(key string) (multinicv1.HostInterface, error) {
	value := h.SafeCache.GetCache(key)
	if value == nil {
		return multinicv1.HostInterface{}, fmt.Errorf("Not Found")
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
	if lastLenStr, exists := daemon.Labels[JOIN_LABEL_NAME]; exists {
		// already join
		lastLen, _ := strconv.ParseInt(lastLenStr, 10, 64)
		if hifLen == int(lastLen) {
			// join with the same number of hostinterfaces
			return nil
		}
	}
	podAddress := GetDaemonAddressByPod(daemon)
	h.Log.Info(fmt.Sprintf("Join %s with %d hifs", daemon.HostIP, hifLen))
	err := h.DaemonConnector.Join(podAddress, hifs)
	if err != nil {
		return err
	}
	err = h.addLabel(daemon, JOIN_LABEL_NAME, fmt.Sprintf("%d", hifLen))
	if err != nil {
		h.Log.Info(fmt.Sprintf("Fail to add label to %s: %v", daemon.Name, err))
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
