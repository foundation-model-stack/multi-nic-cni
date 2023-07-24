/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	netv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
	"github.com/foundation-model-stack/multi-nic-cni/plugin"
	"github.com/operator-framework/operator-lib/leader"
	//+kubebuilder:scaffold:imports
)

var (
	scheme                           = runtime.NewScheme()
	setupLog                         = ctrl.Log.WithName("setup")
	MultiNicNetworkReconcilerPointer *controllers.MultiNicNetworkReconciler
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(multinicv1.AddToScheme(scheme))
	utilruntime.Must(netv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	vars.ZapOpts = &opts
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(vars.ZapOpts)))
	vars.SetLog()

	config := ctrl.GetConfigOrDie()

	leaseDuration := 30 * time.Second
	renewDeadline := 20 * time.Second

	if !enableLeaderElection {
		// Become the leader before proceeding
		err := leader.Become(context.TODO(), "5aaf67fd.fms.io")
		if err != nil {
			setupLog.Error(err, "cannot become leader")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5aaf67fd.fms.io",
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	quit := make(chan struct{})
	defer close(quit)

	daemonCacheHandler := &controllers.DaemonCacheHandler{
		SafeCache: controllers.InitSafeCache(),
	}

	hostInterfaceHandler := controllers.NewHostInterfaceHandler(config, mgr.GetClient())

	defHandler, err := plugin.GetNetAttachDefHandler(config)
	if err != nil {
		setupLog.Error(err, "unable to create NetworkAttachmentdefinition handler")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "unable to init clientset")
		os.Exit(1)
	}

	cidrHandler := controllers.NewCIDRHandler(mgr.GetClient(), config, hostInterfaceHandler, daemonCacheHandler, quit)
	go cidrHandler.Run()

	pluginMap := controllers.GetPluginMap(config)
	setupLog.V(2).Info(fmt.Sprintf("Plugin Map: %v", pluginMap))

	podQueue := make(chan *v1.Pod, vars.MaxQueueSize)
	setupLog.V(7).Info("New Daemon Watcher")
	daemonWatcher := controllers.NewDaemonWatcher(mgr.GetClient(), config, hostInterfaceHandler, daemonCacheHandler, podQueue, quit)
	setupLog.V(7).Info("Run Daemon Watcher")
	go daemonWatcher.Run()
	setupLog.V(7).Info("New Reconcilers")

	cidrReconciler := &controllers.CIDRReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		CIDRHandler:   cidrHandler,
		DaemonWatcher: daemonWatcher,
	}
	if err = (cidrReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CIDR")
		os.Exit(1)
	}
	hostInterfaceReconciler := &controllers.HostInterfaceReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		DaemonWatcher:        daemonWatcher,
		HostInterfaceHandler: hostInterfaceHandler,
		CIDRHandler:          cidrHandler,
	}
	if err = (hostInterfaceReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HostInterface")
		os.Exit(1)
	}

	ippoolReconciler := &controllers.IPPoolReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}
	if err = (ippoolReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IPPool")
		os.Exit(1)
	}
	MultiNicNetworkReconcilerPointer = &controllers.MultiNicNetworkReconciler{
		Client:              mgr.GetClient(),
		NetAttachDefHandler: defHandler,
		CIDRHandler:         cidrHandler,
		Scheme:              mgr.GetScheme(),
		PluginMap:           pluginMap,
	}
	if err = (MultiNicNetworkReconcilerPointer).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiNicNetwork")
		os.Exit(1)
	}
	cfgReconciler := &controllers.ConfigReconciler{
		Client:              mgr.GetClient(),
		Clientset:           clientset,
		Config:              config,
		CIDRHandler:         cidrHandler,
		NetAttachDefHandler: defHandler,
		Scheme:              mgr.GetScheme(),
	}
	if err = (cfgReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Config")
		os.Exit(1)
	}
	err = cfgReconciler.CreateDefaultDaemonConfig()
	if err != nil {
		setupLog.Info(fmt.Sprintf("fail to create default config: %v", err))
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	ticker := time.NewTicker(vars.TickerInterval)
	defer ticker.Stop()

	controllers.RunPeriodicUpdate(ticker, daemonWatcher, cidrHandler, hostInterfaceReconciler, quit)

	setupLog.V(7).Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
