/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

// IPPoolReconciler reconciles a IPPool object
// - if IPPool is deleted, delete corresponding routes
type IPPoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*CIDRHandler
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=ippools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=ippools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multinic.fms.io,resources=ippools/finalizers,verbs=update

const ippoolFinalizer = "finalizers.ippool.multinic.fms.io"

func InitIppoolCache(ippoolHandler *IPPoolHandler) error {
	listObjects, err := ippoolHandler.ListIPPool()
	if err == nil {
		for name, instance := range listObjects {
			ippoolHandler.SetCache(name, instance.Spec)
			// if any label is not set, set the label
			if _, found := instance.ObjectMeta.Labels[vars.HostNameLabel]; !found {
				err = ippoolHandler.AddLabel(&instance)
			} else if _, found := instance.ObjectMeta.Labels[vars.DefNameLabel]; !found {
				err = ippoolHandler.AddLabel(&instance)
			}
		}
	}
	return err
}

func (r *IPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !ConfigReady {
		return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
	}

	instance := &multinicv1.IPPool{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		vars.IPPoolLog.V(7).Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		// ReconcileTime is defined in config_controller
		return ctrl.Result{RequeueAfter: vars.LongReconcileTime}, nil
	}

	// If IPPool is deleted, delete corresponding routes
	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		if controllerutil.ContainsFinalizer(instance, ippoolFinalizer) {
			if err := r.callFinalizer(vars.IPPoolLog, instance); err != nil {
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
				controllerutil.RemoveFinalizer(instance, ippoolFinalizer)
				return r.Client.Update(ctx, instance)
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	} else {
		ippoolName := instance.GetName()
		r.CIDRHandler.IPPoolHandler.SetCache(ippoolName, instance.Spec)
	}

	// Add finalizer to instance
	if !controllerutil.ContainsFinalizer(instance, ippoolFinalizer) {
		controllerutil.AddFinalizer(instance, ippoolFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.IPPool{}).
		Complete(r)
}

// callFinalizer reports remaining allocated IPs, and delete corrsponding routes
func (r *IPPoolReconciler) callFinalizer(reqLogger logr.Logger, instance *multinicv1.IPPool) error {
	// report remaining allocated IPs
	allocations := instance.Spec.Allocations
	if len(allocations) > 0 {
		var remainPods []string
		for _, allocation := range allocations {
			remainPods = append(remainPods, fmt.Sprintf("%s/%s", allocation.Namespace, allocation.Pod))
		}
		reqLogger.V(5).Info(fmt.Sprintf("IPPool %s remains %v allocated", instance.GetName(), remainPods))
	}
	reqLogger.V(5).Info(fmt.Sprintf("Finalized %s", instance.ObjectMeta.Name))
	r.CIDRHandler.IPPoolHandler.SafeCache.UnsetCache(instance.ObjectMeta.Name)
	return nil
}
