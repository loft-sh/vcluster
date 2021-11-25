package create

// CreateOptions holds the create cmd options
type CreateOptions struct {
	ChartVersion string
	ChartName    string
	ChartRepo    string
	K3SImage     string
	Distro       string
	CIDR         string
	ExtraValues  []string

	CreateNamespace    bool
	DisableIngressSync bool
	CreateClusterRole  bool
	Expose             bool
	Connect            bool
	Upgrade            bool

	RunAsUser int64

	ReleaseValues string
}
