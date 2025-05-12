package controllers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	. "github.com/foundation-model-stack/multi-nic-cni/controllers"
	"github.com/foundation-model-stack/multi-nic-cni/internal/compute"
	"github.com/foundation-model-stack/multi-nic-cni/internal/plugin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	interfaceNames   []string = []string{"eth1", "eth2"}
	networkPrefixes  []string = []string{"10.242.0.", "10.242.1."}
	networkAddresses []string = []string{"10.242.0.0/24", "10.242.1.0/24", "10.242.2.0/24", "10.242.3.0/24"}
)

var _ = Describe("Test CIDR Handler	", func() {
	cniVersion := "0.3.0"
	cniType := "ipvlan"
	mode := "l2"
	mtu := 1500
	cniArgs := make(map[string]string)
	cniArgs["mode"] = mode
	cniArgs["mtu"] = fmt.Sprintf("%d", mtu)

	Context("Handler functions", Ordered, func() {
		Context("IPAM", func() {
			multinicnetwork := GetMultiNicCNINetwork("test-ipam", cniVersion, cniType, cniArgs)
			var ipamConfig *multinicv1.PluginConfig
			var cidr multinicv1.CIDRSpec
			var handler *CIDRHandler
			var quit chan struct{}
			BeforeAll(func() {
				var err error
				quit = make(chan struct{})
				handler = newCIDRHandler(quit)
				ipamConfig, err = MultiNicnetworkReconcilerInstance.GetIPAMConfig(multinicnetwork)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterAll(func() {
				close(quit)
			})
			BeforeEach(func() {
				var err error
				cidr, err = MultiNicnetworkReconcilerInstance.CIDRHandler.NewCIDR(*ipamConfig, multinicnetwork.GetNamespace())
				Expect(err).NotTo(HaveOccurred())
				cidr = testUpdateCIDR(handler, cidr, true, true)
			})
			It("Dynamically compute CIDR", func() {
				cidr = testUpdateCIDR(handler, cidr, true, true)
				cidr = testUpdateCIDR(handler, cidr, false, false)
				By("Add Host")
				newHostName := "newHost"
				newHostIndex := MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SafeCache.GetSize()
				newHif := GenerateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
				handler.HostInterfaceHandler.SetCache(newHostName, newHif)
				cidr = testUpdateCIDR(handler, cidr, false, true)
				By("Add Interface")
				newInterfaceName := "eth99"
				newNetworkPrefix := "0.0.0."
				newHif = GenerateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
				handler.HostInterfaceHandler.SetCache(newHostName, newHif)
				cidr = testUpdateCIDR(handler, cidr, false, true)
				By("Remove Interface")
				newHif = GenerateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
				handler.HostInterfaceHandler.SetCache(newHostName, newHif)
				cidr = testUpdateCIDR(handler, cidr, false, true)
				By("Add Interface Back")
				newHif = GenerateNewHostInterface(newHostName, append(interfaceNames, newInterfaceName), append(networkPrefixes, newNetworkPrefix), newHostIndex)
				handler.HostInterfaceHandler.SetCache(newHostName, newHif)
				cidr = testUpdateCIDR(handler, cidr, false, true)
				By("Remove Host")
				handler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
				cidr = testUpdateCIDR(handler, cidr, false, true)
				By("Add Host Back")
				handler.HostInterfaceHandler.SetCache(newHostName, newHif)
				testUpdateCIDR(handler, cidr, false, true)
				By("Clean up")
				handler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
			})

			It("Empty subnet", func() {
				emptySubnetMultinicnetwork := GetMultiNicCNINetwork("empty-ipam", cniVersion, cniType, cniArgs)
				emptySubnetMultinicnetwork.Spec.Subnet = ""
				ipamConfig, err := MultiNicnetworkReconcilerInstance.GetIPAMConfig(emptySubnetMultinicnetwork)
				ipamConfig.InterfaceBlock = 0
				ipamConfig.HostBlock = 4
				Expect(err).NotTo(HaveOccurred())
				cidrSpec, err := handler.GenerateCIDRFromHostSubnet(*ipamConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
				hostIPs := handler.GetHostAddressesToExclude()
				Expect(len(hostIPs)).To(BeEquivalentTo(len(interfaceNames) * handler.HostInterfaceHandler.GetSize()))
			})
		})

		Context("Sync", Ordered, func() {
			multinicnetworkName := "test-sync-cidr"
			multinicnetwork := GetMultiNicCNINetwork(multinicnetworkName, cniVersion, cniType, cniArgs)
			validCIDRs := make(map[string]multinicv1.CIDR)
			newHostName := "newHost"
			activePodName := "active-pod"
			activePodNamespace := "default"
			ctx := context.TODO()
			var ipamConfig *multinicv1.PluginConfig
			var cidr multinicv1.CIDRSpec
			var handler *CIDRHandler
			var quit chan struct{}

			BeforeAll(func() {
				var err error
				ipamConfig, err = MultiNicnetworkReconcilerInstance.GetIPAMConfig(multinicnetwork)
				Expect(err).NotTo(HaveOccurred())
			})
			BeforeEach(func() {
				var err error
				quit = make(chan struct{})
				handler = newCIDRHandler(quit)

				cidr, err = handler.NewCIDR(*ipamConfig, multinicnetwork.GetNamespace())
				Expect(err).NotTo(HaveOccurred())
				cidr = testUpdateCIDR(handler, cidr, true, true)
				validCIDRs[multinicnetworkName] = multinicv1.CIDR{
					ObjectMeta: metav1.ObjectMeta{
						Name: multinicnetworkName,
					},
					Spec: cidr,
				}
				By("Create multinicnetwork and CIDR")
				createNewMultiNicNetworkAndCIDR(ctx, handler, multinicnetwork, cidr)
			})
			AfterEach(func() {
				By("Delete multinicnetwork and CIDR")
				deleteMultiNicNetworkAndCIDR(ctx, handler, multinicnetwork)
				close(quit)
			})
			DescribeTable("CIDR and IPPool", func(hasNewHost bool, hasActivePod bool) {
				activeIP := ""
				activeIPPool := ""
				if hasNewHost {
					By("Add Host")
					newHostIndex := handler.HostInterfaceHandler.SafeCache.GetSize()
					newHif := GenerateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
					handler.HostInterfaceHandler.SetCache(newHostName, newHif)
					cidr = testUpdateCIDR(handler, cidr, false, true)
				}
				ippools := handler.IPPoolHandler.SetIPPoolsCache(multinicnetworkName,
					cidr.CIDRs, []compute.IPValue{})
				By(fmt.Sprintf("Set IPPools with length %d: %v", len(ippools), ippools))
				if hasActivePod {
					By("Add Active Pod")
					for poolName, spec := range ippools {
						// get IP from first ippol
						activeIP = DaemonStub.getAddressByIndex(spec.PodCIDR, 1)
						activeIPPool = poolName
						break
					}
					if activeIP != "" {
						createPodWithNetworkStatus(ctx, activePodName, activePodNamespace, multinicnetworkName, []string{activeIP})
					}
				}
				By("Sync")
				newAllocationMap := handler.SyncIPPools(validCIDRs)
				for poolName := range ippools {
					_, found := newAllocationMap[poolName]
					Expect(found).To(BeTrue())
				}
				Expect(len(newAllocationMap)).To(BeEquivalentTo(len(ippools)))
				if activeIP != "" {
					By(fmt.Sprintf("Check Allocation %v", newAllocationMap))
					allocations, found := newAllocationMap[activeIPPool]
					Expect(found).To(BeTrue())
					Expect(allocations).To(HaveLen(1))
					allocation := allocations[0]
					Expect(allocation.Pod).To(BeEquivalentTo(activePodName))
					Expect(allocation.Namespace).To(BeEquivalentTo(activePodNamespace))
				}
				By("Clean Up")
				if hasNewHost {
					handler.HostInterfaceHandler.SafeCache.UnsetCache(newHostName)
					cidr = testUpdateCIDR(handler, cidr, false, true)
				}
				if activeIP != "" {
					err := ConfigReconcilerInstance.Clientset.CoreV1().Pods(activePodNamespace).Delete(ctx, activePodName, metav1.DeleteOptions{})
					Expect(err).To(BeNil())
				}
				handler.IPPoolHandler.UnsetIPPoolsCache(multinicnetworkName,
					cidr.CIDRs)
			},
				Entry("simple", false, false),
				Entry("hasNewHost", true, false),
				Entry("hasActivePod", false, true),
				Entry("hasNewHost and hasActivePod", true, true),
			)

			// DescribeTable("sync with MultiNicNetwork", func(hasPendingCIDR bool) {
			// 	pendingCIDRName := "pending-cidr"
			// 	if hasPendingCIDR {
			// 		By("Add Pending CIDR")
			// 		copiedSpec := *cidr.DeepCopy()
			// 		createNewCIDR(ctx, pendingCIDRName, copiedSpec)
			// 		MultiNicnetworkReconcilerInstance.CIDRHandler.SetCache(pendingCIDRName, copiedSpec)
			// 	}
			// 	By("Clean Up")
			// 	if hasPendingCIDR {
			// 		MultiNicnetworkReconcilerInstance.CIDRHandler.UnsetCache(pendingCIDRName)
			// 	}
			// })
		})

	})

	Context("Util functions", Ordered, func() {
		It("Sync CIDR/IPPool", func() {
			// get index at bytes[3]
			podCIDR := "192.168.0.0/16"
			unsyncedIp := "192.168.0.10"
			contains, index := MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
			Expect(contains).To(Equal(true))
			Expect(index).To(Equal(10))
			// get index at bytes[2]
			unsyncedIp = "192.168.1.1"
			contains, index = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
			Expect(contains).To(Equal(true))
			Expect(index).To(Equal(257))
			// get index at bytes[1]
			podCIDR = "10.0.0.0/8"
			unsyncedIp = "10.1.1.1"
			contains, index = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
			Expect(contains).To(Equal(true))
			Expect(index).To(Equal(256*256 + 256 + 1))
			// uncontain
			podCIDR = "192.168.0.0/26"
			unsyncedIp = "192.168.1.1"
			contains, _ = MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, unsyncedIp)
			Expect(contains).To(Equal(false))
		})
	})

})

func newCIDRHandler(quit chan struct{}) *CIDRHandler {
	hostInterfaceHandler := NewHostInterfaceHandler(Cfg, K8sClient)
	daemonCacheHandler := &DaemonCacheHandler{SafeCache: InitSafeCache()}
	cidrHandler := NewCIDRHandler(K8sClient, Cfg, hostInterfaceHandler, daemonCacheHandler, quit)
	newHostName := "initialHost"
	newHostIndex := MultiNicnetworkReconcilerInstance.CIDRHandler.HostInterfaceHandler.SafeCache.GetSize()
	newHif := GenerateNewHostInterface(newHostName, interfaceNames, networkPrefixes, newHostIndex)
	cidrHandler.HostInterfaceHandler.SetCache(newHostName, newHif)
	return cidrHandler
}

func testUpdateCIDR(handler *CIDRHandler, cidr multinicv1.CIDRSpec, new, expectChange bool) multinicv1.CIDRSpec {
	def := cidr.Config
	excludes := compute.SortAddress(def.ExcludeCIDRs)
	entriesMap, changed := handler.UpdateEntries(cidr, excludes, new)
	Expect(changed).To(Equal(expectChange))

	expectedPodCIDR := 0
	if changed {
		snapshot := handler.HostInterfaceHandler.ListCache()
		Expect(len(snapshot)).Should(BeNumerically(">", 0))
		for _, hif := range snapshot {
			for _, iface := range hif.Spec.Interfaces {
				for _, masterAddresses := range networkAddresses {
					if iface.NetAddress == masterAddresses {
						expectedPodCIDR += 1
						break
					}
				}
			}
		}
		Expect(expectedPodCIDR).Should(BeNumerically(">", 0))
		totalPodCIDR := 0
		for _, entry := range entriesMap {
			totalPodCIDR += len(entry.Hosts)
		}
		Expect(totalPodCIDR).To(Equal(expectedPodCIDR))
	}
	reservedInterfaceIndex := make(map[int]bool)
	newEntries := []multinicv1.CIDREntry{}
	for _, entry := range entriesMap {
		Expect(entry.InterfaceIndex).Should(BeNumerically(">=", 0))
		found := reservedInterfaceIndex[entry.InterfaceIndex]
		Expect(found).To(Equal(false))
		reservedInterfaceIndex[entry.InterfaceIndex] = true
		if len(entry.Hosts) == 0 {
			continue
		}
		newEntries = append(newEntries, entry)
	}
	return multinicv1.CIDRSpec{
		Config: def,
		CIDRs:  newEntries,
	}
}

func createNewCIDR(ctx context.Context, handler *CIDRHandler, name string, spec multinicv1.CIDRSpec) {
	cidr := &multinicv1.CIDR{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: spec,
	}
	err := K8sClient.Create(ctx, cidr)
	Expect(err).To(BeNil())
	Eventually(func(g Gomega) {
		_, err := handler.GetCIDR(name)
		g.Expect(err).To(BeNil())
	}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
}

func createNewMultiNicNetworkAndCIDR(ctx context.Context, handler *CIDRHandler, multinicnetwork *multinicv1.MultiNicNetwork, cidr multinicv1.CIDRSpec) {
	multinicnetworkName := multinicnetwork.Name
	multinicnetwork.ResourceVersion = ""
	err := K8sClient.Create(ctx, multinicnetwork)
	Expect(err).To(BeNil())
	createNewCIDR(ctx, handler, multinicnetworkName, cidr)
	handler.SetCache(multinicnetworkName, cidr)
	Eventually(func(g Gomega) {
		_, err := MultiNicnetworkReconcilerInstance.GetNetwork(multinicnetworkName)
		g.Expect(err).To(BeNil())
	}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
}

func deleteMultiNicNetworkAndCIDR(ctx context.Context, handler *CIDRHandler, multinicnetwork *multinicv1.MultiNicNetwork) {
	multinicnetworkName := multinicnetwork.Name
	err := K8sClient.Delete(ctx, multinicnetwork)
	Expect(err).To(BeNil())
	getCIDR, err := handler.GetCIDR(multinicnetworkName)
	Expect(err).To(BeNil())
	err = K8sClient.Delete(ctx, getCIDR)
	Expect(err).To(BeNil())
	handler.UnsetCache(multinicnetworkName)
	Eventually(func(g Gomega) {
		_, err := MultiNicnetworkReconcilerInstance.GetNetwork(multinicnetworkName)
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
		_, err = handler.GetCIDR(multinicnetworkName)
		g.Expect(errors.IsNotFound(err)).To(BeTrue())
	}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
}

// createPodWithNetworkStatus creates a new pod with multinicnetwork status and wait until available
func createPodWithNetworkStatus(ctx context.Context, podName, podNamespace, multinicnetworkName string, ips []string) {
	networksStatus := []plugin.NetworkStatus{
		{
			Name: multinicnetworkName,
			IPs:  ips,
		},
	}
	networkStatusStr, err := json.Marshal(networksStatus)
	Expect(err).To(BeNil())
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
			Annotations: map[string]string{
				plugin.StatusesKey: string(networkStatusStr),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "fake-image",
				},
			},
		},
	}
	_, err = ConfigReconcilerInstance.Clientset.CoreV1().Pods(podNamespace).Create(ctx, &pod, metav1.CreateOptions{})
	pod.Status.Phase = corev1.PodRunning
	Eventually(func(g Gomega) {
		_, err = ConfigReconcilerInstance.Clientset.CoreV1().Pods(podNamespace).UpdateStatus(ctx, &pod, metav1.UpdateOptions{})
		Expect(err).To(BeNil())
		getPod, err := ConfigReconcilerInstance.Clientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
		g.Expect(err).To(BeNil())
		g.Expect(getPod.Status.Phase).To(BeEquivalentTo(corev1.PodRunning))
	}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
}
