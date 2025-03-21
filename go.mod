module github.com/foundation-model-stack/multi-nic-cni

go 1.22.0

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/containernetworking/cni v1.0.1
	github.com/go-logr/logr v1.4.2
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.33.1
	github.com/operator-framework/operator-lib v0.11.0
	github.com/pkg/errors v0.9.1
	go.uber.org/zap v1.26.0
	k8s.io/api v0.31.0
	k8s.io/apimachinery v0.31.0
	k8s.io/client-go v0.31.0
	sigs.k8s.io/controller-runtime v0.19.1
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20220211145901-aa98df527546
