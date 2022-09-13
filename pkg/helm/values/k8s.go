package values

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/helm"
	"github.com/loft-sh/vcluster/pkg/log"
)

var K8SAPIVersionMap = map[string]string{
	"1.25": "k8s.gcr.io/kube-apiserver:v1.25.0",
	"1.24": "k8s.gcr.io/kube-apiserver:v1.24.3",
	"1.23": "k8s.gcr.io/kube-apiserver:v1.23.9",
	"1.22": "k8s.gcr.io/kube-apiserver:v1.22.12",
	"1.21": "k8s.gcr.io/kube-apiserver:v1.21.14",
	"1.20": "k8s.gcr.io/kube-apiserver:v1.20.15",
}

var K8SControllerVersionMap = map[string]string{
	"1.25": "k8s.gcr.io/kube-controller-manager:v1.25.0",
	"1.24": "k8s.gcr.io/kube-controller-manager:v1.24.3",
	"1.23": "k8s.gcr.io/kube-controller-manager:v1.23.9",
	"1.22": "k8s.gcr.io/kube-controller-manager:v1.22.12",
	"1.21": "k8s.gcr.io/kube-controller-manager:v1.21.14",
	"1.20": "k8s.gcr.io/kube-controller-manager:v1.20.15",
}

var K8SSchedulerVersionMap = map[string]string{
	"1.25": "k8s.gcr.io/kube-scheduler:v1.25.0",
	"1.24": "k8s.gcr.io/kube-scheduler:v1.24.3",
	"1.23": "k8s.gcr.io/kube-scheduler:v1.23.9",
	"1.22": "k8s.gcr.io/kube-scheduler:v1.22.12",
	"1.21": "k8s.gcr.io/kube-scheduler:v1.21.14",
	"1.20": "k8s.gcr.io/kube-scheduler:v1.20.15",
}

var K8SEtcdVersionMap = map[string]string{
	"1.25": "k8s.gcr.io/etcd:3.5.4-0",
	"1.24": "k8s.gcr.io/etcd:3.5.1-0",
	"1.23": "k8s.gcr.io/etcd:3.5.1-0",
	"1.22": "k8s.gcr.io/etcd:3.5.1-0",
	"1.21": "k8s.gcr.io/etcd:3.4.13-0",
	"1.20": "k8s.gcr.io/etcd:3.4.13-0",
}

func getDefaultK8SReleaseValues(chartOptions *helm.ChartOptions, log log.Logger) (string, error) {
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
		if serverMinorInt > 25 {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.25", serverVersionString)
			apiImage = K8SAPIVersionMap["1.25"]
			controllerImage = K8SControllerVersionMap["1.25"]
			schedulerImage = K8SSchedulerVersionMap["1.25"]
			etcdImage = K8SEtcdVersionMap["1.25"]
		} else {
			log.Infof("officially unsupported host server version %s, will fallback to virtual cluster version v1.20", serverVersionString)
			apiImage = K8SAPIVersionMap["1.20"]
			controllerImage = K8SControllerVersionMap["1.20"]
			schedulerImage = K8SSchedulerVersionMap["1.20"]
			etcdImage = K8SEtcdVersionMap["1.20"]
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
	return addCommonReleaseValues(values, "", chartOptions)
}
