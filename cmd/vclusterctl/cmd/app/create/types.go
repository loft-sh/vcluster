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
	Helm      []HelmChart `json:"helm"      mapstructure:"helm"`
}

type HelmChart struct {
	Bundle    string      `json:"bundle,omitempty"    mapstructure:"bundle,omitempty"`
	Name      string      `json:"name,omitempty"      mapstructure:"name"`
	Repo      string      `json:"repo,omitempty"      mapstructure:"repo"`
	Version   string      `json:"version,omitempty"   mapstructure:"version"`
	Namespace string      `json:"namespace,omitempty" mapstructure:"namespace"`
	Values    string      `json:"values,omitempty"    mapstructure:"values"`
	Release   HelmRelease `json:"release,omitempty"   mapstructure:"release"`
}

type HelmRelease struct {
	Name      string `json:"name,omitempty"      mapstructure:"name"`
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`
}
