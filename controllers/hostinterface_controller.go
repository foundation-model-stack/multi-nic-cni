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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
)

// HostInterfaceReconciler reconciles a HostInterface object
// - if HostInterface is deleted, re-process daemon pods
type HostInterfaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*DaemonWatcher
}

const HostInterfaceReconcileTime = time.Minute

//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces/finalizers,verbs=update

func (r *HostInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("hostinterface", req.NamespacedName)
	instance := &netcogadvisoriov1.HostInterface{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil || instance.GetDeletionTimestamp() != nil {
		// deleted, re-process daemon pods (recreate HostInterface if the daemon pod still exists)
		r.DaemonWatcher.UpdateCurrentList()
		r.DaemonWatcher.UpdateCIDRs()
		return ctrl.Result{}, nil
	}
	err = r.updateInterfaces(instance)
	if err != nil {
		// deamon pod may be missing for a short time
		r.Log.Info(fmt.Sprintf("cannot update interfaces: %v", err))
		return ctrl.Result{RequeueAfter: HostInterfaceReconcileTime}, nil
	}
	return ctrl.Result{RequeueAfter: HostInterfaceReconcileTime}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HostInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netcogadvisoriov1.HostInterface{}).
		Complete(r)
}

func (r *HostInterfaceReconciler) updateInterfaces(instance *netcogadvisoriov1.HostInterface) error {
	nodeName := instance.Spec.HostName
	labels := fmt.Sprintf("%s=%s", HOST_INTERFACE_LABEL, nodeName)
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	podList, _ := r.DaemonWatcher.Clientset.CoreV1().Pods(DAEMON_NAMESPACE).List(context.TODO(), listOptions)
	for _, pod := range podList.Items {
		interfaces, err := r.DaemonWatcher.DaemonConnector.GetInterfaces(pod)
		if err != nil {
			return err
		}
		if r.interfaceChanged(instance.Spec.Interfaces, interfaces) {
			r.DaemonWatcher.IpamJoin(pod)
			err = r.DaemonWatcher.HostInterfaceHandler.UpdateHostInterface(*instance, interfaces)
			if err != nil {
				return err
			}
			r.Log.Info(fmt.Sprintf("%s's interfaces updated", nodeName))
			r.DaemonWatcher.UpdateCIDRs()
			return nil
		} else {
			return nil
		}
	}
	return fmt.Errorf("no daemon pod found for %s", nodeName)
}

func (r *HostInterfaceReconciler) interfaceChanged(olds []netcogadvisoriov1.InterfaceInfoType, news []netcogadvisoriov1.InterfaceInfoType) bool {
	if len(olds) != len(news) {
		return true
	}
	oldMap := make(map[string]netcogadvisoriov1.InterfaceInfoType)
	for _, old := range olds {
		oldMap[old.InterfaceName] = old
	}
	for _, new := range news {
		if old, exists := oldMap[new.InterfaceName]; exists {
			return !old.Equal(new)
		} else {
			return true
		}
	}
	return true
}
