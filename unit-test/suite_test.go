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
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
	"github.com/foundation-model-stack/multi-nic-cni/plugin"
	ctrl "sigs.k8s.io/controller-runtime"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	fakeNodeName      = "fake-node"
	fakeDaemonPodName = "fake-multi-nicd"
)

var k8sClient client.Client
var testEnv *envtest.Environment
var nodes []corev1.Node = generateNodes()
var interfaceNames []string = []string{"eth1", "eth2"}
var networkPrefixes []string = []string{"10.242.0.", "10.242.1."}
var hifList map[string]multinicv1.HostInterface = generateHostInterfaceList(nodes)

var ipvlanPlugin *plugin.IPVLANPlugin
var macvlanPlugin *plugin.MACVLANPlugin
var sriovPlugin *plugin.SriovPlugin
var mellanoxPlugin *plugin.MellanoxPlugin

var multinicnetworkReconciler *controllers.MultiNicNetworkReconciler
var configReconciler *controllers.ConfigReconciler
var daemonWatcher *controllers.DaemonWatcher

// Multi-NIC IPAM
var globalSubnet string = "192.168.0.0/16"
var multiNicIPAMConfig string = `{
	"type":           "multi-nic-ipam",
	"hostBlock":      8,
	"interfaceBlock": 2,
	"excludeCIDRs":   ["192.168.0.64/26","192.168.0.128/30"],
	"vlanMode":       "l2"
   }`

var networkAddresses []string = []string{"10.242.0.0/24", "10.242.1.0/24", "10.242.2.0/24", "10.242.1.0/24"}

// MultiNicNetwork (IPVLAN L2)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	opts := zap.Options{
		Development: true,
		DestWriter:  GinkgoWriter,
		Level:       zapcore.Level(int8(-1)),
	}
	vars.ZapOpts = &opts
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(vars.ZapOpts)))
	vars.SetLog()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), filepath.Join("..", "config", "test", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = multinicv1.AddToScheme(scheme.Scheme)
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

	daemonCacheHandler := &controllers.DaemonCacheHandler{SafeCache: controllers.InitSafeCache()}

	quit := make(chan struct{})
	defer close(quit)

	hostInterfaceHandler := controllers.NewHostInterfaceHandler(cfg, mgr.GetClient())

	defHandler, err := plugin.GetNetAttachDefHandler(cfg)
	Expect(err).ToNot(HaveOccurred())

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("Failed to NewForConfig: %v", err)
	}
	cidrHandler := controllers.NewCIDRHandler(mgr.GetClient(), cfg, hostInterfaceHandler, daemonCacheHandler, quit)
	go cidrHandler.Run()

	pluginMap := controllers.GetPluginMap(cfg)

	// Initialize daemon watcher
	podQueue := make(chan *v1.Pod, vars.MaxQueueSize)
	daemonWatcher = controllers.NewDaemonWatcher(mgr.GetClient(), cfg, hostInterfaceHandler, daemonCacheHandler, podQueue, quit)
	go daemonWatcher.Run()

	err = (&controllers.CIDRReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		CIDRHandler:   cidrHandler,
		DaemonWatcher: daemonWatcher,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.HostInterfaceReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		HostInterfaceHandler: hostInterfaceHandler,
		CIDRHandler:          cidrHandler,
		DaemonWatcher:        daemonWatcher,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.IPPoolReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		CIDRHandler: cidrHandler,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	configReconciler = &controllers.ConfigReconciler{
		Client:              mgr.GetClient(),
		Clientset:           clientset,
		Config:              cfg,
		CIDRHandler:         cidrHandler,
		NetAttachDefHandler: defHandler,
		Scheme:              mgr.GetScheme(),
	}
	err = (configReconciler).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	multinicnetworkReconciler = &controllers.MultiNicNetworkReconciler{
		Client:              mgr.GetClient(),
		NetAttachDefHandler: defHandler,
		CIDRHandler:         cidrHandler,
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
	mnt := multinicv1.HostPathMount{
		Name:        "cnibin",
		PodCNIPath:  "/host/opt/cni/bin",
		HostCNIPath: "/opt/cni/bin",
	}
	daemonConfig := &multinicv1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-nicd",
		},
		Spec: multinicv1.ConfigSpec{
			CNIType:         "multi-nic",
			IPAMType:        "multi-nic-ipam",
			JoinPath:        "/join",
			InterfacePath:   "/interface",
			AddRoutePath:    "/addroute",
			DeleteRoutePath: "/deleteroute",
			Daemon: multinicv1.DaemonSpec{
				Image:           vars.DefaultDaemonImage,
				ImagePullPolicy: "Always",
				SecurityContext: &v1.SecurityContext{
					Privileged: &trueValue,
				},
				Env: []v1.EnvVar{
					env,
				},
				HostPathMounts: []multinicv1.HostPathMount{
					mnt,
				},
				DaemonPort: 11000,
			},
		},
	}

	operatorNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: controllers.OPERATOR_NAMESPACE,
		},
	}
	Expect(k8sClient.Create(context.TODO(), &operatorNamespace)).Should(Succeed())

	// Deploy daemon config
	Expect(k8sClient.Create(context.TODO(), daemonConfig)).Should(Succeed())
	// Deploy host interface
	for _, hif := range hifList {
		Expect(k8sClient.Create(context.TODO(), &hif)).Should(Succeed())
		cidrHandler.HostInterfaceHandler.SetCache(hif.Spec.HostName, hif)
	}

	// Deploy daemon pod
	daemonPod := newDaemonPod(daemonConfig.Spec.Daemon)
	Expect(k8sClient.Create(context.TODO(), daemonPod)).Should(Succeed())
	Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: daemonPod.Name, Namespace: daemonPod.Namespace}, daemonPod)).Should(Succeed())
	updatePodReadyStatus(daemonPod)
	Expect(k8sClient.Get(context.TODO(), types.NamespacedName{Name: daemonPod.Name, Namespace: daemonPod.Namespace}, daemonPod)).Should(Succeed())
	Expect(controllers.IsContainerReady(*daemonPod)).To(Equal(true))

	// Deploy sriov dependency
	ipvlanPlugin = &plugin.IPVLANPlugin{}
	macvlanPlugin = &plugin.MACVLANPlugin{}
	sriovPlugin = &plugin.SriovPlugin{}
	sriovNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: plugin.SRIOV_NAMESPACE,
		},
	}

	plugin.SRIOV_MANIFEST_PATH = "../plugin/template/cni-config"
	Expect(k8sClient.Create(context.TODO(), &sriovNamespace)).Should(Succeed())
	err = sriovPlugin.Init(cfg)
	Expect(err).ToNot(HaveOccurred())

	// Deploy mellanox dependency
	mellanoxPlugin = &plugin.MellanoxPlugin{}
	err = mellanoxPlugin.Init(cfg)
	Expect(err).ToNot(HaveOccurred())
	sriovResourceList := `
		{
			"resourceList": [
				{
					"resourcePrefix": "nvidia.com",
					"resourceName": "host_dev",
					"selectors": {
						"vendors": ["15b3"],
						"isRdma": true
					}
				}
			]
		}
	`
	nicClusterPolicy := plugin.NicClusterPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: plugin.MELLANOX_API_VERSION,
			Kind:       plugin.MELLANOX_POLICY_KIND,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nic-cluster-policy",
		},
		Spec: plugin.NicClusterPolicySpec{
			SriovDevicePlugin: &plugin.DevicePluginSpec{
				ImageSpecWithConfig: plugin.ImageSpecWithConfig{
					Config: &sriovResourceList,
					ImageSpec: plugin.ImageSpec{
						Image:            "sriov-network-device-plugin",
						Repository:       "ghcr.io/k8snetworkplumbingwg",
						Version:          "v3.5.1",
						ImagePullSecrets: []string{},
					},
				},
			},
		},
	}
	createdNicClusterPolicy := &plugin.NicClusterPolicy{}
	Expect(mellanoxPlugin.MellanoxNicClusterPolicyHandler.Create(metav1.NamespaceAll, nicClusterPolicy, createdNicClusterPolicy)).Should(Succeed())
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func generateNodes() []corev1.Node {
	nodes := []corev1.Node{}
	hostNamePrefix := "worker-"
	hostNum := 5

	for i := 0; i < hostNum; i++ {
		hostName := fmt.Sprintf("%s%d", hostNamePrefix, i)
		node := corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: hostName,
			},
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// generateNewHostInterface generates new host
func generateNewHostInterface(hostName string, interfaceNames []string, networkPrefixes []string, i int) multinicv1.HostInterface {
	ifaces := []multinicv1.InterfaceInfoType{}
	for index, ifaceName := range interfaceNames {
		iface := multinicv1.InterfaceInfoType{
			InterfaceName: ifaceName,
			NetAddress:    networkAddresses[index],
			HostIP:        fmt.Sprintf("%s%d", networkPrefixes[index], i),
		}
		ifaces = append(ifaces, iface)
	}
	hif := multinicv1.HostInterface{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostName,
			Labels: map[string]string{
				vars.TestModeLabel: "true",
			},
		},
		Spec: multinicv1.HostInterfaceSpec{
			HostName:   hostName,
			Interfaces: ifaces,
		},
	}
	return hif
}

// generateHostInterfaceList generates stub host and interfaces
func generateHostInterfaceList(nodes []corev1.Node) map[string]multinicv1.HostInterface {

	hifList := make(map[string]multinicv1.HostInterface)
	for i, node := range nodes {
		hostName := node.GetName()
		hif := generateNewHostInterface(hostName, interfaceNames, networkPrefixes, i)
		hifList[hostName] = hif
	}
	return hifList
}

// getMultiNicCNINetwork returns MultiNicNetwork object
func getMultiNicCNINetwork(name string, cniVersion string, cniType string, cniArgs map[string]string) *multinicv1.MultiNicNetwork {
	return &multinicv1.MultiNicNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: multinicv1.MultiNicNetworkSpec{
			Subnet:         globalSubnet,
			IPAM:           multiNicIPAMConfig,
			IsMultiNICIPAM: true,
			MainPlugin: multinicv1.PluginSpec{
				CNIVersion: cniVersion,
				Type:       cniType,
				CNIArgs:    cniArgs,
			},
			MasterNetAddrs: networkAddresses,
		},
	}
}

// newDaemonPod creates new daemonPod
func newDaemonPod(daemonSpec multinicv1.DaemonSpec) *corev1.Pod {
	labels := map[string]string{vars.DeamonLabelKey: vars.DaemonLabelValue}

	// prepare container port
	containerPort := corev1.ContainerPort{ContainerPort: int32(daemonSpec.DaemonPort)}
	mnts := daemonSpec.HostPathMounts
	vmnts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	for _, mnt := range mnts {
		// prepare volume mounting
		vmnt := corev1.VolumeMount{
			Name:      mnt.Name,
			MountPath: mnt.PodCNIPath,
		}
		hostSource := &corev1.HostPathVolumeSource{Path: mnt.HostCNIPath}
		volume := corev1.Volume{
			Name: mnt.Name,
			VolumeSource: corev1.VolumeSource{
				HostPath: hostSource,
			},
		}
		vmnts = append(vmnts, vmnt)
		volumes = append(volumes, volume)
	}
	// hostName environment
	hostNameVar := corev1.EnvVar{
		Name: vars.NodeNameKey,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "spec.nodeName",
			},
		},
	}
	daemonSpec.Env = append(daemonSpec.Env, hostNameVar)

	// prepare secret
	secrets := []corev1.LocalObjectReference{}
	if daemonSpec.ImagePullSecret != "" {
		secret := corev1.LocalObjectReference{
			Name: daemonSpec.ImagePullSecret,
		}
		secrets = append(secrets, secret)
	}
	// prepare container
	container := corev1.Container{
		Name:  "daemon",
		Image: daemonSpec.Image,
		Ports: []corev1.ContainerPort{
			containerPort,
		},
		EnvFrom:         daemonSpec.EnvFrom,
		Env:             daemonSpec.Env,
		Resources:       daemonSpec.Resources,
		VolumeMounts:    vmnts,
		ImagePullPolicy: corev1.PullPolicy(daemonSpec.ImagePullPolicy),
		SecurityContext: daemonSpec.SecurityContext,
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fakeDaemonPodName,
			Namespace: controllers.OPERATOR_NAMESPACE,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			PriorityClassName:  "system-cluster-critical",
			HostNetwork:        true,
			ServiceAccountName: vars.ServiceAccountName,
			NodeSelector:       daemonSpec.NodeSelector,
			Tolerations:        daemonSpec.Tolerations,
			Containers: []corev1.Container{
				container,
			},
			Volumes:          volumes,
			ImagePullSecrets: secrets,
			NodeName:         fakeNodeName,
		},
	}
}

func updatePodReadyStatus(pod *corev1.Pod) {
	readyStatus := v1.ContainerStatus{
		Ready: true,
	}
	pod.Status.ContainerStatuses = []v1.ContainerStatus{readyStatus}
	Expect(k8sClient.Status().Update(context.TODO(), pod)).Should(Succeed())
	Expect(controllers.IsContainerReady(*pod)).To(Equal(true))
}
