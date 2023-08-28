/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package plugin

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	RESOURCE_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
	HOLDER_SUFFIX       = "-net"
)

func getInt(values map[string]string, key string) (int, error) {
	if value, exists := values[key]; exists {
		return strconv.Atoi(value)
	}
	return 0, fmt.Errorf("unset")
}

func getBoolean(values map[string]string, key string) (bool, error) {
	if value, exists := values[key]; exists {
		return strconv.ParseBool(value)
	}
	return false, fmt.Errorf("unset")
}

func ValidateResourceName(name string) string {
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

func GetHolderNetName(name string) string {
	return name + HOLDER_SUFFIX
}
