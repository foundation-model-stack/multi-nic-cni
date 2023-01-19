/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
)

const (
	DEFAULT_DAEMON_NAMESPACE = "multi-nic-cni"
	JOIN_LABEL_NAME          = "multi-nicd-join"
)

// DaemonWatcher watches daemon pods and updates HostInterface and CIDR
type DaemonWatcher struct {
	*kubernetes.Clientset
	PodQueue chan *v1.Pod
	Quit     chan struct{}
	Log      logr.Logger
	*HostInterfaceHandler
	*DaemonCacheHandler
}

func isContainerReady(pod v1.Pod) bool {
	if pod.Status.ContainerStatuses == nil {
		return false
	}
	if len(pod.Status.ContainerStatuses) > 0 {
		return pod.Status.ContainerStatuses[0].Ready
	}
	return false
}

// NewDaemonWatcher creates new daemon watcher
func NewDaemonWatcher(client client.Client, config *rest.Config, logger logr.Logger, hostInterfaceHandler *HostInterfaceHandler, daemonCacheHandler *DaemonCacheHandler, podQueue chan *v1.Pod, quit chan struct{}) *DaemonWatcher {
	clientset, _ := kubernetes.NewForConfig(config)

	watcher := &DaemonWatcher{
		Clientset:            clientset,
		PodQueue:             podQueue,
		Quit:                 quit,
		Log:                  logger,
		HostInterfaceHandler: hostInterfaceHandler,
		DaemonCacheHandler:   daemonCacheHandler,
	}
	// add existing daemon pod to the process queue
	err := watcher.UpdateCurrentList()
	if err != nil {
		watcher.Log.V(4).Info(fmt.Sprintf("cannot UpdateCurrentList: %v", err))
	}

	watcher.Log.V(7).Info("Init Informer")

	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods()

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(prevObj, obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			prevPod, _ := prevObj.(*v1.Pod)
			if !ok {
				return
			}
			if isDaemonPod(pod) && !isContainerReady(*prevPod) && isContainerReady(*pod) {
				// newly-created daemon pod, put to the process queue
				watcher.PodQueue <- pod
			}
		},
	})
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			if !ok {
				return
			}
			if isDaemonPod(pod) {
				// deleted daemon pod, put to the process queue
				watcher.PodQueue <- pod
			}
		},
	})
	factory.Start(watcher.Quit)

	return watcher
}

// getDaemonPods returns all daemon pod
func (w *DaemonWatcher) getDaemonPods() (*v1.PodList, error) {
	labels := fmt.Sprintf("%s=%s", DAEMON_LABEL_NAME, DAEMON_LABEL_VALUE)
	listOptions := metav1.ListOptions{
		LabelSelector: labels,
	}
	return w.Clientset.CoreV1().Pods(DAEMON_NAMESPACE).List(context.TODO(), listOptions)
}

// UpdateCurrentList puts existing daemon pods to the process queue
func (w *DaemonWatcher) UpdateCurrentList() error {
	initialList, err := w.getDaemonPods()
	if err != nil {
		return err
	}
	w.Log.V(4).Info(fmt.Sprintf("Found %d daemons running", len(initialList.Items)))
	for _, existingDaemon := range initialList.Items {
		if isContainerReady(existingDaemon) {
			// early add to the spec for CIDR check
			nodeName := existingDaemon.Spec.NodeName
			daemonPod := DaemonPod{
				Name:      existingDaemon.Name,
				Namespace: existingDaemon.Namespace,
				HostIP:    existingDaemon.Status.HostIP,
				NodeName:  nodeName,
				Labels:    existingDaemon.Labels,
			}
			w.DaemonCacheHandler.SetCache(nodeName, daemonPod)
		}
	}
	return nil
}

// Run executes daemon watcher routine until get quit signal
func (w *DaemonWatcher) Run() {
	defer close(w.PodQueue)
	w.Log.V(7).Info("start watching multi-nic Daemon")
	wait.Until(w.ProcessPodQueue, 0, w.Quit)
}

// ProcessPodQueue creates HostInterface when daemon is not going to be terminated
//
//	deletes HostInterface if daemon is deleted
//	updates CIDR according to the change
func (w *DaemonWatcher) ProcessPodQueue() {
	daemon := <-w.PodQueue
	if daemon != nil {
		nodeName := daemon.Spec.NodeName
		if daemon.GetDeletionTimestamp() == nil {
			w.Log.V(7).Info(fmt.Sprintf("Daemon pod %s for %s update", daemon.GetName(), nodeName))
			// set daemon
			daemonPod := DaemonPod{
				Name:      daemon.Name,
				Namespace: daemon.Namespace,
				HostIP:    daemon.Status.HostIP,
				NodeName:  nodeName,
				Labels:    daemon.Labels,
			}
			w.DaemonCacheHandler.SetCache(nodeName, daemonPod)

			// not terminating, update HostInterface
			err := w.createHostInterfaceInfo(*daemon)
			if err != nil {
				w.Log.V(4).Info(fmt.Sprintf("Fail to create hostinterface %s: %v", daemon.GetName(), err))
			}
		} else {
			w.Log.V(4).Info(fmt.Sprintf("Daemon pod for %s deleted", nodeName))
			// deleted, delete HostInterface
			w.DaemonCacheHandler.SafeCache.UnsetCache(nodeName)
			w.HostInterfaceHandler.DeleteHostInterface(daemon.Spec.NodeName)
		}
	}
}

// isDaemonPod checks if created/updated pod label with DEFAULT_DAEMON_LABEL_NAME=DEFAULT_DAEMON_LABEL_VALUE
func isDaemonPod(pod *v1.Pod) bool {
	if val, ok := pod.ObjectMeta.Labels[DAEMON_LABEL_NAME]; ok {
		if val == DAEMON_LABEL_VALUE {
			return true
		}
	}
	return false
}

// updateHostInterfaceInfo creates if HostInterface is not exists
func (w *DaemonWatcher) createHostInterfaceInfo(daemon v1.Pod) error {
	_, hifFoundErr := w.HostInterfaceHandler.GetHostInterface(daemon.Spec.NodeName)
	if hifFoundErr != nil && errors.IsNotFound(hifFoundErr) {
		// not exists, create new HostInterface
		createErr := w.HostInterfaceHandler.CreateHostInterface(daemon.Spec.NodeName, []multinicv1.InterfaceInfoType{})
		return createErr
	}
	return hifFoundErr
}

func (w *DaemonWatcher) IsDaemonSetReady() bool {
	ds, err := w.Clientset.AppsV1().DaemonSets(DAEMON_NAMESPACE).Get(context.TODO(), DaemonName, metav1.GetOptions{})
	if err == nil {
		return ds.Status.NumberAvailable == ds.Status.DesiredNumberScheduled
	}
	return false
}
