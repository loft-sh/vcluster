module github.com/loft-sh/vcluster

go 1.15

require (
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/loads v0.19.5
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/loft-sh/kiosk v0.1.25
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/apiserver v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	k8s.io/kube-aggregator v0.20.2
	k8s.io/kubectl v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)

replace github.com/kubernetes-incubator/reference-docs => github.com/kubernetes-sigs/reference-docs v0.0.0-20170929004150-fcf65347b256

replace github.com/markbates/inflect => github.com/markbates/inflect v1.0.4

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
