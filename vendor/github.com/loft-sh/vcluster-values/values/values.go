package values

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
	NodePort           bool
	SyncNodes          bool
	K3SImage           string
	Isolate            bool
	KubernetesVersion  Version
	Pro                bool

	DisableTelemetry    bool
	InstanceCreatorType string
	MachineID           string
	PlatformInstanceID  string
	PlatformUserID      string
}

type Version struct {
	Major string
	Minor string
}
