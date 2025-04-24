package controllers

import (
	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
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
