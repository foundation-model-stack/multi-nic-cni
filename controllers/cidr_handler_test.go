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
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	interfaceNames   []string = []string{"eth1", "eth2"}
	networkPrefixes  []string = []string{"10.242.0.", "10.242.1."}
	networkAddresses []string = []string{"10.242.0.0/24", "10.242.1.0/24", "10.242.2.0/24", "10.242.3.0/24"}
)

var _ = Describe("Test CIDR Handler	", func() {
	ConfigReady = true
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

			It("Empty subnet and interfaces", func() {
				emptySubnetMultinicnetwork := GetMultiNicCNINetwork("empty-ipam", cniVersion, cniType, cniArgs)
				emptySubnetMultinicnetwork.Spec.Subnet = ""
				ipamConfig, err := MultiNicnetworkReconcilerInstance.GetIPAMConfig(emptySubnetMultinicnetwork)
				ipamConfig.InterfaceBlock = 0
				ipamConfig.HostBlock = 4
				Expect(err).NotTo(HaveOccurred())

				By("Testing with existing interfaces")
				cidrSpec, err := handler.GenerateCIDRFromHostSubnet(*ipamConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(cidrSpec.CIDRs)).To(Equal(len(interfaceNames)))
				hostIPs := handler.GetHostAddressesToExclude()
				Expect(len(hostIPs)).To(BeEquivalentTo(len(interfaceNames) * handler.HostInterfaceHandler.GetSize()))

				By("Testing with no interfaces")
				// Clear the HostInterface cache to simulate no interfaces
				handler.HostInterfaceHandler.SafeCache.Clear()

				cidrSpec, err = handler.GenerateCIDRFromHostSubnet(*ipamConfig)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(cidrSpec.CIDRs)).To(Equal(0))
				hostIPs = handler.GetHostAddressesToExclude()
				Expect(len(hostIPs)).To(BeZero())
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

		Context("InitCustomCRCache", func() {
			It("should initialize IPPool and HostInterface caches", func() {
				var handler *CIDRHandler
				var quit chan struct{}
				ctx := context.TODO()
				dummyName := "test-dummy-network"

				By("Create new CIDRHandler")
				quit = make(chan struct{})
				handler = newCIDRHandler(quit)
				defer close(quit)

				By("Creating dummy CIDR")
				dummyCIDRSpec := multinicv1.CIDRSpec{
					Config: multinicv1.PluginConfig{
						Name:           dummyName,
						Type:           "ipvlan",
						Subnet:         "10.0.0.0/16",
						MasterNetAddrs: []string{"eth1"},
					},
					CIDRs: []multinicv1.CIDREntry{
						{
							NetAddress: "10.0.0.0/16",
							Hosts: []multinicv1.HostInterfaceInfo{
								{
									HostIndex:     0,
									HostName:      "host1",
									InterfaceName: "eth1",
									HostIP:        "10.0.0.1",
									PodCIDR:       "10.0.0.1/30",
									IPPool:        "ippool1",
								},
							},
						},
					},
				}
				createNewCIDR(ctx, handler, dummyName, dummyCIDRSpec)
				defer func() {
					cidr, err := handler.GetCIDR(dummyName)
					if err == nil {
						if delErr := K8sClient.Delete(ctx, cidr); delErr != nil {
							fmt.Printf("Warning: failed to delete CIDR: %v\n", delErr)
						}
					}
					handler.UnsetCache(dummyName)
				}()

				By("Creating dummy IPPool")
				dummyIPPool := &multinicv1.IPPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ippool1",
					},
					Spec: multinicv1.IPPoolSpec{
						PodCIDR:     "10.0.0.1/30",
						Allocations: []multinicv1.Allocation{},
						Excludes:    []string{},
					},
				}
				err := K8sClient.Create(ctx, dummyIPPool)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					if err := K8sClient.Delete(ctx, dummyIPPool); err != nil {
						fmt.Printf("Warning: failed to delete IPPool: %v\n", err)
					}
					handler.IPPoolHandler.SafeCache.UnsetCache("ippool1")
				}()

				By("Initializing caches")
				handler.InitCustomCRCache()

				By("Verifying IPPool cache")
				ippoolMap, err := handler.IPPoolHandler.ListIPPool()
				Expect(err).NotTo(HaveOccurred())
				Expect(ippoolMap).To(HaveKey("ippool1"))
				Expect(len(ippoolMap)).To(Equal(1))

				By("Verifying HostInterface cache")
				his := handler.HostInterfaceHandler.ListCache()
				Expect(len(his)).To(BeNumerically(">", 0))

				By("Verifying CIDR is in cache")
				cidr, err := handler.GetCIDR(dummyName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cidr.Spec).To(Equal(dummyCIDRSpec))
			})
		})

		Context("SyncAllPendingCustomCR/syncWithMultinicNetwork/deletePendingCIDR", func() {
			var (
				handler    *CIDRHandler
				defHandler *plugin.NetAttachDefHandler
				scheme     *runtime.Scheme
			)

			BeforeEach(func() {
				// Setup common test resources
				handler = newCIDRHandler(make(chan struct{}))
				scheme = runtime.NewScheme()
				Expect(multinicv1.AddToScheme(scheme)).To(Succeed())
				var err error
				defHandler, err = plugin.GetNetAttachDefHandler(Cfg, scheme)
				Expect(err).To(BeNil())
			})

			It("should sync network attachments and update internal state", func() {
				testCIDRName := "test-network"
				testPodCIDR := "10.0.0.1/30"
				testIPPoolName := handler.GetIPPoolName(testCIDRName, testPodCIDR)
				testCIDRSpec := multinicv1.CIDRSpec{
					Config: multinicv1.PluginConfig{
						Name:           "test-network",
						Type:           "ipvlan",
						Subnet:         "10.0.0.0/16",
						MasterNetAddrs: []string{"eth1", "eth2"},
						HostBlock:      0,
						InterfaceBlock: 0,
						ExcludeCIDRs:   nil,
						VlanMode:       "",
					},
					CIDRs: []multinicv1.CIDREntry{
						{
							NetAddress: "10.0.0.0/16",
							Hosts: []multinicv1.HostInterfaceInfo{
								{
									HostIndex:     0,
									HostName:      "host1",
									InterfaceName: "eth1",
									HostIP:        "10.0.0.1",
									PodCIDR:       testPodCIDR,
									IPPool:        "ippool1",
								},
							},
						},
					},
				}

				// Set up the test state
				By("Creating MultiNicNetwork")
				multinicnetwork := GetMultiNicCNINetwork(testCIDRName, cniVersion, cniType, cniArgs)
				err := handler.Client.Create(context.TODO(), multinicnetwork)
				Expect(err).NotTo(HaveOccurred())

				By("Setting up test CIDR in cache")
				handler.SetCache(testCIDRName, testCIDRSpec)

				By("Creating CIDR resource in fake client")
				cidrObj := &multinicv1.CIDR{
					ObjectMeta: metav1.ObjectMeta{
						Name: testCIDRName,
					},
					Spec: testCIDRSpec,
				}
				err = K8sClient.Create(context.TODO(), cidrObj)
				Expect(err).NotTo(HaveOccurred())

				By("Setting up IPPool in cache")
				mockIPPool := multinicv1.IPPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: testIPPoolName,
					},
					Spec: multinicv1.IPPoolSpec{
						PodCIDR: testPodCIDR,
					},
				}
				handler.IPPoolHandler.SafeCache.SetCache(testIPPoolName, mockIPPool.Spec)

				// Act
				By("Calling SyncAllPendingCustomCR")
				handler.SyncAllPendingCustomCR(defHandler)

				// Assert
				By("Verifying CIDR state")
				Eventually(func(g Gomega) {
					cidr, err := handler.GetCIDR(testCIDRName)
					g.Expect(err).NotTo(HaveOccurred())
					Expect(cidr).NotTo(BeNil())
					Expect(cidr.Spec.Config.Name).To(Equal(testCIDRName))
					Expect(cidr.Spec.Config.Subnet).To(Equal("10.0.0.0/16"))
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())

				By("Verifying IPPool state")
				Eventually(func(g Gomega) {
					_, err := handler.IPPoolHandler.GetIPPool(testIPPoolName)
					g.Expect(err).NotTo(HaveOccurred())
				}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
			})

			It("should delete CIDR when no corresponding MultiNicNetwork exists", func() {
				// Create CIDR without MultiNicNetwork
				cidrName := "test-no-network"
				cidrSpec := multinicv1.CIDRSpec{
					Config: multinicv1.PluginConfig{
						Name:           cidrName,
						Type:           "ipvlan",
						Subnet:         "10.0.0.0/16",
						MasterNetAddrs: []string{"eth1"},
					},
					CIDRs: []multinicv1.CIDREntry{
						{
							NetAddress:     "10.0.0.0/16",
							InterfaceIndex: 0,
							Hosts: []multinicv1.HostInterfaceInfo{
								{
									HostIndex:     0,
									HostName:      "host1",
									InterfaceName: "eth1",
									HostIP:        "10.0.0.1",
									PodCIDR:       "10.0.0.1/30",
								},
							},
						},
					},
				}

				cidr := &multinicv1.CIDR{
					ObjectMeta: metav1.ObjectMeta{
						Name: cidrName,
					},
					Spec: cidrSpec,
				}
				err := K8sClient.Create(context.TODO(), cidr)
				Expect(err).NotTo(HaveOccurred())

				// Add
				handler.SetCache(cidrName, cidrSpec)

				// Act
				handler.SyncAllPendingCustomCR(defHandler)

				// Assert: CIDR should be deleted
				Eventually(func() bool {
					_, err := handler.GetCIDR(cidrName)
					return errors.IsNotFound(err)
				}, "5s", "100ms").Should(BeTrue(), "CIDR should be deleted")
			})

			It("should keep CIDR when corresponding MultiNicNetwork exists", func() {
				// Create MultiNicNetwork first
				netName := "test-with-network"
				network := GetMultiNicCNINetwork(netName, "0.3.0", "ipvlan", nil)
				err := K8sClient.Create(context.TODO(), network)
				Expect(err).NotTo(HaveOccurred())

				// Create CIDR
				cidrSpec := multinicv1.CIDRSpec{
					Config: multinicv1.PluginConfig{
						Name:           netName,
						Type:           "ipvlan",
						Subnet:         "10.0.0.0/16",
						MasterNetAddrs: []string{"eth1"},
					},
					CIDRs: []multinicv1.CIDREntry{
						{
							NetAddress:     "10.0.0.0/16",
							InterfaceIndex: 0,
							Hosts: []multinicv1.HostInterfaceInfo{
								{
									HostIndex:     0,
									HostName:      "host1",
									InterfaceName: "eth1",
									HostIP:        "10.0.0.1",
									PodCIDR:       "10.0.0.1/30",
								},
							},
						},
					},
				}

				cidr := &multinicv1.CIDR{
					ObjectMeta: metav1.ObjectMeta{
						Name: netName,
					},
					Spec: cidrSpec,
				}
				err = K8sClient.Create(context.TODO(), cidr)
				Expect(err).NotTo(HaveOccurred())

				// Add
				handler.SetCache(netName, cidrSpec)

				// Act
				handler.SyncAllPendingCustomCR(defHandler)

				// Assert: CIDR should still exist
				var foundCIDR *multinicv1.CIDR
				Eventually(func() error {
					foundCIDR, err = handler.GetCIDR(netName)
					return err
				}, "5s", "100ms").ShouldNot(HaveOccurred())

				Expect(foundCIDR.Spec).To(Equal(cidrSpec))
			})

			AfterEach(func() {
				// Cleanup resources
				ctx := context.Background()
				err := K8sClient.DeleteAllOf(ctx, &multinicv1.CIDR{})
				Expect(err).NotTo(HaveOccurred())
				err = K8sClient.DeleteAllOf(ctx, &multinicv1.MultiNicNetwork{})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("Util functions", func() {
		Context("GetAllNetAddrs", func() {
			It("returns all unique network addresses from HostInterfaceHandler", func() {
				// Prepare mock HostInterfaceHandler cache
				handler := newCIDRHandler(make(chan struct{}))

				// Clear the cache to remove the "initialHost" added by newCIDRHandler
				handler.HostInterfaceHandler.SafeCache.Clear()

				mockCache := map[string]multinicv1.HostInterface{
					"host1": {
						Spec: multinicv1.HostInterfaceSpec{
							Interfaces: []multinicv1.InterfaceInfoType{
								{NetAddress: "10.0.0.1"},
								{NetAddress: "10.0.0.2"},
							},
						},
					},
					"host2": {
						Spec: multinicv1.HostInterfaceSpec{
							Interfaces: []multinicv1.InterfaceInfoType{
								{NetAddress: "10.0.0.3"},
								{NetAddress: "10.0.0.4"},
							},
						},
					},
				}

				// Populate the SafeCache with the mock data
				for hostName, hostInterface := range mockCache {
					handler.HostInterfaceHandler.SafeCache.SetCache(hostName, hostInterface)
				}

				addrs := handler.GetAllNetAddrs()
				Expect(addrs).To(ContainElements("10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"))
				Expect(len(addrs)).To(Equal(4))

				// Clean up the SafeCache after the test
				handler.HostInterfaceHandler.SafeCache.Clear()
			})
		})

		Context("GetHostInterfaceIndexMap", func() {
			It("returns a map from (host name, interface index) to HostInterfaceInfo of CIDR", func() {
				// Prepare mock CIDREntries
				cidrEntries := []multinicv1.CIDREntry{
					{
						NetAddress:     "10.0.0.0/24",
						InterfaceIndex: 0,
						VlanCIDR:       "10.0.0.0/24",
						Hosts: []multinicv1.HostInterfaceInfo{
							{
								HostIndex:     0,
								HostName:      "host1",
								InterfaceName: "eth0",
								HostIP:        "10.0.0.1",
								PodCIDR:       "10.0.0.1/30",
								IPPool:        "ippool1",
							},
							{
								HostIndex:     1,
								HostName:      "host2",
								InterfaceName: "eth0",
								HostIP:        "10.0.0.2",
								PodCIDR:       "10.0.0.2/30",
								IPPool:        "ippool2",
							},
						},
					},
					{
						NetAddress:     "10.0.1.0/24",
						InterfaceIndex: 1,
						VlanCIDR:       "10.0.1.0/24",
						Hosts: []multinicv1.HostInterfaceInfo{
							{
								HostIndex:     0,
								HostName:      "host1",
								InterfaceName: "eth1",
								HostIP:        "10.0.1.1",
								PodCIDR:       "10.0.1.1/30",
								IPPool:        "ippool3",
							},
						},
					},
				}

				// Create a CIDRHandler
				handler := newCIDRHandler(make(chan struct{}))

				// Act
				hostInterfaceIndexMap := handler.GetHostInterfaceIndexMap(cidrEntries)

				// Assertions
				Expect(hostInterfaceIndexMap).To(HaveLen(2)) // Expect 2 hosts

				// Check host1
				Expect(hostInterfaceIndexMap["host1"]).To(HaveLen(2)) // Expect 2 interfaces
				Expect(hostInterfaceIndexMap["host1"][0].InterfaceName).To(Equal("eth0"))
				Expect(hostInterfaceIndexMap["host1"][1].InterfaceName).To(Equal("eth1"))

				// Check host2
				Expect(hostInterfaceIndexMap["host2"]).To(HaveLen(1)) // Expect 1 interface
				Expect(hostInterfaceIndexMap["host2"][0].InterfaceName).To(Equal("eth0"))
			})

			It("returns an empty map when there are no CIDR entries", func() {
				// Prepare an empty slice of CIDREntries
				cidrEntries := []multinicv1.CIDREntry{}

				// Create a CIDRHandler
				handler := newCIDRHandler(make(chan struct{}))

				// Act
				hostInterfaceIndexMap := handler.GetHostInterfaceIndexMap(cidrEntries)

				// Assertions
				Expect(hostInterfaceIndexMap).To(BeEmpty()) // Expect an empty map
			})

			It("handles CIDREntries with empty Host lists", func() {
				// Prepare mock CIDREntries with empty Host lists
				cidrEntries := []multinicv1.CIDREntry{
					{
						NetAddress:     "10.0.0.0/24",
						InterfaceIndex: 0,
						VlanCIDR:       "10.0.0.0/24",
						Hosts:          []multinicv1.HostInterfaceInfo{}, // Empty Host list
					},
				}

				// Create a CIDRHandler
				handler := newCIDRHandler(make(chan struct{}))

				// Act
				hostInterfaceIndexMap := handler.GetHostInterfaceIndexMap(cidrEntries)

				// Assertions
				Expect(hostInterfaceIndexMap).To(BeEmpty()) // Expect an empty map
			})
		})

		Context("Sync CIDR/IPPool", func() {
			DescribeTable("Getting index in range",
				func(podCIDR, testIP string, expectedContains bool, expectedIndex int) {
					contains, index := MultiNicnetworkReconcilerInstance.CIDRHandler.CIDRCompute.GetIndexInRange(podCIDR, testIP)
					Expect(contains).To(Equal(expectedContains))
					if expectedContains {
						Expect(index).To(Equal(expectedIndex))
					}
				},
				Entry("index at bytes[3]", "192.168.0.0/16", "192.168.0.10", true, 10),
				Entry("index at bytes[2]", "192.168.0.0/16", "192.168.1.1", true, 257),
				Entry("index at bytes[1]", "10.0.0.0/8", "10.1.1.1", true, 256*256+256+1),
				Entry("uncontained address", "192.168.0.0/26", "192.168.1.1", false, 0),
			)
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
