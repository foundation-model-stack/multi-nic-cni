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

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
	"github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/plugin"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	SERVICE_ACCOUNT_NAME = "multi-nic-cni-operator-controller-manager"
	OPERATOR_NAMESPACE   = "multi-nic-cni-operator-system"

	// NetworkAttachmentDefinition watching queue size
	MAX_QSIZE = 100
	// referred by daemon watcher
	DAEMON_LABEL_NAME  = "app"
	DAEMON_LABEL_VALUE = "multi-nicd"
)

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

//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=configs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=configs/status,verbs=get;update;patch

const ReconcileTime = 30 * time.Minute

func (r *ConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("config", req.NamespacedName)

	instance := &netcogadvisoriov1.Config{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// - if Config is deleted, delete daemon
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.callFinalizer(r.Log, req.NamespacedName.Name)
			return ctrl.Result{}, nil
		}
		r.Log.Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: ReconcileTime}, nil
	}

	dsName := instance.GetName()
	_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Get(context.TODO(), dsName, metav1.GetOptions{})
	daemonset := r.newCNIDaemonSet(r.Clientset, dsName, instance.Spec.Daemon)
	if err != nil {
		// - create CNI daemonset, and NetworkAttachmentDefinition watcher if not exist
		if errors.IsNotFound(err) {
			r.newNetAttachDefWatcher(instance)
			_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Create(context.TODO(), daemonset, metav1.CreateOptions{})
			r.Log.Info(fmt.Sprintf("Create new multi-nic daemonset #%s (cni=%s,ipam=%s), err=%v ", dsName, instance.Spec.CNIType, instance.Spec.IPAMType, err))
		} else {
			r.Log.Info(fmt.Sprintf("Cannot get daemonset #%v ", err))
		}
	} else {
		// - otherwise, update the existing daemonset and restart NetworkAttachmentDefinition watcher
		r.newNetAttachDefWatcher(instance)
		_, err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Update(context.TODO(), daemonset, metav1.UpdateOptions{})
		r.Log.Info(fmt.Sprintf("Update multi-nic daemonset #%s, err=%v ", dsName, err))
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netcogadvisoriov1.Config{}).
		Complete(r)
}

// newNetAttachDefWatcher restarts NetworkAttachmentDefinition watcher
func (r *ConfigReconciler) newNetAttachDefWatcher(instance *netcogadvisoriov1.Config) {
	r.NetAttachDefHandler.DaemonPort = instance.Spec.Daemon.DaemonPort
	r.NetAttachDefHandler.TargetCNI = instance.Spec.CNIType
	SetDaemon(instance.Spec)
}

// newCNIDaemonSet creates new CNI daemonset
func (r *ConfigReconciler) newCNIDaemonSet(client *kubernetes.Clientset, name string, daemonSpec netcogadvisoriov1.DaemonSpec) *appsv1.DaemonSet {
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
	secret := corev1.LocalObjectReference{
		Name: daemonSpec.ImagePullSecret,
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
					Containers: []corev1.Container{
						container,
					},
					Volumes: volumes,
					ImagePullSecrets: []corev1.LocalObjectReference{
						secret,
					},
				},
			},
		},
	}
}

// callFinalizer deletes all CIDRs, waits for all ippools deleted, deletes CNI deamonset, and stops NetworkAttachmentDefinition watcher
func (r *ConfigReconciler) callFinalizer(reqLogger logr.Logger, dsName string) error {
	reqLogger.Info(fmt.Sprintf("Finalize %s", dsName))

	// delete all CIDRs
	cidrMap, err := r.CIDRHandler.ListCIDR()
	if err != nil {
		for _, cidr := range cidrMap {
			r.CIDRHandler.DeleteCIDR(cidr)
		}
	}
	// wait for all ippools deleted
	for {
		poolMap, err := r.CIDRHandler.IPPoolHandler.ListIPPool()
		if err != nil || len(poolMap) == 0 {
			break
		}
		reqLogger.Info(fmt.Sprintf("%d ippools left, wait...", len(poolMap)))
		time.Sleep(1 * time.Second)
	}
	// delete CNI deamonset
	err = r.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Delete(context.TODO(), dsName, metav1.DeleteOptions{})
	return nil
}
