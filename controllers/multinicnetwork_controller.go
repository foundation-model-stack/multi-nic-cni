/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

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
	"github.com/foundation-model-stack/multi-nic-cni/plugin"
)

const (
	MULTI_NIC_IPAM_TYPE        = "multi-nic-ipam"
	DEFAULT_MULTI_NIC_CNI_TYPE = "multi-nic"
	DEFAULT_DAEMON_PORT        = 11000
)

type PluginInterface plugin.Plugin

// MultiNicNetworkReconciler reconciles a MultiNicNetwork object
type MultiNicNetworkReconciler struct {
	client.Client
	*plugin.NetAttachDefHandler
	*CIDRHandler
	Log       logr.Logger
	Scheme    *runtime.Scheme
	PluginMap map[string]*PluginInterface
}

func GetPluginMap(config *rest.Config, logger logr.Logger) map[string]*PluginInterface {
	pluginMap := make(map[string]*PluginInterface)
	pluginMap[plugin.IPVLAN_TYPE] = new(PluginInterface)
	*pluginMap[plugin.IPVLAN_TYPE] = &plugin.IPVLANPlugin{
		Log: logger,
	}
	pluginMap[plugin.MACVLAN_TYPE] = new(PluginInterface)
	*pluginMap[plugin.MACVLAN_TYPE] = &plugin.MACVLANPlugin{
		Log: logger,
	}
	pluginMap[plugin.SRIOV_TYPE] = new(PluginInterface)
	sriovPlugin := &plugin.SriovPlugin{
		Log: logger,
	}
	sriovPlugin.Init(config)
	*pluginMap[plugin.SRIOV_TYPE] = sriovPlugin
	pluginMap[plugin.AWS_VPC_CNI] = new(PluginInterface)
	awsVpcCNIPlugin := &plugin.AwsVpcCNIPlugin{
		Log: logger,
	}
	*pluginMap[plugin.AWS_VPC_CNI] = awsVpcCNIPlugin
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
	return simpleIPAM.Type == MULTI_NIC_IPAM_TYPE, nil
}

func (r *MultiNicNetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	instance := &multinicv1.MultiNicNetwork{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			r.Log.Info(fmt.Sprintf("Network %s deleted ", instance.GetName()))
			return ctrl.Result{}, nil
		}
		r.Log.Info(fmt.Sprintf("Cannot get #%v ", err))
		// Error reading the object - requeue the request.
		// ReconcileTime is defined in config_controller
		return ctrl.Result{RequeueAfter: ReconcileTime}, nil
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
		if controllerutil.ContainsFinalizer(instance, multinicnetworkFinalizer) {
			if err := r.CallFinalizer(r.Log, instance); err != nil {
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

	// Get main plugin
	mainPlugin, annotations, err := r.GetMainPluginConf(instance)

	if err != nil {
		r.Log.Info(fmt.Sprintf("Failed to get main config %s: %v", instance.GetName(), err))
	} else {
		mainPlugin = plugin.RemoveEmpty(instance.Spec.MainPlugin.CNIArgs, mainPlugin)
		r.Log.Info(fmt.Sprintf("main plugin: %s", mainPlugin))
		// Create net attach def
		err = r.NetAttachDefHandler.CreateOrUpdate(instance, mainPlugin, annotations)
		if err != nil {
			r.Log.Info(fmt.Sprintf("Failed to create %s: %v", instance.GetName(), err))
			return ctrl.Result{RequeueAfter: ReconcileTime}, nil
		}
		// Handle multi-nic IPAM
		err = r.HandleMultiNicIPAM(instance)
		if err != nil {
			r.Log.Info(fmt.Sprintf("cannot handle MultiNICIPAM: %v", err))
		}
	}
	routeStatus := instance.Status.Status
	if routeStatus == multinicv1.RouteUnknown {
		// some route is failed
		if cidr, ok := CIDRCache[instance.Name]; ok {
			routeStatus = r.CIDRHandler.SyncCIDRRoute(cidr, false)
			err := r.CIDRHandler.MultiNicNetworkHandler.SyncStatus(instance.Name, cidr, routeStatus)
			if err != nil {
				r.Log.Info(fmt.Sprintf("failed to update route status of %s: %v", instance.Name, err))
			}
			if routeStatus == multinicv1.RouteUnknown {
				return ctrl.Result{RequeueAfter: ReconcileTime}, nil
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *MultiNicNetworkReconciler) GetMainPluginConf(instance *multinicv1.MultiNicNetwork) (string, map[string]string, error) {
	spec := instance.Spec.MainPlugin
	if p, exist := r.PluginMap[spec.Type]; exist {
		return (*p).GetConfig(*instance, HostInterfaceCache)
	}
	return "", map[string]string{}, fmt.Errorf("cannot find plugin %s", spec.Type)
}

// SetupWithManager sets up the controller with the Manager.
func (r *MultiNicNetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multinicv1.MultiNicNetwork{}).
		Complete(r)
}

// getIPAMConfig get IPAM config from network definition
func (r *MultiNicNetworkReconciler) getIPAMConfig(instance *multinicv1.MultiNicNetwork) (*multinicv1.PluginConfig, error) {
	isMultiNicIPAM, err := IsMultiNICIPAM(instance)
	ipamConfig := &multinicv1.PluginConfig{}
	if err != nil {
		return nil, err
	}
	if isMultiNicIPAM {
		name := instance.GetName()
		json.Unmarshal([]byte(instance.Spec.IPAM), ipamConfig)
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
	ipamConfig, err := r.getIPAMConfig(instance)
	if err == nil {
		cidrName := instance.GetName()
		_, err := r.CIDRHandler.GetCIDR(cidrName)
		// create new cidr if not created yet. otherwise, let cidr controller update
		if err == nil {
			r.Log.Info(fmt.Sprintf("CIDR %s already exists", cidrName))
		} else {
			_, err := r.CIDRHandler.NewCIDRWithNewConfig(*ipamConfig, instance.GetNamespace())
			if err != nil {
				r.Log.Info(fmt.Sprintf("Cannot init CIDR %s: %v", cidrName, err))
			}
			return err
		}
	}
	return err
}

// deprecated
// isExistConfig checks if considering plugin config do not change from the config in the existing CIDR
func (r *MultiNicNetworkReconciler) isExistConfig(instance *multinicv1.MultiNicNetwork, ipamConfig multinicv1.PluginConfig) bool {
	cidrName := instance.GetName()
	cidr, err := r.CIDRHandler.GetCIDR(cidrName)
	if err == nil {
		return reflect.DeepEqual(cidr.Spec.Config, ipamConfig)
	}
	return false
}

// CallFinalizer deletes NetworkAttachmentDefinition, CIDR and its dependencies
func (r *MultiNicNetworkReconciler) CallFinalizer(reqLogger logr.Logger, instance *multinicv1.MultiNicNetwork) error {
	isMultiNicIPAM, err := IsMultiNICIPAM(instance)
	if err == nil && isMultiNicIPAM {
		cidrName := instance.GetName()
		cidrInstance, _ := r.CIDRHandler.GetCIDR(cidrName)
		// Delete CIDR and its dependencies
		r.CIDRHandler.DeleteCIDR(*cidrInstance)
	}
	// CleanUp plugin resources
	spec := instance.Spec.MainPlugin
	if p, exist := r.PluginMap[spec.Type]; exist {
		err = (*p).CleanUp(*instance)
		reqLogger.Info(fmt.Sprintf("Clean up error: %v", err))
	}
	// Delete NetworkAttachmentDefinition
	err = r.NetAttachDefHandler.DeleteNets(instance)
	reqLogger.Info(fmt.Sprintf("Finalized %s: %v", instance.ObjectMeta.Name, err))
	return nil
}
