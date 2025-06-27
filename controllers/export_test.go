package controllers

import (
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	appsv1 "k8s.io/api/apps/v1"
)

func (r *ConfigReconciler) GetDefaultConfigSpec() multinicv1.ConfigSpec {
	return r.getDefaultConfigSpec()
}

func (r *ConfigReconciler) NewCNIDaemonSet(name string, daemonSpec multinicv1.DaemonSpec) *appsv1.DaemonSet {
	return r.newCNIDaemonSet(r.Clientset, name, daemonSpec)
}

func (r *ConfigReconciler) GetCNIHostPath() string {
	return r.getCNIHostPath()
}

func (h *CIDRHandler) SyncIPPools(validCIDRs map[string]multinicv1.CIDR) map[string][]multinicv1.Allocation {
	return h.syncIPPools(validCIDRs)
}

func (h *IPPoolHandler) SetIPPoolsCache(defName string, entries []multinicv1.CIDREntry, excludes []compute.IPValue) map[string]multinicv1.IPPoolSpec {
	ippools := make(map[string]multinicv1.IPPoolSpec, 0)
	for _, entry := range entries {
		for _, host := range entry.Hosts {
			ippoolName, spec, _ := h.initIPPool(defName, host.PodCIDR, entry.VlanCIDR, host.HostName, host.InterfaceName, excludes)
			h.SetCache(ippoolName, spec)
			ippools[ippoolName] = spec
		}
	}
	return ippools
}

func (h *IPPoolHandler) UnsetIPPoolsCache(defName string, entries []multinicv1.CIDREntry) {
	for _, entry := range entries {
		for _, host := range entry.Hosts {
			ippoolName, _, _ := h.initIPPool(defName, host.PodCIDR, entry.VlanCIDR, host.HostName, host.InterfaceName, []compute.IPValue{})
			h.UnsetCache(ippoolName)
		}
	}
}

func (h *IPPoolHandler) CheckPoolValidity(excludeCIDRs []string, allocations []multinicv1.Allocation) []multinicv1.Allocation {
	return h.checkPoolValidity(excludeCIDRs, allocations)
}

func (h *IPPoolHandler) ExtractMatchExcludesFromPodCIDR(excludes []compute.IPValue, podCIDR string) []string {
	return h.extractMatchExcludesFromPodCIDR(excludes, podCIDR)
}
