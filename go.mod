module github.ibm.com/CognitiveAdvisor/multi-nic-cni-operator

go 1.16

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/containernetworking/cni v1.0.1
	github.com/go-logr/logr v1.2.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	sigs.k8s.io/controller-runtime v0.11.0
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/prometheus/client_golang v1.11.1 // indirect
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20220211145901-aa98df527546
