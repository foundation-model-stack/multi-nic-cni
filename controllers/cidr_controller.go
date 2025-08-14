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

// CIDRReconciler reconciles a CIDR object
// - if CIDR is deleted, delete CIDR dependency (ippools, routes)
// - otherwise, update CIDR
type CIDRReconciler struct {
	client.Client
	*CIDRHandler
	*DaemonWatcher
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=cidrs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=cidrs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multinic.fms.io,resources=cidrs/finalizers,verbs=update

const cidrFinalizer = "finalizers.cidr.multinic.fms.io"

func (r *CIDRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !ConfigReady {
		return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
	}
	_ = vars.CIDRLog.WithValues("cidr", req.NamespacedName)
	cidrName := req.Name
	instance := &multinicv1.CIDR{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		vars.CIDRLog.V(7).Info(fmt.Sprintf("Requeue CIDR %s: %v", cidrName, err))
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: vars.LongReconcileTime}, nil
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
			if err := r.callFinalizer(vars.CIDRLog, instance); err != nil {
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
				controllerutil.RemoveFinalizer(instance, cidrFinalizer)
				return r.Client.Update(ctx, instance)
			})
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// sync status
	routeStatus := r.CIDRHandler.SyncCIDRRoute(instance.Spec, true)
	daemonSize := r.CIDRHandler.DaemonCacheHandler.SafeCache.GetSize()
	infoAvailableSize := r.CIDRHandler.HostInterfaceHandler.GetInfoAvailableSize()
	netStatus, err := r.CIDRHandler.MultiNicNetworkHandler.SyncAllStatus(cidrName, instance.Spec, routeStatus, daemonSize, infoAvailableSize, true)
	if err != nil {
		vars.CIDRLog.V(2).Info(fmt.Sprintf("Failed to update route status of %s: %v", cidrName, err))
		vars.CIDRLog.V(7).Info(fmt.Sprintf("Requeue CIDR %s: %v", cidrName, err))
		return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
	} else if netStatus.CIDRProcessedHost != netStatus.InterfaceInfoAvailable {
		r.UpdateCIDRs()
	}

	// call greeting
	daemonSnapshot := r.CIDRHandler.DaemonCacheHandler.ListCache()
	for _, daemon := range daemonSnapshot {
		err = r.DaemonWatcher.IpamJoin(daemon)
		if err != nil {
			vars.CIDRLog.V(4).Info(fmt.Sprintf("Failed to join %s: %v", daemon.NodeName, err))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CIDRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.CIDR{}).
		Complete(r)
}

// callFinalizer deletes CIDR and its dependencies
func (r *CIDRReconciler) callFinalizer(reqLogger logr.Logger, instance *multinicv1.CIDR) error {
	name := instance.ObjectMeta.Name
	err := r.CIDRHandler.DeleteCIDR(*instance)
	if err != nil {
		vars.CIDRLog.V(3).Info(fmt.Sprintf("Failed to delete %s", name))
	}
	r.CIDRHandler.SafeCache.UnsetCache(name)
	reqLogger.V(3).Info(fmt.Sprintf("Finalized %s", name))
	return nil
}
