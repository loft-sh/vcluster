module github.com/loft-sh/vcluster

go 1.15

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/loads v0.19.5
	github.com/hashicorp/golang-lru v0.5.4
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/loft-sh/kiosk v0.2.7
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/rhysd/go-github-selfupdate v1.2.3
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.7.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/square/go-jose.v2 v2.2.2
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/apiserver v0.21.1
	k8s.io/client-go v0.21.1
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-aggregator v0.21.1
	k8s.io/kubectl v0.21.1
	k8s.io/kubelet v0.21.1
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.0
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/kubernetes-incubator/reference-docs => github.com/kubernetes-sigs/reference-docs v0.0.0-20170929004150-fcf65347b256
	github.com/markbates/inflect => github.com/markbates/inflect v1.0.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
)
