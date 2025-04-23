package controllers

import (
	"time"

	multinicv1 "github.com/foundation-model-stack/multi-nic-cni/api/v1"
	"github.com/foundation-model-stack/multi-nic-cni/internal/vars"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	//+kubebuilder:scaffold:imports
)

var (
	expectedUrgentReconcileTime = 1 * time.Second
	expectedNormalReconcileTime = 1 * time.Minute
	expectedLongReconcileTime   = 1 * time.Minute
	expectedLogLevel            = 7
)

var _ = Describe("Config Test", func() {
	It("Check update from ConfigSpec", func() {
		spec := &multinicv1.ConfigSpec{
			UrgentReconcileSeconds: 1,
			NormalReconcileMinutes: 1,
			LongReconcileMinutes:   1,
			LogLevel:               expectedLogLevel,
		}
		Expect(vars.ConfigLog.V(expectedLogLevel).Enabled()).To(Equal(false))
		configReconciler.UpdateConfigBySpec(spec)
		Expect(vars.UrgentReconcileTime).To(BeEquivalentTo(expectedUrgentReconcileTime))
		Expect(vars.NormalReconcileTime).To(BeEquivalentTo(expectedNormalReconcileTime))
		Expect(vars.LongReconcileTime).To(BeEquivalentTo(expectedLongReconcileTime))
		levelEnabled := vars.ZapOpts.Level.Enabled(zapcore.Level(int8(-expectedLogLevel)))
		Expect(levelEnabled).To(Equal(true))
		Expect(vars.ConfigLog.V(expectedLogLevel).Enabled()).To(Equal(true))
	})
})
