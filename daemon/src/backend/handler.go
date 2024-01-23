/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	API_VERSION       = "multinic.fms.io/v1"
	APISERVER_TIMEOUT = 2 * time.Minute
)

type DynamicHandler struct {
	DC           *discovery.DiscoveryClient
	DYN          dynamic.Interface
	ResourceName string
	Kind         string
}

func (h *DynamicHandler) BasicObject(name string) map[string]interface{} {
	obj := make(map[string]interface{})
	obj["apiVersion"] = API_VERSION
	obj["kind"] = h.Kind
	obj["metadata"] = map[string]interface{}{
		"name": name,
	}
	return obj
}

func (h *DynamicHandler) Untidy(structObj interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	jsonStr, err := json.Marshal(structObj)
	if err != nil {
		return output
	}
	err = json.Unmarshal(jsonStr, &output)
	return output
}

func (h *DynamicHandler) Parse(obj map[string]interface{}, output interface{}) interface{} {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return output
	}
	err = json.Unmarshal(jsonBytes, output)
	return output
}

func (h *DynamicHandler) GetName(uobj unstructured.Unstructured) string {
	return uobj.Object["metadata"].(map[string]interface{})["name"].(string)
}

func (h *DynamicHandler) Create(mapObj map[string]interface{}, namespace string, options metav1.CreateOptions) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{Object: mapObj}
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	log.Println(fmt.Sprintf("Create %s/%s", h.ResourceName, mapObj["metadata"].(map[string]interface{})["name"]))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	res, err := h.DYN.Resource(*gvr).Namespace(namespace).Create(ctx, obj, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("Create%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return res, err
}

func (h *DynamicHandler) Update(mapObj map[string]interface{}, namespace string, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{Object: mapObj}
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	log.Println(fmt.Sprintf("Update %s/%s", h.ResourceName, mapObj["metadata"].(map[string]interface{})["name"]))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	res, err := h.DYN.Resource(*gvr).Namespace(namespace).Update(ctx, obj, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("Update%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return res, err
}

func (h *DynamicHandler) List(namespace string, options metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	res, err := h.DYN.Resource(*gvr).Namespace(namespace).List(ctx, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("List%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return res, err
}

func (h *DynamicHandler) Get(name string, namespace string, options metav1.GetOptions) (*unstructured.Unstructured, error) {
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	res, err := h.DYN.Resource(*gvr).Namespace(namespace).Get(ctx, name, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("Get%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return res, err
}

func (h *DynamicHandler) Delete(name string, namespace string, options metav1.DeleteOptions) error {
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	log.Println(fmt.Sprintf("Delete %s/%s", h.ResourceName, name))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	err := h.DYN.Resource(*gvr).Namespace(namespace).Delete(ctx, name, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("Delete%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return err
}

func (h *DynamicHandler) Patch(name string, namespace string, pt types.PatchType, data []byte, options metav1.PatchOptions) (*unstructured.Unstructured, error) {
	gvr, _ := schema.ParseResourceArg(h.ResourceName)
	log.Println(fmt.Sprintf("Patch %s/%s - %s", h.ResourceName, name, string(data)))
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), APISERVER_TIMEOUT)
	defer cancel()
	res, err := h.DYN.Resource(*gvr).Namespace(namespace).Patch(ctx, name, pt, data, options)
	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("Patch%s elapsed: %d us", h.Kind, int64(elapsed/time.Microsecond)))
	return res, err
}
