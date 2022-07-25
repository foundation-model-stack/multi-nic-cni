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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	netcogadvisoriov1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
)

// CIDRReconciler reconciles a CIDR object
// - if CIDR is deleted, delete CIDR dependency (ippools, routes)
// - otherwise, update CIDR
type CIDRReconciler struct {
	client.Client
	*CIDRHandler
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=cidrs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=cidrs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.cogadvisor.io,resources=cidrs/finalizers,verbs=update

const CIDRReconcileTime = time.Minute
const cidrFinalizer = "finalizers.cidr.net.cogadvisor.io"

var CIDRCache map[string]netcogadvisoriov1.CIDRSpec = make(map[string]netcogadvisoriov1.CIDRSpec)

func (r *CIDRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("cidr", req.NamespacedName)
	cidrName := req.Name
	instance := &netcogadvisoriov1.CIDR{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.Log.Info(fmt.Sprintf("CIDR %s deleted ", cidrName))
			delete(CIDRCache, cidrName)
			return ctrl.Result{}, nil
		}
		r.Log.Info(fmt.Sprintf("Requeue CIDR %s: %v", cidrName, err))
		// Error reading the object - requeue the request.
		// ReconcileTime is defined in config_controller
		return ctrl.Result{RequeueAfter: CIDRReconcileTime}, nil
	}

	// Add finalizer to instance
	if !controllerutil.ContainsFinalizer(instance, cidrFinalizer) {
		controllerutil.AddFinalizer(instance, cidrFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// If CIDR is deleted, delete CIDR dependency (ippools, routes)
	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		if controllerutil.ContainsFinalizer(instance, cidrFinalizer) {
			if err := r.callFinalizer(r.Log, instance); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(instance, cidrFinalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// sync route from CIDR
	routeStatus := r.CIDRHandler.SyncCIDRRoute(instance.Spec, true)
	r.CIDRHandler.MultiNicNetworkHandler.UpdateStatus(*instance, routeStatus)
	if routeStatus == netcogadvisoriov1.RouteUnknown || routeStatus == netcogadvisoriov1.SomeRouteFailed {
		// if some routes are not properly updated, retry
		r.Log.Info(fmt.Sprintf("Requeue CIDR %s, some routes cannot be updated.", cidrName))
		return ctrl.Result{RequeueAfter: CIDRReconcileTime}, nil
	} else {
		//success
		r.Log.Info(fmt.Sprintf("CIDR %s successfully applied", cidrName))
		CIDRCache[cidrName] = *instance.Spec.DeepCopy()
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CIDRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netcogadvisoriov1.CIDR{}).
		Complete(r)
}

// callFinalizer deletes CIDR and its dependencies
func (r *CIDRReconciler) callFinalizer(reqLogger logr.Logger, instance *netcogadvisoriov1.CIDR) error {
	r.CIDRHandler.DeleteCIDR(*instance)
	reqLogger.Info(fmt.Sprintf("Finalized %s", instance.ObjectMeta.Name))
	return nil
}
