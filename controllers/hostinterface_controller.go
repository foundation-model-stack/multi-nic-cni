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
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

// HostInterfaceReconciler reconciles a HostInterface object
// - if HostInterface is deleted, re-process daemon pods
type HostInterfaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*DaemonWatcher
	*HostInterfaceHandler
	*CIDRHandler
}

const HostInterfaceReconcileTime = time.Second

const TestModelLabel = "test-mode"

func InitHostInterfaceCache(clientset *kubernetes.Clientset, hostInterfaceHandler *HostInterfaceHandler, daemonCacheHandler *DaemonCacheHandler) error {
	listObjects, err := hostInterfaceHandler.ListHostInterface()
	if err == nil {
		for name, instance := range listObjects {
			if _, foundErr := daemonCacheHandler.GetCache(name); foundErr != nil {
				// not found, check whether node is still there.
				if _, foundErr = clientset.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{}); foundErr != nil {
					// delete HostInterface and do not add
					hostInterfaceHandler.DeleteHostInterface(name)
					continue
				}
			}
			hostInterfaceHandler.SetCache(name, instance)
		}
	}
	return err
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces/finalizers,verbs=update

func (r *HostInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("hostinterface", req.NamespacedName)
	instance := &multinicv1.HostInterface{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.UpdateCIDRs()
			return ctrl.Result{}, nil
		}
		// error by other reasons, requeue
		r.Log.Info(fmt.Sprintf("Requeue HostInterface %s: %v", req.Name, err))
		return ctrl.Result{RequeueAfter: HostInterfaceReconcileTime}, nil
	}

	if !ConfigReady || !r.DaemonWatcher.IsDaemonSetReady() {
		return ctrl.Result{RequeueAfter: ConfigWaitingReconcileTime}, nil
	}

	r.Log.Info(fmt.Sprintf("HostInterface reconciled: %s", instance.ObjectMeta.Name))
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
		For(&multinicv1.HostInterface{}).
		Complete(r)
}

func (r *HostInterfaceReconciler) UpdateInterfaces(instance multinicv1.HostInterface) error {
	nodeName := instance.Spec.HostName
	hifName := instance.GetName()
	pod, err := r.DaemonWatcher.DaemonCacheHandler.GetCache(nodeName)
	if err == nil {
		podAddress := GetDaemonAddressByPod(pod)
		interfaces, err := r.DaemonWatcher.DaemonConnector.GetInterfaces(podAddress)
		if err != nil {
			return err
		}
		if !r.HostInterfaceHandler.SafeCache.Contains(hifName) {
			r.HostInterfaceHandler.SetCache(hifName, *instance.DeepCopy())
		}
		if r.interfaceChanged(instance.Spec.Interfaces, interfaces) {
			r.DaemonWatcher.IpamJoin(pod)
			updatedHif, err := r.HostInterfaceHandler.UpdateHostInterface(instance, interfaces)
			if err != nil {
				return err
			}
			r.Log.Info(fmt.Sprintf("%s's interfaces updated", nodeName))
			r.HostInterfaceHandler.SetCache(hifName, *updatedHif.DeepCopy())
			r.handleUpdatedHostInterface(true)
		}
		return nil
	}
	if _, ok := instance.Labels[TestModelLabel]; ok {
		r.Log.Info(fmt.Sprintf("%s on test mode", nodeName))
		return nil
	}
	_, err = r.DaemonWatcher.Clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		// not found node
		r.HostInterfaceHandler.DeleteHostInterface(nodeName)
		r.Log.Info(fmt.Sprintf("Delete Hostinterface %s: node no more exists", nodeName))
		return nil
	} else {
		// not found pod even DaemonSet is already ready, node was tainted
		r.HostInterfaceHandler.DeleteHostInterface(nodeName)
		r.Log.Info(fmt.Sprintf("Hostinterface %s: no daemon pod found", nodeName))
		return nil
	}
}

func (r *HostInterfaceReconciler) handleUpdatedHostInterface(added bool) {
	routeChange := r.CIDRHandler.UpdateCIDRs()
	// call greeting if route changed
	daemonSnapshot := r.DaemonWatcher.DaemonCacheHandler.ListCache()
	if added && routeChange {
		for _, daemon := range daemonSnapshot {
			r.DaemonWatcher.IpamJoin(daemon)
		}
	}
}

func (r *HostInterfaceReconciler) interfaceChanged(olds []multinicv1.InterfaceInfoType, news []multinicv1.InterfaceInfoType) bool {
	if len(olds) != len(news) {
		return true
	}
	oldMap := make(map[string]multinicv1.InterfaceInfoType)
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
