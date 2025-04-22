/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type DynamicHandler struct {
	DYN dynamic.Interface
	GVR schema.GroupVersionResource
}

func (h *DynamicHandler) Create(namespace string, dynamicObj interface{}, result interface{}) error {
	options := metav1.CreateOptions{}
	ctx := context.TODO()
	obj := &unstructured.Unstructured{}
	inbytes, err := json.Marshal(dynamicObj)
	if err != nil {
		return err
	}
	err = obj.UnmarshalJSON(inbytes)
	if err != nil {
		return err
	}
	ures, err := h.DYN.Resource(h.GVR).Namespace(namespace).Create(ctx, obj, options)
	if err != nil {
		return err
	}
	outbytes, err := ures.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(outbytes, result)
	return err
}

func (h *DynamicHandler) Update(namespace string, dynamicObj interface{}, result interface{}) error {
	options := metav1.UpdateOptions{}
	ctx := context.TODO()
	obj := &unstructured.Unstructured{}
	inbytes, err := json.Marshal(dynamicObj)
	if err != nil {
		return err
	}
	err = obj.UnmarshalJSON(inbytes)
	if err != nil {
		return err
	}
	ures, err := h.DYN.Resource(h.GVR).Namespace(namespace).Update(ctx, obj, options)
	if err != nil {
		return err
	}
	outbytes, err := ures.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(outbytes, result)
	return err
}

func (h *DynamicHandler) Delete(name string, namespace string) error {
	options := metav1.DeleteOptions{}
	ctx := context.TODO()
	return h.DYN.Resource(h.GVR).Namespace(namespace).Delete(ctx, name, options)
}

func (h *DynamicHandler) Get(name string, namespace string, result interface{}) error {
	options := metav1.GetOptions{}
	ctx := context.TODO()
	ures, err := h.DYN.Resource(h.GVR).Namespace(namespace).Get(ctx, name, options)
	if err != nil {
		return err
	}
	outbytes, err := ures.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(outbytes, result)
	return err
}

func (h *DynamicHandler) GetFirst(namespace string, result interface{}) error {
	options := metav1.ListOptions{}
	ctx := context.TODO()
	itemList, err := h.DYN.Resource(h.GVR).Namespace(namespace).List(ctx, options)
	if err != nil {
		return err
	}
	if len(itemList.Items) > 0 {
		ures := itemList.Items[0]
		outbytes, err := ures.MarshalJSON()
		if err != nil {
			return err
		}
		return json.Unmarshal(outbytes, result)
	}
	return fmt.Errorf("no item")
}
