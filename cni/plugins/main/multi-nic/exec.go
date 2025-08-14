/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"

	current "github.com/containernetworking/cni/pkg/types/100"
)

var defaultExec = &invoke.DefaultExec{
	RawExec: &invoke.RawExec{Stderr: os.Stderr},
}

func execPlugin(plugin string, command string, confBytes []byte, args *skel.CmdArgs, ifName string, withResult bool) (*current.Result, error) {
	cniPath := os.Getenv("CNI_PATH")
	singleNicArgs := &invoke.Args{
		Command:       command,
		ContainerID:   args.ContainerID,
		NetNS:         args.Netns,
		IfName:        ifName,
		PluginArgsStr: args.Args,
		Path:          cniPath,
	}
	paths := filepath.SplitList(cniPath)
	pluginPath, err := defaultExec.FindInPath(plugin, paths)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if withResult {
		r, err := invoke.ExecPluginWithResult(ctx, pluginPath, confBytes, singleNicArgs, defaultExec)
		if err != nil {
			return nil, err
		}
		return current.NewResultFromResult(r)
	} else {
		err = invoke.ExecPluginWithoutResult(ctx, pluginPath, confBytes, singleNicArgs, defaultExec)
		return nil, err
	}
}
