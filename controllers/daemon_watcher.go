/*
 * Copyright 2022- IBM Inc. All rights reserved
 * SPDX-License-Identifier: Apache2.0
 */

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netcogadvisoriov1 "github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator/api/v1"
)

const (
	DEFAULT_DAEMON_NAMESPACE   = "multi-nic-cni"
	DEFAULT_DAEMON_LABEL_NAME  = "app"
	DEFAULT_DAEMON_LABEL_VALUE = "multi-nicd"

	HOST_INTERFACE_LABEL = "hostinterface-nodename"
	JOIN_LABEL_NAME      = "multi-nicd-join"
)

// DaemonWatcher watches daemon pods and updates HostInterface and CIDR
type DaemonWatcher struct {
	*kubernetes.Clientset
	PodQueue chan *v1.Pod
	Quit     chan struct{}
	Log      logr.Logger
	*HostInterfaceHandler
	*CIDRHandler
	DaemonConnector
}

// NewDaemonWatcher creates new daemon watcher
func NewDaemonWatcher(client client.Client, config *rest.Config, logger logr.Logger, hifLog logr.Logger, cidrHandler *CIDRHandler, podQueue chan *v1.Pod, quit chan struct{}) *DaemonWatcher {
	clientset, _ := kubernetes.NewForConfig(config)

	watcher := &DaemonWatcher{
		Clientset: clientset,
		PodQueue:  podQueue,
		Quit:      quit,
		Log:       logger,
		HostInterfaceHandler: &HostInterfaceHandler{
			Client: client,
			Log:    hifLog,
		},
		CIDRHandler: cidrHandler,
		DaemonConnector: DaemonConnector{
			Clientset: clientset,
		},
	}
	// add existing daemon pod to the process queue
	watcher.UpdateCurrentList()

	factory := informers.NewSharedInformerFactory(clientset, 0)
	podInformer := factory.Core().V1().Pods()

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(prevObj, obj interface{}) {
			pod, ok := obj.(*v1.Pod)
			prevPod, _ := prevObj.(*v1.Pod)
			if !ok {
				return
			}
			if isDaemonPod(pod) && prevPod.Status.PodIP == "" && pod.Status.PodIP != "" {
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
	for _, existingDaemon := range initialList.Items {
		w.PodQueue <- existingDaemon.DeepCopy()
	}
	return nil
}

// Run executes daemon watcher routine until get quit signal
func (w *DaemonWatcher) Run() {
	defer close(w.PodQueue)
	w.Log.Info("start watching multi-nic Daemon")
	wait.Until(w.ProcessPodQueue, 0, w.Quit)
}

// ProcessPodQueue creates HostInterface when daemon is not going to be terminated
//                 deletes HostInterface if daemon is deleted
//                 updates CIDR according to the change
func (w *DaemonWatcher) ProcessPodQueue() {
	daemon := <-w.PodQueue
	_, err := w.Clientset.CoreV1().Pods(DAEMON_NAMESPACE).Get(context.TODO(), daemon.ObjectMeta.Name, metav1.GetOptions{})
	if err == nil {
		if daemon.GetDeletionTimestamp() == nil {
			w.Log.Info(fmt.Sprintf("Daemon pod %s update", daemon.GetName()))

			nodeName := daemon.Spec.NodeName
			err = w.addLabel(*daemon, HOST_INTERFACE_LABEL, nodeName)
			if err != nil {
				w.Log.Info(fmt.Sprintf("Fail to add label to %s: %v", daemon.GetName(), err))
			}

			// not terminating, update HostInterface
			err = w.createHostInterfaceInfo(*daemon)
			if err != nil {
				w.Log.Info(fmt.Sprintf("Fail to create hostinterface %s: %v", daemon.GetName(), err))
			}
		}
	} else {
		if errors.IsNotFound(err) {
			// not found, delete HostInterface
			w.HostInterfaceHandler.DeleteHostInterface(daemon.Spec.NodeName)
		}
	}
}

// isDaemonPod checks if created/updated pod label with DEFAULT_DAEMON_LABEL_NAME=DEFAULT_DAEMON_LABEL_VALUE
func isDaemonPod(pod *v1.Pod) bool {
	if val, ok := pod.ObjectMeta.Labels[DEFAULT_DAEMON_LABEL_NAME]; ok {
		if val == DEFAULT_DAEMON_LABEL_VALUE {
			return true
		}
	}
	return false
}

// ipamJoin calls daemon to greet the existing hosts by referring to HostInterface list
func (w *DaemonWatcher) IpamJoin(daemon v1.Pod) error {
	podIP := daemon.Status.PodIP
	if podIP == "" {
		// ip hasn't been assigned yet
		return nil
	}
	hifMap, err := w.HostInterfaceHandler.ListHostInterface()
	var hifs []netcogadvisoriov1.InterfaceInfoType
	for _, hif := range hifMap {
		if hif.Spec.HostName == daemon.Status.PodIP {
			continue
		}
		hifs = append(hifs, hif.Spec.Interfaces...)
	}
	hifLen := len(hifs)
	if lastLenStr, exists := daemon.ObjectMeta.Labels[JOIN_LABEL_NAME]; exists {
		// already join
		lastLen, _ := strconv.ParseInt(lastLenStr, 10, 64)
		if hifLen == int(lastLen) {
			// join with the same number of hostinterfaces
			return nil
		}
	}
	w.Log.Info(fmt.Sprintf("Join %s with %d hifs", daemon.Status.PodIP, hifLen))
	err = w.DaemonConnector.Join(daemon, hifs)
	if err != nil {
		return err
	}
	err = w.addLabel(daemon, JOIN_LABEL_NAME, fmt.Sprintf("%d", hifLen))
	if err != nil {
		w.Log.Info("Fail to add label to %s: %v", daemon.GetName(), err)
	}
	return nil
}

// updateHostInterfaceInfo creates if HostInterface is not exists and calls ipamJoin
func (w *DaemonWatcher) createHostInterfaceInfo(daemon v1.Pod) error {
	interfaces, connectErr := w.DaemonConnector.GetInterfaces(daemon)
	_, hifFoundErr := w.HostInterfaceHandler.GetHostInterface(daemon.Spec.NodeName)
	if hifFoundErr != nil && errors.IsNotFound(hifFoundErr) {
		// not exists, create new HostInterface
		createErr := w.HostInterfaceHandler.CreateHostInterface(daemon.Spec.NodeName, interfaces)
		if connectErr == nil {
			w.IpamJoin(daemon)
		}
		if connectErr != nil || hifFoundErr != nil {
			w.Log.Info(fmt.Sprintf("%v,%v", connectErr, hifFoundErr))
		}
		return createErr
	}
	if connectErr == nil && hifFoundErr == nil {
		return nil
	}
	return fmt.Errorf("%v,%v", connectErr, hifFoundErr)
}

// updateCIDR modifies existing CIDR from the new HostInterface information
func (w *DaemonWatcher) UpdateCIDRs() {
	routeChange := false
	cidrMap, _ := w.CIDRHandler.ListCIDR()
	for cidrName, cidr := range cidrMap {
		w.Log.Info(fmt.Sprintf("Update cidr %s", cidrName))
		change, err := w.CIDRHandler.UpdateCIDR(cidr.Spec, cidr.Spec.Namespace)
		if err != nil {
			w.Log.Info(fmt.Sprintf("Fail to update CIDR: %v", err))
		} else if change {
			routeChange = true
		}
	}
	// call greeting if route changed
	if routeChange {
		podMap, err := w.DaemonConnector.GetDaemonHostMap()
		if err != nil {
			for _, daemon := range podMap {
				w.IpamJoin(daemon)
			}
		}
	}
}

// addLabel labels daemon pod with node name to update HostInterface
type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (w *DaemonWatcher) addLabel(daemon v1.Pod, labelName string, labelValue string) error {
	namespace := daemon.GetNamespace()
	podName := daemon.GetName()
	payload := []patchStringValue{{
		Op:    "replace",
		Path:  fmt.Sprintf("/metadata/labels/%s", labelName),
		Value: labelValue,
	}}
	payloadBytes, _ := json.Marshal(payload)

	_, err := w.Clientset.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	return err
}
