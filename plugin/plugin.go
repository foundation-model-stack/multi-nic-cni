/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"encoding/json"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AUTO_GEN_LABEL = "multi-nic-cni-generated"
)

var netConfKeys []string = []string{"cniVersion", "type"}

type Plugin interface {
	GetConfig(net netcogadvisoriov1.MultiNicNetwork, hifList map[string]netcogadvisoriov1.HostInterface) (string, map[string]string, error)
	CleanUp(net netcogadvisoriov1.MultiNicNetwork) error
}

func GetMetaObject(name string, namespace string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			AUTO_GEN_LABEL: "true",
		},
		Annotations: annotations,
	}
}

func RemoveEmpty(args map[string]string, pluginStr string) string {
	var pluginObj map[string]interface{}
	cleanedObj := make(map[string]interface{})
	json.Unmarshal([]byte(pluginStr), &pluginObj)
	for _, key := range netConfKeys {
		cleanedObj[key] = pluginObj[key]
	}

	for key, value := range pluginObj {
		if _, exist := args[key]; exist {
			cleanedObj[key] = value
		}
	}
	cleanedBytes, _ := json.Marshal(cleanedObj)
	return string(cleanedBytes)
}
