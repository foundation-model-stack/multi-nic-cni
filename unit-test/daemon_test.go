package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//+kubebuilder:scaffold:imports
)

func cleanCache() {
	// clear cache to simulate when list cannot be updated at start
	multinicnetworkReconciler.CIDRHandler.DaemonCacheHandler.UnsetCache(fakeNodeName)
	// no daemon pod in cache
	daemonPods := multinicnetworkReconciler.CIDRHandler.DaemonCacheHandler.ListCache()
	Expect(len(daemonPods)).To(Equal(0))
}

var _ = Describe("Daemon Test", func() {
	It("Test TryGetDaemonPod for valid daemon", func() {
		cleanCache()
		daemonPod, err := daemonWatcher.TryGetDaemonPod(fakeNodeName)
		Expect(err).Should(Succeed())
		Expect(daemonPod.Name).Should(Equal(fakeDaemonPodName))
		// daemonPods should be set
		daemonPods := multinicnetworkReconciler.CIDRHandler.DaemonCacheHandler.ListCache()
		Expect(len(daemonPods)).To(Equal(1))
	})

	It("Test TryGetDaemonPod for tainted daemon", func() {
		cleanCache()
		_, err := daemonWatcher.TryGetDaemonPod(fakeNodeName + "-invalid")
		Expect(err).NotTo(BeNil())
		// daemonPods should not be set
		daemonPods := multinicnetworkReconciler.CIDRHandler.DaemonCacheHandler.ListCache()
		Expect(len(daemonPods)).To(Equal(0))
	})
})
