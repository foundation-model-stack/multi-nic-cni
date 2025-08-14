/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache-2.0
 */

package controllers

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

// NamespaceWatcher watches new namespace and generate net-attach-def
type NamespaceWatcher struct {
	*kubernetes.Clientset
	NamespaceQueue chan string
	Quit           chan struct{}
	*MultiNicNetworkReconciler
}

// NewNamespaceWatcher creates new namespace watcher
func NewNamespaceWatcher(client client.Client, config *rest.Config, multinicnetworkReconciler *MultiNicNetworkReconciler, quit chan struct{}) *NamespaceWatcher {
	clientset, _ := kubernetes.NewForConfig(config)
	watcher := &NamespaceWatcher{
		Clientset:                 clientset,
		NamespaceQueue:            make(chan string),
		Quit:                      quit,
		MultiNicNetworkReconciler: multinicnetworkReconciler,
	}
	factory := informers.NewSharedInformerFactory(clientset, 0)
	nsInformer := factory.Core().V1().Namespaces()

	_, err := nsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if ns, ok := obj.(*v1.Namespace); ok {
				watcher.NamespaceQueue <- ns.Name
			}
		},
	})

	if err != nil {
		vars.NetworkLog.Error(err, "failed to add namespace add event handler")
	}

	factory.Start(watcher.Quit)

	return watcher
}

// Run executes namespace watcher routine until get quit signal
func (w *NamespaceWatcher) Run() {
	defer close(w.NamespaceQueue)
	wait.Until(w.ProcessNamespaceQueue, 0, w.Quit)
}

func (w *NamespaceWatcher) ProcessNamespaceQueue() {
	ns := <-w.NamespaceQueue
	w.MultiNicNetworkReconciler.HandleNewNamespace(ns)
}
