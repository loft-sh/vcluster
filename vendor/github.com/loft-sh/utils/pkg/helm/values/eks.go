package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var EKSAPIVersionMap = map[string]string{
	"1.27": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.27.1-eks-1-27-3",
	"1.26": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.26.4-eks-1-26-9",
	"1.25": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.25.9-eks-1-25-13",
	"1.24": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.24.13-eks-1-24-17",
	"1.23": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.23.17-eks-1-23-22",
	"1.22": "public.ecr.aws/eks-distro/kubernetes/kube-apiserver:v1.22.17-eks-1-22-27",
}

var EKSControllerVersionMap = map[string]string{
	"1.27": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.27.1-eks-1-27-3",
	"1.26": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.26.4-eks-1-26-9",
	"1.25": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.25.9-eks-1-25-13",
	"1.24": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.24.13-eks-1-24-17",
	"1.23": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.23.17-eks-1-23-22",
	"1.22": "public.ecr.aws/eks-distro/kubernetes/kube-controller-manager:v1.22.17-eks-1-22-27",
}

var EKSEtcdVersionMap = map[string]string{
	"1.27": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-27-3",
	"1.26": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-26-9",
	"1.25": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-25-13",
	"1.24": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-24-17",
	"1.23": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-23-22",
	"1.22": "public.ecr.aws/eks-distro/etcd-io/etcd:v3.5.7-eks-1-22-27",
}

var EKSCoreDNSVersionMap = map[string]string{
	"1.27": "public.ecr.aws/eks-distro/coredns/coredns:v1.10.1-eks-1-27-3",
	"1.26": "public.ecr.aws/eks-distro/coredns/coredns:v1.9.3-eks-1-26-9",
	"1.25": "public.ecr.aws/eks-distro/coredns/coredns:v1.9.3-eks-1-25-13",
	"1.24": "public.ecr.aws/eks-distro/coredns/coredns:v1.9.3-eks-1-24-17",
	"1.23": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.7-eks-1-23-22",
	"1.22": "public.ecr.aws/eks-distro/coredns/coredns:v1.8.7-eks-1-22-27",
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
		if serverMinorInt > 26 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.26", serverVersionString)
			apiImage = EKSAPIVersionMap["1.26"]
			controllerImage = EKSControllerVersionMap["1.26"]
			etcdImage = EKSEtcdVersionMap["1.26"]
			corednsImage = EKSCoreDNSVersionMap["1.26"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.22", serverVersionString)
			apiImage = EKSAPIVersionMap["1.22"]
			controllerImage = EKSControllerVersionMap["1.22"]
			etcdImage = EKSEtcdVersionMap["1.22"]
			corednsImage = EKSCoreDNSVersionMap["1.22"]
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
