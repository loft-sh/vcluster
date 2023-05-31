package helm

const (
	K3SChart = "vcluster"
	K0SChart = "vcluster-k0s"
	K8SChart = "vcluster-k8s"
	EKSChart = "vcluster-eks"

	K3SProChart = "vcluster-pro"
	K0SProChart = "vcluster-pro-k0s"
	K8SProChart = "vcluster-pro-k8s"
	EKSProChart = "vcluster-pro-eks"
)

// ChartOptions holds the chart options
type ChartOptions struct {
	ChartName           string
	ChartRepo           string
	ChartVersion        string
	CIDR                string
	CreateClusterRole   bool
	DisableIngressSync  bool
	Expose              bool
	NodePort            bool
	SyncNodes           bool
	K3SImage            string
	Isolate             bool
	KubernetesVersion   Version
	DisableTelemetry    bool
	InstanceCreatorType string
	InstanceCreatorUID  string
}

type Version struct {
	Major string
	Minor string
}

func IsVclusterPro(chartName string) bool {
	return chartName == K3SProChart || chartName == K0SProChart || chartName == K8SProChart || chartName == EKSProChart
}
