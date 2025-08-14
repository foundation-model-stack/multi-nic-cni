/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin

import (
	"encoding/json"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AUTO_GEN_LABEL = "multi-nic-cni-generated"
)

var netConfKeys []string = []string{"cniVersion", "type"}

type Plugin interface {
	GetConfig(net multinicv1.MultiNicNetwork, hifList map[string]multinicv1.HostInterface) (string, map[string]string, error)
	CleanUp(net multinicv1.MultiNicNetwork) error
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

// RemoveEmpty constructs a new config string based on netConfKeys and valid argument list.
func RemoveEmpty(args map[string]string, pluginStr string) string {
	var pluginObj map[string]interface{}
	cleanedObj := make(map[string]interface{})
	err := json.Unmarshal([]byte(pluginStr), &pluginObj)
	if err != nil {
		// cannot check, return original value
		return pluginStr
	}
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
