package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var EKSAPIVersionMap = map[string]string{
	"1.24": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.24.9-eks-1-24-7",
	"1.23": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.23.15-eks-1-23-12",
	"1.22": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.22.17-eks-1-22-17",
	"1.21": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.21.14-eks-1-21-24",
	"1.20": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.20.15-eks-1-20-22",
}

var EKSControllerVersionMap = map[string]string{
	"1.24": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.24.9-eks-1-24-7",
	"1.23": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.23.15-eks-1-23-12",
	"1.22": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.22.17-eks-1-22-17",
	"1.21": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.21.14-eks-1-21-24",
	"1.20": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.20.15-eks-1-20-22",
}

var EKSEtcdVersionMap = map[string]string{
	"1.24": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.6-eks-1-24-7",
	"1.23": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.6-eks-1-23-12",
	"1.22": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.6-eks-1-22-17",
	"1.21": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.4.21-eks-1-21-24",
	"1.20": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.4.21-eks-1-20-22",
}

var EKSCoreDNSVersionMap = map[string]string{
	"1.24": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.7-eks-1-24-7",
	"1.23": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.7-eks-1-23-12",
	"1.22": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.7-eks-1-22-17",
	"1.21": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.4-eks-1-21-24",
	"1.20": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.3-eks-1-20-22",
}

func getDefaultEKSReleaseValues(chartOptions *helm.ChartOptions, log log.SimpleLogger) (string, error) {
	serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
	serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
	if err != nil {
		return "", err
	}

	apiImage := EKSAPIVersionMap[serverVersionString]
	controllerImage := EKSControllerVersionMap[serverVersionString]
	etcdImage := EKSEtcdVersionMap[serverVersionString]
	corednsImage, ok := EKSCoreDNSVersionMap[serverVersionString]
	if !ok {
		if serverMinorInt > 24 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.24", serverVersionString)
			apiImage = EKSAPIVersionMap["1.24"]
			controllerImage = EKSControllerVersionMap["1.24"]
			etcdImage = EKSEtcdVersionMap["1.24"]
			corednsImage = EKSCoreDNSVersionMap["1.24"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.20", serverVersionString)
			apiImage = EKSAPIVersionMap["1.20"]
			controllerImage = EKSControllerVersionMap["1.20"]
			etcdImage = EKSEtcdVersionMap["1.20"]
			corednsImage = EKSCoreDNSVersionMap["1.20"]
		}
	}

	// build values
	values := `api:
  image: ##API_IMAGE##
controller:
  image: ##CONTROLLER_IMAGE##
etcd:
  image: ##ETCD_IMAGE##
coredns:
  image: ##COREDNS_IMAGE##
`
	values = strings.ReplaceAll(values, "##API_IMAGE##", apiImage)
	values = strings.ReplaceAll(values, "##CONTROLLER_IMAGE##", controllerImage)
	values = strings.ReplaceAll(values, "##ETCD_IMAGE##", etcdImage)
	values = strings.ReplaceAll(values, "##COREDNS_IMAGE##", corednsImage)
	return addCommonReleaseValues(values, chartOptions)
}
