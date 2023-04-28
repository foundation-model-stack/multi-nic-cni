/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package plugin

import (
	"fmt"
	"strconv"
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
