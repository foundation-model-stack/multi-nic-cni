/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/controllers/vars"
	"github.com/foundation-model-stack/multi-nic-cni/plugin"
)

type PluginInterface plugin.Plugin

// MultiNicNetworkReconciler reconciles a MultiNicNetwork object
type MultiNicNetworkReconciler struct {
	client.Client
	*plugin.NetAttachDefHandler
	*CIDRHandler
	Scheme    *runtime.Scheme
	PluginMap map[string]*PluginInterface
}

func GetPluginMap(config *rest.Config) map[string]*PluginInterface {
	pluginMap := make(map[string]*PluginInterface)
	pluginMap[plugin.IPVLAN_TYPE] = new(PluginInterface)
	*pluginMap[plugin.IPVLAN_TYPE] = &plugin.IPVLANPlugin{}
	pluginMap[plugin.MACVLAN_TYPE] = new(PluginInterface)
	*pluginMap[plugin.MACVLAN_TYPE] = &plugin.MACVLANPlugin{}
	pluginMap[plugin.SRIOV_TYPE] = new(PluginInterface)
	sriovPlugin := &plugin.SriovPlugin{}
	err := sriovPlugin.Init(config)
	if err != nil {
		vars.NetworkLog.V(2).Info("Failed to init SR-IoV Plugin: %v", err)
	}
	*pluginMap[plugin.SRIOV_TYPE] = sriovPlugin
	pluginMap[plugin.AWS_IPVLAN_TYPE] = new(PluginInterface)
	awsVpcCNIPlugin := &plugin.AwsVpcCNIPlugin{}
	*pluginMap[plugin.AWS_IPVLAN_TYPE] = awsVpcCNIPlugin
	mellanoxPlugin := &plugin.MellanoxPlugin{}
	err = mellanoxPlugin.Init(config)
	if err != nil {
		vars.NetworkLog.V(2).Info("Failed to init Mellanox Plugin: %v", err)
	}
	pluginMap[plugin.MELLANOX_TYPE] = new(PluginInterface)
	*pluginMap[plugin.MELLANOX_TYPE] = mellanoxPlugin
	return pluginMap
}

//+kubebuilder:rbac:groups=multinic.fms.io,resources=multinicnetworks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multinic.fms.io,resources=multinicnetworks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multinic.fms.io,resources=multinicnetworks/finalizers,verbs=update

const multinicnetworkFinalizer = "finalizers.multinicnetwork.multinic.fms.io"

func IsMultiNICIPAM(instance *multinicv1.MultiNicNetwork) (bool, error) {
	simpleIPAM := &types.IPAM{}
	err := json.Unmarshal([]byte(instance.Spec.IPAM), simpleIPAM)
	if err != nil {
		return false, fmt.Errorf("%s: %v", instance.Spec.IPAM, err)
	}
	return simpleIPAM.Type == vars.MultiNICIPAMType, nil
}

func (r *MultiNicNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !ConfigReady {
		return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
	}
	_ = log.FromContext(ctx)

	instance := &multinicv1.MultiNicNetwork{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		vars.NetworkLog.V(7).Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		// ReconcileTime is defined in config_controller
		return ctrl.Result{RequeueAfter: vars.LongReconcileTime}, nil
	}

	// Add finalizer to instance
	if !controllerutil.ContainsFinalizer(instance, multinicnetworkFinalizer) {
		controllerutil.AddFinalizer(instance, multinicnetworkFinalizer)
		err = r.Update(ctx, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	is_deleted := instance.GetDeletionTimestamp() != nil
	if is_deleted {
		vars.NetworkLog.V(2).Info(fmt.Sprintf("Network %s deletion set: %v ", instance.GetName(), instance.GetDeletionTimestamp()))
		if controllerutil.ContainsFinalizer(instance, multinicnetworkFinalizer) {
			if err := r.callFinalizer(vars.NetworkLog, instance); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(instance, multinicnetworkFinalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// setNetAddress if not defined
	if len(instance.Spec.MasterNetAddrs) == 0 {
		instance.Spec.MasterNetAddrs = r.CIDRHandler.GetAllNetAddrs()
	}
	// set latest discovery information
	daemonSize := r.CIDRHandler.DaemonCacheHandler.SafeCache.GetSize()
	infoAvailableSize := r.CIDRHandler.HostInterfaceHandler.GetInfoAvailableSize()
	instance.Status.DiscoverStatus.ExistDaemon = daemonSize
	instance.Status.InterfaceInfoAvailable = infoAvailableSize

	// Get main plugin
	mainPlugin, annotations, err := r.GetMainPluginConf(instance)
	multinicnetworkName := instance.GetName()
	if err != nil {
		message := fmt.Sprintf("Failed to get main config %s: %v", multinicnetworkName, err)
		err = r.CIDRHandler.MultiNicNetworkHandler.UpdateNetConfigStatus(instance, multinicv1.ConfigFailed, message)
		if err != nil {
			message = fmt.Sprintf("%s and Failed to UpdateNetConfigStatus: %v", message, err)
		}
		vars.NetworkLog.V(2).Info(message)
	} else {
		mainPlugin = plugin.RemoveEmpty(instance.Spec.MainPlugin.CNIArgs, mainPlugin)
		vars.NetworkLog.V(2).Info(fmt.Sprintf("main plugin: %s", mainPlugin))
		// Create net attach def
		err = r.NetAttachDefHandler.CreateOrUpdate(instance, mainPlugin, annotations)
		if err != nil {
			message := fmt.Sprintf("Failed to create %s: %v", multinicnetworkName, err)
			err = r.CIDRHandler.MultiNicNetworkHandler.UpdateNetConfigStatus(instance, multinicv1.ConfigFailed, message)
			if err != nil {
				message = fmt.Sprintf("%s and Failed to UpdateNetConfigStatus: %v", message, err)
			}
			vars.NetworkLog.V(2).Info(message)
			// Reconcile if fail to update or create some of net-attach-def
			return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
		}
		// Handle multi-nic IPAM
		err = r.HandleMultiNicIPAM(instance)
		if err != nil {
			message := fmt.Sprintf("Failed to manage %s: %v", multinicnetworkName, err)
			vars.NetworkLog.V(2).Info(message)
			return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
		}
	}
	routeStatus := instance.Status.RouteStatus
	if routeStatus == multinicv1.RouteUnknown || (instance.Spec.IsMultiNICIPAM && routeStatus == multinicv1.RouteNoApplied) {
		// some route is failed or route not applied yet
		cidr, err := r.CIDRHandler.GetCache(multinicnetworkName)
		if err == nil {
			routeStatus = r.CIDRHandler.SyncCIDRRoute(cidr, false)
			netStatus, err := r.CIDRHandler.MultiNicNetworkHandler.SyncAllStatus(multinicnetworkName, cidr, routeStatus, daemonSize, infoAvailableSize, false)
			if err != nil {
				vars.NetworkLog.V(2).Info(fmt.Sprintf("Failed to update route status of %s: %v", multinicnetworkName, err))
			} else if netStatus.CIDRProcessedHost != netStatus.InterfaceInfoAvailable {
				r.CIDRHandler.UpdateCIDRs()
			}
			if routeStatus == multinicv1.RouteUnknown {
				return ctrl.Result{RequeueAfter: vars.NormalReconcileTime}, nil
			} else if routeStatus == multinicv1.AllRouteApplied {
				//success
				vars.NetworkLog.V(3).Info(fmt.Sprintf("CIDR %s successfully applied", multinicnetworkName))
			}
		}
	} else if !instance.Spec.IsMultiNICIPAM && routeStatus == multinicv1.RouteNoApplied {
		// not related to L3
		err = r.CIDRHandler.MultiNicNetworkHandler.UpdateNetConfigStatus(instance, multinicv1.ConfigComplete, "")
		if err != nil {
			vars.NetworkLog.V(3).Info(fmt.Sprintf("Failed to UpdateNetConfigStatus %s for non-L3: %v", instance.Name, err))
		}
	} else if routeStatus != multinicv1.AllRouteApplied {
		// some route still fails
		err = r.CIDRHandler.MultiNicNetworkHandler.UpdateNetConfigStatus(instance, multinicv1.WaitForConfig, "")
		if err != nil {
			vars.NetworkLog.V(3).Info(fmt.Sprintf("Failed to UpdateNetConfigStatus %s at route failure: %v", instance.Name, err))
		}
	}
	return ctrl.Result{}, nil
}

func (r *MultiNicNetworkReconciler) GetMainPluginConf(instance *multinicv1.MultiNicNetwork) (string, map[string]string, error) {
	spec := instance.Spec.MainPlugin
	if p, exist := r.PluginMap[spec.Type]; exist {
		return (*p).GetConfig(*instance, r.CIDRHandler.HostInterfaceHandler.ListCache())
	}
	return "", map[string]string{}, fmt.Errorf("cannot find plugin %s", spec.Type)
}

// SetupWithManager sets up the controller with the Manager.
func (r *MultiNicNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.MultiNicNetwork{}).
		Complete(r)
}

// GetIPAMConfig get IPAM config from network definition
func (r *MultiNicNetworkReconciler) GetIPAMConfig(instance *multinicv1.MultiNicNetwork) (*multinicv1.PluginConfig, error) {
	isMultiNicIPAM, err := IsMultiNICIPAM(instance)
	ipamConfig := &multinicv1.PluginConfig{}
	if err != nil {
		return nil, err
	}
	if isMultiNicIPAM {
		name := instance.GetName()
		err := json.Unmarshal([]byte(instance.Spec.IPAM), ipamConfig)
		if err != nil {
			return nil, err
		}
		ipamConfig.Name = name
		ipamConfig.Type = instance.Spec.MainPlugin.Type
		ipamConfig.Subnet = instance.Spec.Subnet
		ipamConfig.MasterNetAddrs = instance.Spec.MasterNetAddrs
		return ipamConfig, nil
	}
	return nil, fmt.Errorf("non-MultiNicIPAM")
}

// HandleMultiNicIPAM handles ipam if target type
func (r *MultiNicNetworkReconciler) HandleMultiNicIPAM(instance *multinicv1.MultiNicNetwork) error {
	ipamConfig, err := r.GetIPAMConfig(instance)
	if err == nil {
		cidrName := instance.GetName()
		_, err := r.CIDRHandler.GetCIDR(cidrName)
		// create new cidr if not created yet. otherwise, let cidr controller update
		if err == nil {
			vars.NetworkLog.V(3).Info(fmt.Sprintf("CIDR %s already exists", cidrName))
		} else {
			if errors.IsNotFound(err) {
				_, err = r.CIDRHandler.NewCIDRWithNewConfig(*ipamConfig, instance.GetNamespace())
			}
			if err != nil {
				vars.NetworkLog.V(3).Info(fmt.Sprintf("Failed to HandleMultiNicIPAM for %s: %v", cidrName, err))
			}
			return err
		}
	}
	return err
}

// callFinalizer deletes NetworkAttachmentDefinition, CIDR and its dependencies
func (r *MultiNicNetworkReconciler) callFinalizer(reqLogger logr.Logger, instance *multinicv1.MultiNicNetwork) error {
	isMultiNicIPAM, err := IsMultiNICIPAM(instance)
	if err == nil && isMultiNicIPAM {
		cidrName := instance.GetName()
		cidrInstance, _ := r.CIDRHandler.GetCIDR(cidrName)
		// Delete CIDR and its dependencies
		err = r.CIDRHandler.DeleteCIDR(*cidrInstance)
		if err != nil {
			reqLogger.V(2).Info(fmt.Sprintf("Failed to delete CIDR %s: %v", cidrName, err))
		}
	}
	// CleanUp plugin resources
	spec := instance.Spec.MainPlugin
	if p, exist := r.PluginMap[spec.Type]; exist {
		err = (*p).CleanUp(*instance)
		reqLogger.V(2).Info(fmt.Sprintf("Clean up error: %v", err))
	}
	// Delete NetworkAttachmentDefinition
	err = r.NetAttachDefHandler.DeleteNets(instance)
	reqLogger.V(2).Info(fmt.Sprintf("Finalized %s: %v", instance.ObjectMeta.Name, err))
	r.CIDRHandler.MultiNicNetworkHandler.SafeCache.UnsetCache(instance.Name)
	return nil
}
