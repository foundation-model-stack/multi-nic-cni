/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"os"
)

const configFinalizer = "finalizers.config.multinic.fms.io"

var (
	OPERATOR_NAMESPACE string = getOperatorNamespace()
	ConfigReady        bool   = false
	// referred by daemon watcher
	DaemonName string = vars.DaemonLabelValue
)

func getOperatorNamespace() string {
	key := "OPERATOR_NAMESPACE"
	val, found := os.LookupEnv(key)
	if !found {
		return vars.DefaultOperatorNamespace
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
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=configs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=configs/status,verbs=get;update;patch

func (r *ConfigReconciler) getDefaultConfigSpec() multinicv1.ConfigSpec {
	daemonEnv := corev1.EnvVar{
		Name:  "DAEMON_PORT",
		Value: fmt.Sprintf("%d", vars.DefaultDaemonPort),
	}
	routeEnv := corev1.EnvVar{
		Name:  "RT_TABLE_PATH",
		Value: "/opt/rt_tables",
	}
	env := []corev1.EnvVar{daemonEnv, routeEnv}
	binMnt := multinicv1.HostPathMount{
		Name:        "cnibin",
		PodCNIPath:  "/host/opt/cni/bin",
		HostCNIPath: r.getCNIHostPath(),
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
	hwDataMnt := multinicv1.HostPathMount{
		Name:        "hwdata",
		PodCNIPath:  "/usr/share/hwdata",
		HostCNIPath: "/usr/share/hwdata",
	}
	hostPathMounts := []multinicv1.HostPathMount{binMnt, devPluginMnt, routeMnt, hwDataMnt}
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
		Image:           vars.DefaultDaemonImage,
		Env:             env,
		HostPathMounts:  hostPathMounts,
		Resources:       resources,
		SecurityContext: securityContext,
		DaemonPort:      vars.DefaultDaemonPort,
	}
	spec := multinicv1.ConfigSpec{
		CNIType:         vars.DefaultCNIType,
		IPAMType:        vars.DefaultIPAMType,
		Daemon:          daemonSpec,
		JoinPath:        vars.DefaultJoinPath,
		InterfacePath:   vars.DefaultInterfacePath,
		AddRoutePath:    vars.DefaultAddRoutePath,
		DeleteRoutePath: vars.DefaultDeleteRoutePath,
	}
	return spec
}

func (r *ConfigReconciler) CreateDefaultDaemonConfig() error {
	objMeta := metav1.ObjectMeta{
		Name: DaemonName,
	}

	cfg := &multinicv1.Config{
		ObjectMeta: objMeta,
		Spec:       r.getDefaultConfigSpec(),
	}
	err := r.Client.Create(context.TODO(), cfg)
	return err
}

func (r *ConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = vars.ConfigLog.WithValues("config", req.NamespacedName)

	instance := &multinicv1.Config{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// - if Config is deleted, delete daemon
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		vars.ConfigLog.V(7).Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: vars.LongReconcileTime}, nil
	}

	// Add finalizer to instance
	if !controllerutil.ContainsFinalizer(instance, configFinalizer) {
		controllerutil.AddFinalizer(instance, configFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// If config is deleted, delete corresponding daemonset
	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		if controllerutil.ContainsFinalizer(instance, configFinalizer) {
			if err = r.callFinalizer(vars.ConfigLog, req.NamespacedName.Name); err != nil {
				return ctrl.Result{}, err
			}
			err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				err := r.Client.Get(ctx, req.NamespacedName, instance)
				if err != nil {
					if errors.IsNotFound(err) {
						return nil
					}
					return err
				}
				controllerutil.RemoveFinalizer(instance, configFinalizer)
				return r.Client.Update(ctx, instance)
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	r.UpdateConfigBySpec(&instance.Spec)

	if !ConfigReady {
		r.CIDRHandler.SyncAllPendingCustomCR(r.NetAttachDefHandler)
		vars.ConfigLog.Info("Set ConfigReady")
		ConfigReady = true
		// initial run
		r.CIDRHandler.UpdateCIDRs()
	}

	dsName := instance.GetName()
	_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Get(context.TODO(), dsName, metav1.GetOptions{})
	daemonset := r.newCNIDaemonSet(r.Clientset, dsName, instance.Spec.Daemon)
	if err != nil {
		// - create CNI daemonset
		if errors.IsNotFound(err) {
			SetDaemonConnector(instance.Spec)
			_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Create(context.TODO(), daemonset, metav1.CreateOptions{})
			vars.ConfigLog.V(7).Info(fmt.Sprintf("Create new multi-nic daemonset #%s (cni=%s,ipam=%s), err=%v ", dsName, instance.Spec.CNIType, instance.Spec.IPAMType, err))
		} else {
			vars.ConfigLog.Info(fmt.Sprintf("Cannot get daemonset #%v ", err))
		}
	} else {
		// - otherwise, update the existing daemonset
		SetDaemonConnector(instance.Spec)
		_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Update(context.TODO(), daemonset, metav1.UpdateOptions{})
		vars.ConfigLog.V(7).Info(fmt.Sprintf("Update multi-nic daemonset #%s, err=%v ", dsName, err))
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.Config{}).
		Complete(r)
}

// UpdateConfigBySpec updates configuration variables defined in the spec
func (r *ConfigReconciler) UpdateConfigBySpec(spec *multinicv1.ConfigSpec) {
	// set configurations
	if spec.UrgentReconcileSeconds > 0 {
		vars.ConfigLog.Info(fmt.Sprintf("Configure UrgentReconcileSeconds = %d", spec.UrgentReconcileSeconds))
		vars.UrgentReconcileTime = time.Duration(spec.UrgentReconcileSeconds) * time.Second
	}
	if spec.NormalReconcileMinutes > 0 {
		vars.ConfigLog.Info(fmt.Sprintf("Configure NormalReconcileMinutes = %d", spec.NormalReconcileMinutes))
		vars.NormalReconcileTime = time.Duration(spec.NormalReconcileMinutes) * time.Minute
	}
	if spec.LongReconcileMinutes > 0 {
		vars.ConfigLog.Info(fmt.Sprintf("Configure LongReconcileMinutes = %d", spec.LongReconcileMinutes))
		vars.LongReconcileTime = time.Duration(spec.LongReconcileMinutes) * time.Minute
	}
	if spec.ContextTimeoutMinutes > 0 {
		vars.ConfigLog.Info(fmt.Sprintf("Configure ContextTimeoutMinutes = %d", spec.ContextTimeoutMinutes))
		vars.ContextTimeout = time.Duration(spec.ContextTimeoutMinutes) * time.Minute
	}
	if spec.LogLevel >= 1 && spec.LogLevel <= 127 {
		if !vars.ConfigLog.V(spec.LogLevel).Enabled() {
			vars.ConfigLog.Info(fmt.Sprintf("Configure LogLevel = %d", spec.LogLevel))
			intLevel := -1 * spec.LogLevel
			zaplevel := zapcore.Level(int8(intLevel))
			vars.ZapOpts.Level = zaplevel
			vars.SetLog()
		}
	}
	vars.DaemonPort = spec.Daemon.DaemonPort
	vars.TargetCNI = spec.CNIType
}

// newCNIDaemonSet creates new CNI daemonset
func (r *ConfigReconciler) newCNIDaemonSet(client *kubernetes.Clientset, name string, daemonSpec multinicv1.DaemonSpec) *appsv1.DaemonSet {
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
				},
			},
		},
	}
}

func (r *ConfigReconciler) getCNIHostPath() string {
	ctx, cancel := context.WithTimeout(context.Background(), vars.ContextTimeout)
	defer cancel()
	// find multus pod
	labels := fmt.Sprintf("%s=%s", vars.MultusLabelKey, vars.MultusLabelValue)
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
		Limit:         1,
	}
	multusPods, err := r.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
	if err != nil {
		return vars.DefaultCNIHostPath
	}
	for _, multusPod := range multusPods.Items {
		volumes := multusPod.Spec.Volumes
		for _, volume := range volumes {
			if volume.Name == vars.CNIBinVolumeName {
				if volume.HostPath != nil {
					cniPath := volume.HostPath.Path
					vars.ConfigLog.Info(fmt.Sprintf("Find Multus CNI path: %s", cniPath))
					return cniPath
				} else {
					// hostpath is not defined
					return vars.DefaultCNIHostPath
				}
			}
		}
	}
	return vars.DefaultCNIHostPath
}

// callFinalizer deletes all CIDRs, waits for all ippools deleted, deletes CNI deamonset, and stops NetworkAttachmentDefinition watcher
func (r *ConfigReconciler) callFinalizer(reqLogger logr.Logger, dsName string) error {
	reqLogger.Info(fmt.Sprintf("Finalize %s", dsName))
	// delete all CIDRs
	cidrMap, err := r.CIDRHandler.ListCIDR()
	if err == nil {
		for cidrName, cidr := range cidrMap {
			err := r.CIDRHandler.DeleteCIDR(cidr)
			if err != nil {
				reqLogger.V(3).Info(fmt.Sprintf("Failed to delete CIDR %s: %v", cidrName, err))
			}
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
	if err != nil {
		vars.ConfigLog.Info(fmt.Sprintf("Failed to finalize %s: %v", dsName, err))
	}
	return nil
}
