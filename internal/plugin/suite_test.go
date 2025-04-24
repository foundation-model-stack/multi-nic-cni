package plugin

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	testEnv   *envtest.Environment
	K8sClient client.Client
	Cfg       *rest.Config
)

func TestCompute(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Suite")
}

var _ = BeforeSuite(func() {
	ctx := context.TODO()
	opts := zap.Options{
		Development: true,
		DestWriter:  GinkgoWriter,
		Level:       zapcore.Level(int8(-1)),
	}
	vars.ZapOpts = &opts
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(vars.ZapOpts)))
	vars.SetLog()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases"), filepath.Join("..", "..", "config", "test", "crd")},
		ErrorIfCRDPathMissing: true,
	}
	var err error
	Cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(Cfg).NotTo(BeNil())

	K8sClient, err = client.New(Cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(K8sClient).NotTo(BeNil())

	err = multinicv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	sriovNamespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SRIOV_NAMESPACE,
		},
	}
	Expect(K8sClient.Create(ctx, &sriovNamespace)).Should(Succeed())

	//+kubebuilder:scaffold:scheme
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	Eventually(func(g Gomega) {
		err := testEnv.Stop()
		g.Expect(err).NotTo(HaveOccurred())
	}).WithTimeout(60 * time.Second).WithPolling(1000 * time.Millisecond).Should(Succeed())
})
