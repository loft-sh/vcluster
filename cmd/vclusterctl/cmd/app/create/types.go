package create

// Options holds the create cmd options
type Options struct {
	KubeConfigContextName string
	ChartVersion          string
	ChartName             string
	ChartRepo             string
	LocalChartDir         string
	Distro                string
	CIDR                  string
	Values                []string
	SetValues             []string
	DeprecatedExtraValues []string

	KubernetesVersion string

	CreateNamespace    bool
	DisableIngressSync bool
	UpdateCurrent      bool
	Expose             bool
	ExposeLocal        bool

	Connect bool
	Upgrade bool
	Isolate bool

	// Pro
	Project         string
	Cluster         string
	Template        string
	TemplateVersion string
	Links           []string
	Annotations     []string
	Labels          []string
	Params          string
	SetParams       []string
	DisablePro      bool
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
