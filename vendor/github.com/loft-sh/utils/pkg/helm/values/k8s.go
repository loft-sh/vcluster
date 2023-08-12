package values

import (
	"strings"

	"github.com/loft-sh/utils/pkg/helm"
	"github.com/loft-sh/utils/pkg/log"
)

var K8SAPIVersionMap = map[string]string{
	"1.27": "registry.k8s.io/kube-apiserver:v1.27.2",
	"1.26": "registry.k8s.io/kube-apiserver:v1.26.5",
	"1.25": "registry.k8s.io/kube-apiserver:v1.25.10",
	"1.24": "registry.k8s.io/kube-apiserver:v1.24.14",
}

var K8SControllerVersionMap = map[string]string{
	"1.27": "registry.k8s.io/kube-controller-manager:v1.27.2",
	"1.26": "registry.k8s.io/kube-controller-manager:v1.26.5",
	"1.25": "registry.k8s.io/kube-controller-manager:v1.25.10",
	"1.24": "registry.k8s.io/kube-controller-manager:v1.24.14",
}

var K8SSchedulerVersionMap = map[string]string{
	"1.27": "registry.k8s.io/kube-scheduler:v1.27.2",
	"1.26": "registry.k8s.io/kube-scheduler:v1.26.5",
	"1.25": "registry.k8s.io/kube-scheduler:v1.25.10",
	"1.24": "registry.k8s.io/kube-scheduler:v1.24.14",
}

var K8SEtcdVersionMap = map[string]string{
	"1.27": "registry.k8s.io/etcd:3.5.7-0",
	"1.26": "registry.k8s.io/etcd:3.5.6-0",
	"1.25": "registry.k8s.io/etcd:3.5.6-0",
	"1.24": "registry.k8s.io/etcd:3.5.6-0",
}

func getDefaultK8SReleaseValues(chartOptions *helm.ChartOptions, log log.SimpleLogger) (string, error) {
	serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
	serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
	if err != nil {
		return "", err
	}

	apiImage := K8SAPIVersionMap[serverVersionString]
	controllerImage := K8SControllerVersionMap[serverVersionString]
	schedulerImage := K8SSchedulerVersionMap[serverVersionString]
	etcdImage, ok := K8SEtcdVersionMap[serverVersionString]
	if !ok {
		if serverMinorInt > 27 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.27", serverVersionString)
			apiImage = K8SAPIVersionMap["1.27"]
			controllerImage = K8SControllerVersionMap["1.27"]
			schedulerImage = K8SSchedulerVersionMap["1.27"]
			etcdImage = K8SEtcdVersionMap["1.27"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.24", serverVersionString)
			apiImage = K8SAPIVersionMap["1.24"]
			controllerImage = K8SControllerVersionMap["1.24"]
			schedulerImage = K8SSchedulerVersionMap["1.24"]
			etcdImage = K8SEtcdVersionMap["1.24"]
		}
	}

	// build values
	values := `api:
  image: ##API_IMAGE##
scheduler:
  image: ##SCHEDULER_IMAGE##
controller:
  image: ##CONTROLLER_IMAGE##
etcd:
  image: ##ETCD_IMAGE##
`
	values = strings.ReplaceAll(values, "##API_IMAGE##", apiImage)
	values = strings.ReplaceAll(values, "##CONTROLLER_IMAGE##", controllerImage)
	values = strings.ReplaceAll(values, "##SCHEDULER_IMAGE##", schedulerImage)
	values = strings.ReplaceAll(values, "##ETCD_IMAGE##", etcdImage)
	return addCommonReleaseValues(values, chartOptions)
}
