package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	//+kubebuilder:scaffold:imports

	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
)

func cleanCache() {
	// clear cache to simulate when list cannot be updated at start
	MultiNicnetworkReconcilerInstance.CIDRHandler.DaemonCacheHandler.UnsetCache(fakeNodeName)
	// no daemon pod in cache
	daemonPods := MultiNicnetworkReconcilerInstance.CIDRHandler.DaemonCacheHandler.ListCache()
	Expect(len(daemonPods)).To(Equal(0))
}

var _ = Describe("Daemon Test", func() {
	It("Test TryGetDaemonPod for valid daemon", func() {
		cleanCache()
		daemonPod, err := daemonWatcher.TryGetDaemonPod(fakeNodeName)
		Expect(err).Should(Succeed())
		Expect(daemonPod.Name).Should(Equal(fakeDaemonPodName))
		// daemonPods should be set
		daemonPods := MultiNicnetworkReconcilerInstance.CIDRHandler.DaemonCacheHandler.ListCache()
		Expect(len(daemonPods)).To(Equal(1))
	})

	It("Test TryGetDaemonPod for tainted daemon", func() {
		cleanCache()
		_, err := daemonWatcher.TryGetDaemonPod(fakeNodeName + "-invalid")
		Expect(err).NotTo(BeNil())
		// daemonPods should not be set
		daemonPods := MultiNicnetworkReconcilerInstance.CIDRHandler.DaemonCacheHandler.ListCache()
		Expect(len(daemonPods)).To(Equal(0))
	})

	It("Test IsUnmanaged function", func() {
		newHostName := "unmanagedHost"
		newHif := GenerateNewHostInterface(newHostName, interfaceNames, networkPrefixes, 0)
		Expect(vars.IsUnmanaged(newHif.ObjectMeta)).To(Equal(false))
		newHif.ObjectMeta.Labels[vars.UnmanagedLabelName] = "true"
		Expect(vars.IsUnmanaged(newHif.ObjectMeta)).To(Equal(true))
		newHif.ObjectMeta.Labels[vars.UnmanagedLabelName] = "false"
		Expect(vars.IsUnmanaged(newHif.ObjectMeta)).To(Equal(false))
	})
})
