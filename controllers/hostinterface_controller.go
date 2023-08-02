/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"

	"k8s.io/apimachinery/pkg/api/errors"
)

// HostInterfaceReconciler reconciles a HostInterface object
// - if HostInterface is deleted, re-process daemon pods
type HostInterfaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*DaemonWatcher
	*HostInterfaceHandler
	*CIDRHandler
}

const hifFinalizer = "finalizers.hostinterface.multinic.fms.io"

func InitHostInterfaceCache(clientset *kubernetes.Clientset, hostInterfaceHandler *HostInterfaceHandler, daemonCacheHandler *DaemonCacheHandler) error {
	listObjects, err := hostInterfaceHandler.ListHostInterface()
	if err == nil {
		// check existing hostinterface
		for name, instance := range listObjects {
			if _, ok := instance.Labels[vars.TestModeLabel]; ok {
				// on test mode, no need to init
				break
			}
			if _, foundErr := daemonCacheHandler.GetCache(name); foundErr != nil {
				// not found, check whether node is still there.
				if _, foundErr = clientset.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{}); foundErr != nil {
					// delete HostInterface and do not add
					err = hostInterfaceHandler.DeleteHostInterface(name)
					if err != nil {
						vars.HifLog.V(4).Info(fmt.Sprintf("Failed to delete HostInterface %s: %v", name, err))
					}
					continue
				}
			}
			hostInterfaceHandler.SetCache(name, instance)
		}
		// check missing hostinterface
		daemonSnapshot := daemonCacheHandler.ListCache()
		for name, daemon := range daemonSnapshot {
			if _, found := listObjects[name]; !found {
				// create missing hostinterface
				createErr := hostInterfaceHandler.CreateHostInterface(daemon.NodeName, []multinicv1.InterfaceInfoType{})
				if createErr != nil {
					vars.HifLog.V(4).Info(fmt.Sprintf("Failed to initialize HostInterface %s: %v", name, err))
				}
			}
		}
	}
	return err
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multinic.fms.io,resources=hostinterfaces/finalizers,verbs=update

func (r *HostInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = vars.HifLog.WithValues("hostinterface", req.NamespacedName)
	instance := &multinicv1.HostInterface{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// error by other reasons, requeue
		vars.HifLog.V(7).Info(fmt.Sprintf("Requeue HostInterface %s: %v", req.Name, err))
		return ctrl.Result{RequeueAfter: vars.UrgentReconcileTime}, nil
	}

	// Add finalizer to instance
	if !controllerutil.ContainsFinalizer(instance, hifFinalizer) {
		controllerutil.AddFinalizer(instance, hifFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		if controllerutil.ContainsFinalizer(instance, hifFinalizer) {
			if err := r.CallFinalizer(vars.HifLog, instance); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(instance, hifFinalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	hifName := instance.GetName()
	if vars.IsUnmanaged(instance.ObjectMeta) {
		// unmanaged hostinterface
		r.HostInterfaceHandler.SetCache(hifName, *instance.DeepCopy())
		return ctrl.Result{}, nil
	}

	if !r.HostInterfaceHandler.SafeCache.Contains(hifName) && len(instance.Spec.Interfaces) > 0 {
		r.HostInterfaceHandler.SetCache(hifName, *instance.DeepCopy())
	}

	if !ConfigReady || !r.DaemonWatcher.IsDaemonSetReady() {
		// only hostinterface must be reconciled urgently after config is ready because it's tightly coupling with daemon
		return ctrl.Result{RequeueAfter: vars.UrgentReconcileTime}, nil
	}

	vars.HifLog.V(7).Info(fmt.Sprintf("HostInterface reconciled: %s", instance.ObjectMeta.Name))
	err = r.UpdateInterfaces(*instance)
	if err != nil {
		// deamon pod may be missing for a short time
		vars.HifLog.V(4).Info(fmt.Sprintf("Requeue HostInterface %s, cannot update interfaces: %v", instance.ObjectMeta.Name, err))
		return ctrl.Result{RequeueAfter: vars.UrgentReconcileTime}, nil
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
	if vars.IsUnmanaged(instance.ObjectMeta) {
		return nil
	}
	nodeName := instance.Spec.HostName
	hifName := instance.GetName()
	pod, err := r.DaemonWatcher.TryGetDaemonPod(nodeName)
	if err == nil {
		// daemon exists
		if pod.Name == "" {
			// cannot confirm pod status
			vars.HifLog.V(4).Info(fmt.Sprintf("Hostinterface %s: cannot confirm daemon pod status", nodeName))
			return fmt.Errorf(vars.ThrottlingError)
		}
		podAddress := GetDaemonAddressByPod(pod)
		interfaces, err := r.DaemonWatcher.DaemonConnector.GetInterfaces(podAddress)
		if err != nil {
			return err
		}
		if r.interfaceChanged(instance.Spec.Interfaces, interfaces) {
			err = r.DaemonWatcher.IpamJoin(pod)
			if err != nil {
				vars.HifLog.V(4).Info(fmt.Sprintf("Failed to join %s: %v", nodeName, err))
			}

			updatedHif, err := r.HostInterfaceHandler.UpdateHostInterface(instance, interfaces)
			if err != nil {
				return err
			}
			vars.HifLog.V(7).Info(fmt.Sprintf("%s's interfaces updated", nodeName))
			r.HostInterfaceHandler.SetCache(hifName, *updatedHif.DeepCopy())
			r.CIDRHandler.UpdateCIDRs()
		}
		return nil
	}
	if _, ok := instance.Labels[vars.TestModeLabel]; ok {
		vars.HifLog.V(7).Info(fmt.Sprintf("%s on test mode", nodeName))
		return nil
	}
	// daemon pod does not exist
	_, err = r.DaemonWatcher.Clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err == nil {
		// node exists but might be tainted
		vars.HifLog.V(4).Info(fmt.Sprintf("Hostinterface %s: no daemon pod found (node exists)", nodeName))
		return nil
	}
	if errors.IsNotFound(err) {
		// not found node
		r.DaemonCacheHandler.UnsetCache(nodeName)
		err = r.HostInterfaceHandler.DeleteHostInterface(nodeName)
		if err != nil {
			vars.HifLog.V(4).Info(fmt.Sprintf("Failed to delete HostInterface %s: %v", nodeName, err))
		} else {
			vars.HifLog.V(4).Info(fmt.Sprintf("Delete Hostinterface %s: node no more exists", nodeName))
		}
		return nil
	} else {
		// err to get node
		vars.HifLog.V(4).Info(fmt.Sprintf("Hostinterface %s: cannot confirm node status", nodeName))
		return err
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

// CallFinalizer updates CIDRs
func (r *HostInterfaceReconciler) CallFinalizer(reqLogger logr.Logger, instance *multinicv1.HostInterface) error {
	r.HostInterfaceHandler.SafeCache.UnsetCache(instance.Name)
	r.CIDRHandler.UpdateCIDRs()
	reqLogger.V(4).Info(fmt.Sprintf("Finalized %s", instance.Name))
	return nil
}
