package helm

import "k8s.io/apimachinery/pkg/version"

const (
	K3SChart = "vcluster"
	K0SChart = "vcluster-k0s"
	K8SChart = "vcluster-k8s"
	EKSChart = "vcluster-eks"
)

// ChartOptions holds the chart options
type ChartOptions struct {
	ChartName          string
	ChartRepo          string
	ChartVersion       string
	CIDR               string
	CreateClusterRole  bool
	DisableIngressSync bool
	Expose             bool
	K3SImage           string
	Isolate            bool
	KubernetesVersion  *version.Info
}
