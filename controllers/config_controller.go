/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/plugin"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"os"
)

const (
	SERVICE_ACCOUNT_NAME       = "multi-nic-cni-operator-controller-manager"
	DEFAULT_OPERATOR_NAMESPACE = "multi-nic-cni-operator-system"

	// NetworkAttachmentDefinition watching queue size
	MAX_QSIZE = 100

	ConfigWaitingReconcileTime = 5 * time.Second
)

var (
	OPERATOR_NAMESPACE string = getOperatorNamespace()
	ConfigReady        bool   = false
	// referred by daemon watcher
	DAEMON_LABEL_NAME         = "app"
	DAEMON_LABEL_VALUE        = "multi-nicd"
	DaemonName         string = DAEMON_LABEL_VALUE
)

func getOperatorNamespace() string {
	key := "OPERATOR_NAMESPACE"
	val, found := os.LookupEnv(key)
	if !found {
		return DEFAULT_OPERATOR_NAMESPACE
	}
	return val
}

// ConfigReconciler reconciles a Config object
// - if Config is deleted, delete daemon
// - create CNI daemonset, and NetworkAttachmentDefinition watcher if not exist
// - otherwise, update the existing daemonset and restart NetworkAttachmentDefinition watcher

type ConfigReconciler struct {
	client.Client
	*kubernetes.Clientset
	*rest.Config
	*CIDRHandler
	*plugin.NetAttachDefHandler
	DefLog logr.Logger
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=configs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=configs/status,verbs=get;update;patch

const ReconcileTime = 30 * time.Minute

func (r *ConfigReconciler) CreateDefaultDaemonConfig() error {
	objMeta := metav1.ObjectMeta{
		Name: DaemonName,
	}
	daemonEnv := corev1.EnvVar{
		Name:  "DAEMON_PORT",
		Value: "11000",
	}
	routeEnv := corev1.EnvVar{
		Name:  "RT_TABLE_PATH",
		Value: "/opt/rt_tables",
	}
	env := []corev1.EnvVar{daemonEnv, routeEnv}
	binMnt := multinicv1.HostPathMount{
		Name:        "cnibin",
		PodCNIPath:  "/host/opt/cni/bin",
		HostCNIPath: "/var/lib/cni/bin",
	}
	devPluginMnt := multinicv1.HostPathMount{
		Name:        "device-plugin",
		PodCNIPath:  "/var/lib/kubelet/device-plugins",
		HostCNIPath: "/var/lib/kubelet/device-plugins",
	}
	routeMnt := multinicv1.HostPathMount{
		Name:        "rt-tables",
		PodCNIPath:  "/opt/rt_tables",
		HostCNIPath: "/etc/iproute2/rt_tables",
	}
	hostPathMounts := []multinicv1.HostPathMount{binMnt, devPluginMnt, routeMnt}
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	}
	var privileged bool = true
	securityContext := &corev1.SecurityContext{
		Privileged: &privileged,
	}
	daemonSpec := multinicv1.DaemonSpec{
		Image:           "ghcr.io/foundation-model-stack/multi-nic-cni-daemon:v1.0.2",
		Env:             env,
		HostPathMounts:  hostPathMounts,
		Resources:       resources,
		SecurityContext: securityContext,
		DaemonPort:      11000,
	}
	spec := multinicv1.ConfigSpec{
		CNIType:         "multi-nic",
		IPAMType:        "multi-nic-ipam",
		Daemon:          daemonSpec,
		JoinPath:        "/join",
		InterfacePath:   "/interface",
		AddRoutePath:    "/addl3",
		DeleteRoutePath: "/deletel3",
	}
	cfg := &multinicv1.Config{
		ObjectMeta: objMeta,
		Spec:       spec,
	}
	err := r.Client.Create(context.TODO(), cfg)
	return err
}

func (r *ConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("config", req.NamespacedName)

	instance := &multinicv1.Config{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// - if Config is deleted, delete daemon
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.callFinalizer(r.Log, req.NamespacedName.Name)
			return ctrl.Result{}, nil
		}
		r.Log.V(7).Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: ReconcileTime}, nil
	}
	if !ConfigReady {
		r.CIDRHandler.SyncAllPendingCustomCR(r.NetAttachDefHandler)
		r.Log.Info("Set ConfigReady")
		ConfigReady = true
		// initial run
		r.CIDRHandler.UpdateCIDRs()
	}

	dsName := instance.GetName()
	_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Get(context.TODO(), dsName, metav1.GetOptions{})
	daemonset := r.newCNIDaemonSet(r.Clientset, dsName, instance.Spec.Daemon)
	if err != nil {
		// - create CNI daemonset, and NetworkAttachmentDefinition watcher if not exist
		if errors.IsNotFound(err) {
			r.newNetAttachDefWatcher(instance)
			_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Create(context.TODO(), daemonset, metav1.CreateOptions{})
			r.Log.V(7).Info(fmt.Sprintf("Create new multi-nic daemonset #%s (cni=%s,ipam=%s), err=%v ", dsName, instance.Spec.CNIType, instance.Spec.IPAMType, err))
		} else {
			r.Log.Info(fmt.Sprintf("Cannot get daemonset #%v ", err))
		}
	} else {
		// - otherwise, update the existing daemonset and restart NetworkAttachmentDefinition watcher
		r.newNetAttachDefWatcher(instance)
		_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Update(context.TODO(), daemonset, metav1.UpdateOptions{})
		r.Log.V(7).Info(fmt.Sprintf("Update multi-nic daemonset #%s, err=%v ", dsName, err))
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.Config{}).
		Complete(r)
}

// newNetAttachDefWatcher restarts NetworkAttachmentDefinition watcher
func (r *ConfigReconciler) newNetAttachDefWatcher(instance *multinicv1.Config) {
	r.NetAttachDefHandler.DaemonPort = instance.Spec.Daemon.DaemonPort
	r.NetAttachDefHandler.TargetCNI = instance.Spec.CNIType
	DaemonName = instance.GetName()
	SetDaemon(instance.Spec)
}

// newCNIDaemonSet creates new CNI daemonset
func (r *ConfigReconciler) newCNIDaemonSet(client *kubernetes.Clientset, name string, daemonSpec multinicv1.DaemonSpec) *appsv1.DaemonSet {
	labels := map[string]string{DAEMON_LABEL_NAME: DAEMON_LABEL_VALUE}

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

	// prepare secret
	secrets := []corev1.LocalObjectReference{}
	if daemonSpec.ImagePullSecret == "" {
		secret := corev1.LocalObjectReference{
			Name: daemonSpec.ImagePullSecret,
		}
		secrets = append(secrets, secret)
	}
	// prepare container
	container := corev1.Container{
		Name:  name,
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

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: OPERATOR_NAMESPACE,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					HostNetwork:        true,
					ServiceAccountName: SERVICE_ACCOUNT_NAME,
					NodeSelector:       daemonSpec.NodeSelector,
					Tolerations:        daemonSpec.Tolerations,
					Containers: []corev1.Container{
						container,
					},
					Volumes:          volumes,
					ImagePullSecrets: secrets,
				},
			},
		},
	}
}

// callFinalizer deletes all CIDRs, waits for all ippools deleted, deletes CNI deamonset, and stops NetworkAttachmentDefinition watcher
func (r *ConfigReconciler) callFinalizer(reqLogger logr.Logger, dsName string) error {
	// reset default name
	DaemonName = DAEMON_LABEL_VALUE
	reqLogger.Info(fmt.Sprintf("Finalize %s", dsName))

	// delete all CIDRs
	cidrMap, err := r.CIDRHandler.ListCIDR()
	if err == nil {
		for _, cidr := range cidrMap {
			r.CIDRHandler.DeleteCIDR(cidr)
		}
	}
	// wait for all ippools deleted
	for {
		if err != nil || r.CIDRHandler.IPPoolHandler.SafeCache.GetSize() == 0 {
			break
		}
		reqLogger.V(5).Info(fmt.Sprintf("%d ippools left, wait...", r.CIDRHandler.IPPoolHandler.SafeCache.GetSize()))
		time.Sleep(1 * time.Second)
	}
	// delete CNI deamonset
	err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Delete(context.TODO(), dsName, metav1.DeleteOptions{})
	ConfigReady = false
	return nil
}
