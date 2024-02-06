package options

const (
	DefaultHostsRewriteImage = "library/alpine:3.13.1"
	GenericConfig            = "CONFIG"
)

// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
	// PRO Options
	ProOptions VirtualClusterProOptions `json:",inline"`

	// OSS Options below
	Controllers []string `json:"controllers,omitempty"`

	ServerCaCert        string   `json:"serverCaCert,omitempty"`
	ServerCaKey         string   `json:"serverCaKey,omitempty"`
	TLSSANs             []string `json:"tlsSans,omitempty"`
	RequestHeaderCaCert string   `json:"requestHeaderCaCert,omitempty"`
	ClientCaCert        string   `json:"clientCaCert,omitempty"`
	KubeConfigPath      string   `json:"kubeConfig,omitempty"`

	KubeConfigContextName     string   `json:"kubeConfigContextName,omitempty"`
	KubeConfigSecret          string   `json:"kubeConfigSecret,omitempty"`
	KubeConfigSecretNamespace string   `json:"kubeConfigSecretNamespace,omitempty"`
	KubeConfigServer          string   `json:"kubeConfigServer,omitempty"`
	Tolerations               []string `json:"tolerations,omitempty"`

	BindAddress string `json:"bindAddress,omitempty"`
	Port        int    `json:"port,omitempty"`

	Name string `json:"name,omitempty"`

	TargetNamespace string `json:"targetNamespace,omitempty"`
	ServiceName     string `json:"serviceName,omitempty"`

	SetOwner bool `json:"setOwner,omitempty"`

	SyncAllNodes        bool     `json:"syncAllNodes,omitempty"`
	EnableScheduler     bool     `json:"enableScheduler,omitempty"`
	DisableFakeKubelets bool     `json:"disableFakeKubelets,omitempty"`
	FakeKubeletIPs      bool     `json:"fakeKubeletIPs,omitempty"`
	ClearNodeImages     bool     `json:"clearNodeImages,omitempty"`
	TranslateImages     []string `json:"translateImages,omitempty"`

	NodeSelector        string `json:"nodeSelector,omitempty"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector,omitempty"`
	ServiceAccount      string `json:"serviceAccount,omitempty"`

	OverrideHosts               bool   `json:"overrideHosts,omitempty"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage,omitempty"`

	ClusterDomain string `json:"clusterDomain,omitempty"`

	LeaderElect   bool  `json:"leaderElect,omitempty"`
	LeaseDuration int64 `json:"leaseDuration,omitempty"`
	RenewDeadline int64 `json:"renewDeadline,omitempty"`
	RetryPeriod   int64 `json:"retryPeriod,omitempty"`

	DisablePlugins      bool     `json:"disablePlugins,omitempty"`
	PluginListenAddress string   `json:"pluginListenAddress,omitempty"`
	Plugins             []string `json:"plugins,omitempty"`

	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	EnforcePodSecurityStandard string `json:"enforcePodSecurityStandard,omitempty"`

	MapHostServices    []string `json:"mapHostServices,omitempty"`
	MapVirtualServices []string `json:"mapVirtualServices,omitempty"`

	SyncLabels []string `json:"syncLabels,omitempty"`

	// hostpath mapper options
	// this is only needed if using vcluster-hostpath-mapper component
	// see: https://github.com/loft-sh/vcluster-hostpath-mapper
	MountPhysicalHostPaths bool `json:"mountPhysicalHostPaths,omitempty"`
	// To enable FSMounts functionality
	VirtualLogsPath          string
	VirtualPodLogsPath       string
	VirtualContainerLogsPath string
	VirtualKubeletPodPath    string

	HostMetricsBindAddress    string `json:"hostMetricsBindAddress,omitempty"`
	VirtualMetricsBindAddress string `json:"virtualMetricsBindAddress,omitempty"`

	MultiNamespaceMode bool     `json:"multiNamespaceMode,omitempty"`
	NamespaceLabels    []string `json:"namespaceLabels,omitempty"`
	SyncAllSecrets     bool     `json:"syncAllSecrets,omitempty"`
	SyncAllConfigMaps  bool     `json:"syncAllConfigMaps,omitempty"`

	ProxyMetricsServer         bool `json:"proxyMetricsServer,omitempty"`
	ServiceAccountTokenSecrets bool `json:"serviceAccountTokenSecrets,omitempty"`

	// DEPRECATED FLAGS
	RewriteHostPaths                   bool `json:"rewriteHostPaths,omitempty"`
	DeprecatedSyncNodeChanges          bool `json:"syncNodeChanges"`
	DeprecatedDisableSyncResources     string
	DeprecatedOwningStatefulSet        string
	DeprecatedUseFakeNodes             bool
	DeprecatedUseFakePersistentVolumes bool
	DeprecatedEnableStorageClasses     bool
	DeprecatedEnablePriorityClasses    bool
	DeprecatedSuffix                   string
	DeprecatedUseFakeKubelets          bool
}
