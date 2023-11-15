package values

import (
	"strings"

	"github.com/go-logr/logr"
	"github.com/loft-sh/utils/pkg/helm"
)

var K8SAPIVersionMap = map[string]string{
	"1.28": "registry.k8s.io/kube-apiserver:v1.28.2",
	"1.27": "registry.k8s.io/kube-apiserver:v1.27.6",
	"1.26": "registry.k8s.io/kube-apiserver:v1.26.9",
	"1.25": "registry.k8s.io/kube-apiserver:v1.25.14",
}

var K8SControllerVersionMap = map[string]string{
	"1.28": "registry.k8s.io/kube-controller-manager:v1.28.2",
	"1.27": "registry.k8s.io/kube-controller-manager:v1.27.6",
	"1.26": "registry.k8s.io/kube-controller-manager:v1.26.9",
	"1.25": "registry.k8s.io/kube-controller-manager:v1.25.14",
}

var K8SSchedulerVersionMap = map[string]string{
	"1.28": "registry.k8s.io/kube-scheduler:v1.28.2",
	"1.27": "registry.k8s.io/kube-scheduler:v1.27.6",
	"1.26": "registry.k8s.io/kube-scheduler:v1.26.9",
	"1.25": "registry.k8s.io/kube-scheduler:v1.25.14",
}

var K8SEtcdVersionMap = map[string]string{
	"1.28": "registry.k8s.io/etcd:3.5.9-0",
	"1.27": "registry.k8s.io/etcd:3.5.7-0",
	"1.26": "registry.k8s.io/etcd:3.5.6-0",
	"1.25": "registry.k8s.io/etcd:3.5.6-0",
}

func getDefaultK8SReleaseValues(chartOptions *helm.ChartOptions, log logr.Logger) (string, error) {
	apiImage := ""
	controllerImage := ""
	etcdImage := ""
	schedulerImage := ""
	if chartOptions.KubernetesVersion.Major != "" && chartOptions.KubernetesVersion.Minor != "" {
		serverVersionString := GetKubernetesVersion(chartOptions.KubernetesVersion)
		serverMinorInt, err := GetKubernetesMinorVersion(chartOptions.KubernetesVersion)
		if err != nil {
			return "", err
		}

		var ok bool
		apiImage = K8SAPIVersionMap[serverVersionString]
		controllerImage = K8SControllerVersionMap[serverVersionString]
		schedulerImage = K8SSchedulerVersionMap[serverVersionString]
		etcdImage, ok = K8SEtcdVersionMap[serverVersionString]
		if !ok {
			if serverMinorInt > 28 {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.28", "serverVersion", serverVersionString)
				apiImage = K8SAPIVersionMap["1.28"]
				controllerImage = K8SControllerVersionMap["1.28"]
				schedulerImage = K8SSchedulerVersionMap["1.28"]
				etcdImage = K8SEtcdVersionMap["1.28"]
			} else {
				log.Info("officially unsupported host server version, will fallback to virtual cluster version v1.25", "serverVersion", serverVersionString)
				apiImage = K8SAPIVersionMap["1.25"]
				controllerImage = K8SControllerVersionMap["1.25"]
				schedulerImage = K8SSchedulerVersionMap["1.25"]
				etcdImage = K8SEtcdVersionMap["1.25"]
			}
		}
	}

	// build values
	values := ""
	if apiImage != "" {
		values = `api:
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
	}
	return addCommonReleaseValues(values, chartOptions)
}
