package controllers_test

import (
	"context"
	"time"

	. "github.com/foundation-model-stack/multi-nic-cni/controllers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
)

var _ = Describe("Test Config Controller", func() {
	ConfigReady = true

	It("default config", Serial, func() {
		dummyConfigName := "dummy-config"
		spec := ConfigReconcilerInstance.GetDefaultConfigSpec()
		objMeta := metav1.ObjectMeta{
			Name: dummyConfigName,
		}

		cfg := &multinicv1.Config{
			ObjectMeta: objMeta,
			Spec:       spec,
		}
		ctx := context.TODO()
		By("creating")
		err := ConfigReconcilerInstance.Client.Create(ctx, cfg)
		Expect(err).To(BeNil())
		namespacedName := types.NamespacedName{Name: dummyConfigName, Namespace: metav1.NamespaceAll}
		var config multinicv1.Config
		By("getting")
		Eventually(func(g Gomega) {
			err := ConfigReconcilerInstance.Client.Get(ctx, namespacedName, &config)
			g.Expect(err).To(BeNil())
		}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
		By("new daemonset")
		ds := ConfigReconcilerInstance.NewCNIDaemonSet(dummyConfigName, spec.Daemon)
		Expect(ds).NotTo(BeNil())
		// Wait for DaemonSet to be created by the controller
		Eventually(func(g Gomega) {
			daemonset, err := ConfigReconcilerInstance.Clientset.AppsV1().DaemonSets(OPERATOR_NAMESPACE).Get(ctx, dummyConfigName, metav1.GetOptions{})
			g.Expect(err).To(BeNil())
			g.Expect(daemonset).NotTo(BeNil())
		}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
		By("deleting")
		err = ConfigReconcilerInstance.Client.Delete(ctx, cfg)
		Expect(err).To(BeNil())
		// Wait for Config to be deleted by the controller
		Eventually(func(g Gomega) {
			err := ConfigReconcilerInstance.Client.Get(ctx, namespacedName, &config)
			g.Expect(err).NotTo(BeNil())
		}).WithTimeout(30 * time.Second).WithPolling(2 * time.Second).Should(Succeed())
		// set config ready back to true for the rest test
		ConfigReady = true
	})

	Context("Multus", Ordered, func() {
		ctx := context.TODO()
		cniPath := "/opt/cni/bin"

		BeforeAll(func() {
			multusPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multus-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app": "multus",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "multus-container",
							Image: "multus-image",
						},
					},
					Volumes: []corev1.Volume{
						{Name: "cnibin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: cniPath,
								},
							},
						},
					},
				},
			}
			_, err := ConfigReconcilerInstance.Clientset.CoreV1().Pods("default").Create(ctx, &multusPod, metav1.CreateOptions{})
			Expect(err).To(BeNil())
		})

		It("get CNI path", func() {
			path := ConfigReconcilerInstance.GetCNIHostPath()
			Expect(path).To(Equal(cniPath))
		})
	})
})
