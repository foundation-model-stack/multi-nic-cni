/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package main

import (
	"flag"
	"fmt"
	"os"

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

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	netv1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/controllers"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/plugin"
	//+kubebuilder:scaffold:imports
)

var (
	MAX_QSIZE = 100
	scheme    = runtime.NewScheme()
	setupLog  = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netcogadvisoriov1.AddToScheme(scheme))
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

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5aaf67fd.cogadvisor.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	quit := make(chan struct{})
	defer close(quit)

	daemonLog := ctrl.Log.WithName("controllers").WithName("Daemon")
	defLog := ctrl.Log.WithName("controllers").WithName("NetAttachDef")
	cidrLog := ctrl.Log.WithName("controllers").WithName("CIDR")
	hifLog := ctrl.Log.WithName("controllers").WithName("HostInterface")
	ippoolLog := ctrl.Log.WithName("controllers").WithName("IPPool")
	networkLog := ctrl.Log.WithName("controllers").WithName("MultiNicNetwork")

	defHandler, err := plugin.GetNetAttachDefHandler(config, defLog)
	if err != nil {
		setupLog.Error(err, "unable to create NetworkAttachmentdefinition handler")
		os.Exit(1)
	}
	defHandler.TargetCNI = controllers.DEFAULT_MULTI_NIC_CNI_TYPE
	defHandler.DaemonPort = controllers.DEFAULT_DAEMON_PORT

	clientset, err := kubernetes.NewForConfig(config)
	cidrHandler := controllers.NewCIDRHandler(mgr.GetClient(), config, cidrLog, hifLog, ippoolLog)

	pluginMap := controllers.GetPluginMap(config, networkLog)
	setupLog.Info(fmt.Sprintf("Plugin Map: %v", pluginMap))

	podQueue := make(chan *v1.Pod, MAX_QSIZE)
	daemonWatcher := controllers.NewDaemonWatcher(mgr.GetClient(), config, daemonLog, hifLog, cidrHandler, podQueue, quit)
	go daemonWatcher.Run()

	if err = (&controllers.CIDRReconciler{
		Client:      mgr.GetClient(),
		Log:         cidrLog,
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CIDR")
		os.Exit(1)
	}

	if err = (&controllers.HostInterfaceReconciler{
		Client:        mgr.GetClient(),
		Log:           hifLog,
		Scheme:        mgr.GetScheme(),
		DaemonWatcher: daemonWatcher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HostInterface")
		os.Exit(1)
	}

	if err = (&controllers.IPPoolReconciler{
		Client:      mgr.GetClient(),
		Log:         ippoolLog,
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IPPool")
		os.Exit(1)
	}
	if err = (&controllers.ConfigReconciler{
		Client:              mgr.GetClient(),
		Clientset:           clientset,
		Config:              config,
		CIDRHandler:         cidrHandler,
		NetAttachDefHandler: defHandler,
		Log:                 ctrl.Log.WithName("controllers").WithName("Config"),
		DefLog:              defLog,
		Scheme:              mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Config")
		os.Exit(1)
	}

	if err = (&controllers.MultiNicNetworkReconciler{
		Client:              mgr.GetClient(),
		NetAttachDefHandler: defHandler,
		CIDRHandler:         cidrHandler,
		Log:                 networkLog,
		Scheme:              mgr.GetScheme(),
		PluginMap:           pluginMap,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultiNicNetwork")
		os.Exit(1)
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

	cidrHandler.CleanPreviousCIDR(config, defHandler)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
