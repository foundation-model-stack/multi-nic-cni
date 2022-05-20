/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

/*
Test Suite for Multi-NIC CNI operator
- Deploy crd
-
*/

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/plugin"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var setupLog = ctrl.Log.WithName("setup")
var hifList map[string]netcogadvisoriov1.HostInterface = generateHostInterfaceList()

var ipvlanPlugin *plugin.IPVLANPlugin
var macvlanPlugin *plugin.MACVLANPlugin
var sriovPlugin *plugin.SriovPlugin

var multinicnetworkReconciler *MultiNicNetworkReconciler

// Multi-NIC IPAM
var globalSubnet string = "192.168.0.0/16"
var multiNicIPAMConfig string = `{
	"type":           "multi-nic-ipam",
	"hostBlock":      8,
	"interfaceBlock": 2,
	"excludeCIDRs":   ["192.168.0.64/26","192.168.0.128/30"],
	"vlanMode":       "l2"
   }`

var networkAddresses []string = []string{"10.242.0.0/24", "10.242.1.0/24"}

// MultiNicNetwork (IPVLAN L2)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), filepath.Join("..", "config", "test", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = netcogadvisoriov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Start controllers
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// initial Logs
	daemonLog := ctrl.Log.WithName("controllers").WithName("Daemon")
	defLog := ctrl.Log.WithName("controllers").WithName("NetAttachDef")
	cidrLog := ctrl.Log.WithName("controllers").WithName("CIDR")
	hifLog := ctrl.Log.WithName("controllers").WithName("HostInterface")
	ippoolLog := ctrl.Log.WithName("controllers").WithName("IPPool")
	networkLog := ctrl.Log.WithName("controllers").WithName("MultiNicNetwork")

	defHandler, err := plugin.GetNetAttachDefHandler(cfg, defLog)
	Expect(err).ToNot(HaveOccurred())
	defHandler.TargetCNI = DEFAULT_MULTI_NIC_CNI_TYPE
	defHandler.DaemonPort = DEFAULT_DAEMON_PORT

	clientset, err := kubernetes.NewForConfig(cfg)
	cidrHandler := NewCIDRHandler(mgr.GetClient(), cfg, cidrLog, hifLog, ippoolLog)

	pluginMap := GetPluginMap(cfg, networkLog)

	// Initialize daemon watcher
	quit := make(chan struct{})
	defer close(quit)
	podQueue := make(chan *v1.Pod, MAX_QSIZE)
	daemonWatcher := NewDaemonWatcher(mgr.GetClient(), cfg, daemonLog, hifLog, cidrHandler, podQueue, quit)
	go daemonWatcher.Run()

	err = (&CIDRReconciler{
		Client:      mgr.GetClient(),
		Log:         cidrLog,
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&HostInterfaceReconciler{
		Client:        mgr.GetClient(),
		Log:           hifLog,
		Scheme:        mgr.GetScheme(),
		DaemonWatcher: daemonWatcher,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&IPPoolReconciler{
		Client:      mgr.GetClient(),
		Log:         ippoolLog,
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&ConfigReconciler{
		Client:              mgr.GetClient(),
		Clientset:           clientset,
		Config:              cfg,
		NetAttachDefHandler: defHandler,
		CIDRHandler:         cidrHandler,
		Log:                 ctrl.Log.WithName("controllers").WithName("Config"),
		DefLog:              defLog,
		Scheme:              mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	multinicnetworkReconciler = &MultiNicNetworkReconciler{
		Client:              mgr.GetClient(),
		NetAttachDefHandler: defHandler,
		CIDRHandler:         cidrHandler,
		Log:                 networkLog,
		Scheme:              mgr.GetScheme(),
		PluginMap:           pluginMap,
	}

	err = (multinicnetworkReconciler).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	//+kubebuilder:scaffold:builder

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

	trueValue := true
	env := v1.EnvVar{
		Name:  "DAEMON_PORT",
		Value: "11000"}
	mnt := netcogadvisoriov1.HostPathMount{
		Name:        "cnibin",
		PodCNIPath:  "/host/opt/cni/bin",
		HostCNIPath: "/opt/cni/bin",
	}
	daemonConfig := &netcogadvisoriov1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-nicd",
		},
		Spec: netcogadvisoriov1.ConfigSpec{
			CNIType:         "multi-nic",
			IPAMType:        "multi-nic-ipam",
			JoinPath:        "/join",
			InterfacePath:   "/interface",
			AddRoutePath:    "/addroute",
			DeleteRoutePath: "/deleteroute",
			Daemon: netcogadvisoriov1.DaemonSpec{
				Image:           "res-cpe-team-docker-local.artifactory.swg-devops.com/net/multi-nic-cni-daemon:v1.0.0-alpha",
				ImagePullSecret: "multi-nic-cni-operator-res-cpe-team-docker-local",
				ImagePullPolicy: "Always",
				SecurityContext: &v1.SecurityContext{
					Privileged: &trueValue,
				},
				Env: []v1.EnvVar{
					env,
				},
				HostPathMounts: []netcogadvisoriov1.HostPathMount{
					mnt,
				},
				DaemonPort: 11000,
			},
		},
	}
	// Deploy daemon config
	Expect(k8sClient.Create(context.TODO(), daemonConfig)).Should(Succeed())
	// Deploy host interface
	for _, hif := range hifList {
		Expect(k8sClient.Create(context.TODO(), &hif)).Should(Succeed())
	}
	// Deploy sriov dependency

	logger := ctrl.Log.WithName("controllers")
	ipvlanPlugin = &plugin.IPVLANPlugin{
		Log: logger,
	}
	macvlanPlugin = &plugin.MACVLANPlugin{
		Log: logger,
	}
	sriovPlugin = &plugin.SriovPlugin{
		Log: logger,
	}
	sriovNamespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: plugin.SRIOV_NAMESPACE,
		},
	}

	plugin.SRIOV_MANIFEST_PATH = "../plugin/template/cni-config"
	Expect(k8sClient.Create(context.TODO(), &sriovNamespace)).Should(Succeed())
	sriovPlugin.Init(cfg)

}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// generateHostInterfaceList generates stub host and interfaces
func generateHostInterfaceList() map[string]netcogadvisoriov1.HostInterface {
	hostNamePrefix := "worker-"
	interfaceNames := []string{"eth1", "eth2"}
	networkPrefixes := []string{"10.242.0.", "10.242.1."}

	hostNum := 5
	hifList := make(map[string]netcogadvisoriov1.HostInterface)
	for i := 0; i < hostNum; i++ {
		hostName := fmt.Sprintf("%s%d", hostNamePrefix, i)
		ifaces := []netcogadvisoriov1.InterfaceInfoType{}
		for index, ifaceName := range interfaceNames {
			iface := netcogadvisoriov1.InterfaceInfoType{
				InterfaceName: ifaceName,
				NetAddress:    networkAddresses[index],
				HostIP:        fmt.Sprintf("%s%d", networkPrefixes[index], i),
			}
			ifaces = append(ifaces, iface)
		}
		hif := netcogadvisoriov1.HostInterface{
			ObjectMeta: metav1.ObjectMeta{
				Name: hostName,
			},
			Spec: netcogadvisoriov1.HostInterfaceSpec{
				HostName:   hostName,
				Interfaces: ifaces,
			},
		}
		hifList[hostName] = hif
	}
	return hifList
}

// getMultiNicCNINetwork returns MultiNicNetwork object
func getMultiNicCNINetwork(cniVersion string, cniType string, cniArgs map[string]string) *netcogadvisoriov1.MultiNicNetwork {
	return &netcogadvisoriov1.MultiNicNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: netcogadvisoriov1.MultiNicNetworkSpec{
			Subnet:         globalSubnet,
			MasterNetAddrs: networkAddresses,
			IPAM:           multiNicIPAMConfig,
			IsMultiNICIPAM: true,
			MainPlugin: netcogadvisoriov1.PluginSpec{
				CNIVersion: cniVersion,
				Type:       cniType,
				CNIArgs:    cniArgs,
			},
		},
	}
}
