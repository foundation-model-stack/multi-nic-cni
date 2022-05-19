/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
)

// IPPoolReconciler reconciles a IPPool object
// - if IPPool is deleted, delete corresponding routes
type IPPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*CIDRHandler
}

//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=ippools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=ippools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=ippools/finalizers,verbs=update

const ippoolFinalizer = "finalizers.cidr.net.cogadvisor.io"

func (r *IPPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("ippool", req.NamespacedName)

	instance := &netcogadvisoriov1.IPPool{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.Log.Info(fmt.Sprintf("IPPool %s deleted", instance.GetName()))
			return ctrl.Result{}, nil
		}
		r.Log.Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		// ReconcileTime is defined in config_controller
		return ctrl.Result{RequeueAfter: ReconcileTime}, nil
	}

	// If IPPool is deleted, delete corresponding routes
	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		if controllerutil.ContainsFinalizer(instance, ippoolFinalizer) {
			if err := r.callFinalizer(r.Log, instance); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(instance, ippoolFinalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
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
		For(&netcogadvisoriov1.IPPool{}).
		Complete(r)
}

// callFinalizer reports remaining allocated IPs, and delete corrsponding routes
func (r *IPPoolReconciler) callFinalizer(reqLogger logr.Logger, instance *netcogadvisoriov1.IPPool) error {
	// report remaining allocated IPs
	allocations := instance.Spec.Allocations
	if len(allocations) > 0 {
		var remainPods []string
		for _, allocation := range allocations {
			remainPods = append(remainPods, fmt.Sprintf("%s/%s", allocation.Namespace, allocation.Pod))
		}
		reqLogger.Info(fmt.Sprintf("IPPool %s remains %v allocated", instance.GetName(), remainPods))
	}
	reqLogger.Info(fmt.Sprintf("Finalized %s", instance.ObjectMeta.Name))
	return nil
}
