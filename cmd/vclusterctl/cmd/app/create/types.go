package create

// CreateOptions holds the create cmd options
type CreateOptions struct {
	KubeConfigContextName string
	ChartVersion          string
	ChartName             string
	ChartRepo             string
	LocalChartDir         string
	K3SImage              string
	Distro                string
	CIDR                  string
	ExtraValues           []string

	KubernetesVersion string

	CreateNamespace    bool
	DisableIngressSync bool
	CreateClusterRole  bool
	UpdateCurrent      bool
	Expose             bool
	ExposeLocal        bool

	Connect       bool
	Upgrade       bool
	Isolate       bool
	ReleaseValues string
}

type Values struct {
	Init Init `json:"init" mapstructure:"init"`
}

type Init struct {
	Manifests string      `json:"manifests" mapstructure:"manifests"`
	Helm      []HelmChart `json:"helm" mapstructure:"helm"`
}

type HelmChart struct {
	Bundle    string      `mapstructure:"bundle,omitempty" json:"bundle,omitempty"`
	Name      string      `mapstructure:"name" json:"name,omitempty"`
	Repo      string      `mapstructure:"repo" json:"repo,omitempty"`
	Version   string      `mapstructure:"version" json:"version,omitempty"`
	Namespace string      `mapstructure:"namespace" json:"namespace,omitempty"`
	Values    string      `mapstructure:"values" json:"values,omitempty"`
	Release   HelmRelease `mapstructure:"release" json:"release,omitempty"`
}

type HelmRelease struct {
	Name      string `mapstructure:"name" json:"name,omitempty"`
	Namespace string `mapstructure:"namespace" json:"namespace,omitempty"`
}
