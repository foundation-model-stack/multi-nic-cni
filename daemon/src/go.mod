module github.ibm.com/CognitiveAdvisor/multi-nic-cni/daemon

go 1.17

require (
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jaypipes/ghw v0.8.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	k8s.io/api v0.23.3
	k8s.io/apiextensions-apiserver v0.23.0
	k8s.io/apimachinery v0.23.3
	k8s.io/client-go v0.23.3
	sigs.k8s.io/controller-runtime v0.11.0
)

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/zapr v1.2.0 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/jaypipes/pcidb v0.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo/v2 v2.1.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20211116205334-6203023598ed // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.23.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.3
	k8s.io/apiserver => k8s.io/apiserver v0.23.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.3
	k8s.io/client-go => k8s.io/client-go v0.23.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.3
	k8s.io/code-generator => k8s.io/code-generator v0.23.3
	k8s.io/component-base => k8s.io/component-base v0.23.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.23.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.23.3
	k8s.io/cri-api => k8s.io/cri-api v0.23.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.3
	k8s.io/kubectl => k8s.io/kubectl v0.23.3
	k8s.io/kubelet => k8s.io/kubelet v0.23.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.3
	k8s.io/metrics => k8s.io/metrics v0.23.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.23.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.3
)
