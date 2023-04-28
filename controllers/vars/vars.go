/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package vars

import (
	"os"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	// environment name definition
	MaxQueueSizeKey   = "MAX_QSIZE"       // daemon pod queue size
	TickerIntervalKey = "TICKER_INTERVAL" // synchronizer ticker interval
	NodeNameKey       = "K8S_NODENAME"

	// common constant
	PodStatusField    = "status.phase"
	PodStatusRunning  = "Running"
	JoinLabelName     = "multi-nicd-join"
	HostNameLabel     = "hostname"
	DefNameLabel      = "netname"
	TestModeLabel     = "test-mode"
	DefaultDaemonPort = 11000

	// errors
	ConnectionRefusedError = "connection refused"
	NotFoundError          = "not found"
)

var (
	// var set from environment (cannot be changed on-the-fly by configmap)
	MaxQueueSize   int           = InitIntFromEnv(MaxQueueSizeKey, 100)
	TickerInterval time.Duration = time.Duration(InitIntFromEnv(TickerIntervalKey, 10)) * time.Minute

	// var overrided by config CR
	MultiNICIPAMType    string        = "multi-nic-ipam"
	TargetCNI           string        = "multi-nic"
	DaemonPort          int           = 11000
	UrgentReconcileTime time.Duration = 5 * time.Second
	NormalReconcileTime time.Duration = time.Minute
	LongReconcileTime   time.Duration = 10 * time.Minute
	ContextTimeout      time.Duration = 2 * time.Minute

	// logger options to change log level on the fly
	ZapOpts    *zap.Options
	DaemonLog  logr.Logger
	DefLog     logr.Logger
	CIDRLog    logr.Logger
	HifLog     logr.Logger
	IPPoolLog  logr.Logger
	NetworkLog logr.Logger
	ConfigLog  logr.Logger
	SyncLog    logr.Logger
)

// InitIntFromEnv initialize int value from environment key or set to default if not set or invalid
func InitIntFromEnv(key string, defaultValue int) int {
	strValue, present := os.LookupEnv(key)
	if present {
		value, err := strconv.Atoi(strValue)
		if err == nil {
			return value
		}
	}
	return defaultValue
}

func SetLog() {
	zapp := zap.New(zap.UseFlagOptions(ZapOpts))
	dlog := logf.NewDelegatingLogSink(zapp.GetSink())
	ctrl.Log = logr.New(dlog)
	DaemonLog = ctrl.Log.WithName("controllers").WithName("Daemon")
	DefLog = ctrl.Log.WithName("controllers").WithName("NetAttachDef")
	CIDRLog = ctrl.Log.WithName("controllers").WithName("CIDR")
	HifLog = ctrl.Log.WithName("controllers").WithName("HostInterface")
	IPPoolLog = ctrl.Log.WithName("controllers").WithName("IPPool")
	NetworkLog = ctrl.Log.WithName("controllers").WithName("MultiNicNetwork")
	ConfigLog = ctrl.Log.WithName("controllers").WithName("Config")
	SyncLog = ctrl.Log.WithName("controllers").WithName("Synchronizer")
}
