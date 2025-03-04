/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package vars

import (
	"os"
	"strconv"
	"time"

	logf "github.com/foundation-model-stack/multi-nic-cni/controllers/logr"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	// environment name definition
	MaxQueueSizeKey   = "MAX_QSIZE"       // daemon pod queue size
	TickerIntervalKey = "TICKER_INTERVAL" // synchronizer ticker interval
	NodeNameKey       = "K8S_NODENAME"

	// common constant
	PodStatusField                            = "status.phase"
	PodStatusRunning                          = "Running"
	JoinLabelName                             = "multi-nicd-join"
	HostNameLabel                             = "hostname"
	DefNameLabel                              = "netname"
	TestModeLabel                             = "test-mode"
	DefaultDaemonPort                         = 11000
	DeamonLabelKey                            = "app"
	DaemonLabelValue                          = "multi-nicd"
	ServiceAccountName                        = "multi-nic-cni-operator-controller-manager"
	DefaultOperatorNamespace                  = "multi-nic-cni-operator"
	DefaultCNIType                            = "multi-nic"
	DefaultIPAMType                           = "multi-nic-ipam"
	DefaultDaemonImage                        = "ghcr.io/foundation-model-stack/multi-nic-cni-daemon:v1.2.6"
	DefaultJoinPath                           = "/join"
	DefaultInterfacePath                      = "/interface"
	DefaultAddRoutePath                       = "/addl3"
	DefaultDeleteRoutePath                    = "/deletel3"
	DefaultUrgentReconcileTime  time.Duration = 5 * time.Second
	DefaultNormalReconcileTime  time.Duration = time.Minute
	DefaultLongReconcileTime    time.Duration = 10 * time.Minute
	DefaultContextTimeout       time.Duration = 2 * time.Minute
	DefaultLogLevel                           = 4
	APIServerToleration                       = 5 // maximum retry if getting error from api server timeout
	APIServerTolerationWaitTime time.Duration = 2 * time.Second

	//	multus-related constants
	MultusLabelKey     = "app"
	MultusLabelValue   = "multus"
	DefaultCNIHostPath = "/var/lib/cni/bin"
	CNIBinVolumeName   = "cnibin"

	// errors
	ConnectionRefusedError = "connection refused"
	NotFoundError          = "not found"
	ThrottlingError        = "throttling"
)

var (
	// var set from environment (cannot be changed on-the-fly by configmap)
	MaxQueueSize   int           = InitIntFromEnv(MaxQueueSizeKey, 100)
	TickerInterval time.Duration = time.Duration(InitIntFromEnv(TickerIntervalKey, 10)) * time.Minute

	// var overrided by config CR
	MultiNICIPAMType    string        = DefaultIPAMType
	TargetCNI           string        = DefaultCNIType
	DaemonPort          int           = DefaultDaemonPort
	UrgentReconcileTime time.Duration = DefaultUrgentReconcileTime
	NormalReconcileTime time.Duration = DefaultNormalReconcileTime
	LongReconcileTime   time.Duration = DefaultLongReconcileTime
	ContextTimeout      time.Duration = DefaultContextTimeout

	// logger options to change log level on the fly
	ZapOpts    *zap.Options
	SetupLog   logr.Logger
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
	ctrl.Log.GetSink()
	zapp := zap.New(zap.UseFlagOptions(ZapOpts))
	dlog := logf.NewDelegatingLogSink(zapp.GetSink())
	ctrl.Log = logr.New(dlog)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(ZapOpts)))
	SetupLog = ctrl.Log.WithName("setup")
	DaemonLog = ctrl.Log.WithName("controllers").WithName("Daemon")
	DefLog = ctrl.Log.WithName("controllers").WithName("NetAttachDef")
	CIDRLog = ctrl.Log.WithName("controllers").WithName("CIDR")
	HifLog = ctrl.Log.WithName("controllers").WithName("HostInterface")
	IPPoolLog = ctrl.Log.WithName("controllers").WithName("IPPool")
	NetworkLog = ctrl.Log.WithName("controllers").WithName("MultiNicNetwork")
	ConfigLog = ctrl.Log.WithName("controllers").WithName("Config")
	SyncLog = ctrl.Log.WithName("controllers").WithName("Synchronizer")
}
