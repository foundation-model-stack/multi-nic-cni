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

	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"
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
const TestModelLabel = "test-mode"

var HostInterfaceCache map[string]netcogadvisoriov1.HostInterface = make(map[string]netcogadvisoriov1.HostInterface)

//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=hostinterfaces/finalizers,verbs=update

func (r *HostInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("hostinterface", req.NamespacedName)
	instance := &netcogadvisoriov1.HostInterface{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// deleted, re-process daemon pods (recreate HostInterface if the daemon pod still exists)
			r.DaemonWatcher.UpdateCurrentList()
			r.DaemonWatcher.UpdateCIDRs()
			delete(HostInterfaceCache, req.Name)
			return ctrl.Result{}, nil
		}
		// error by other reasons, requeue
		r.Log.Info(fmt.Sprintf("Requeue HostInterface %s: %v", req.Name, err))
		return ctrl.Result{RequeueAfter: HostInterfaceReconcileTime}, nil
	}
	err = r.UpdateInterfaces(*instance)
	if err != nil {
		// deamon pod may be missing for a short time
		r.Log.Info(fmt.Sprintf("Requeue HostInterface %s, cannot update interfaces: %v", instance.ObjectMeta.Name, err))
		return ctrl.Result{RequeueAfter: HostInterfaceReconcileTime}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HostInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netcogadvisoriov1.HostInterface{}).
		Complete(r)
}

func (r *HostInterfaceReconciler) UpdateInterfaces(instance netcogadvisoriov1.HostInterface) error {
	nodeName := instance.Spec.HostName
	hifName := instance.GetName()

	if pod, ok := DaemonCache[nodeName]; ok {
		podAddress := GetDaemonAddressByPod(pod)
		interfaces, err := r.DaemonWatcher.DaemonConnector.GetInterfaces(podAddress)
		if err != nil {
			return err
		}
		_, found := HostInterfaceCache[hifName]
		if !found {
			HostInterfaceCache[hifName] = *instance.DeepCopy()
			r.DaemonWatcher.UpdateCIDRs()
		} else if r.interfaceChanged(instance.Spec.Interfaces, interfaces) {
			r.DaemonWatcher.IpamJoin(pod)
			updatedHif, err := r.DaemonWatcher.HostInterfaceHandler.UpdateHostInterface(instance, interfaces)
			if err != nil {
				return err
			}
			r.Log.Info(fmt.Sprintf("%s's interfaces updated", nodeName))
			HostInterfaceCache[hifName] = *updatedHif.DeepCopy()
			r.DaemonWatcher.UpdateCIDRs()
		}
		return nil
	}
	if _, ok := instance.Labels[TestModelLabel]; ok {
		r.Log.Info(fmt.Sprintf("%s on test mode", nodeName))
		return nil
	}
	_, err := r.DaemonWatcher.Clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		// not found node
		r.HostInterfaceHandler.DeleteHostInterface(nodeName)
		r.Log.Info(fmt.Sprintf("Delete Hostinterface %s: node no more exists", nodeName))
		return nil
	} else {
		return fmt.Errorf("no daemon pod found for %s", nodeName)
	}
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
