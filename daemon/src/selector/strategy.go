/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package selector

type Strategy string

const (
	None     Strategy = "none"
	CostOpt           = "costOpt"
	PerfOpt           = "perfOpt"
	DevClass          = "devClass"
	Topology          = "topology"
)
