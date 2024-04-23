package legacyconfig

// LegacyVirtualClusterOptions holds the cmd flags
type LegacyVirtualClusterOptions struct {
	// PRO Options
	ProOptions LegacyVirtualClusterProOptions `json:",inline"`

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

	SyncAllNodes        bool `json:"syncAllNodes,omitempty"`
	EnableScheduler     bool `json:"enableScheduler,omitempty"`
	DisableFakeKubelets bool `json:"disableFakeKubelets,omitempty"`
	FakeKubeletIPs      bool `json:"fakeKubeletIPs,omitempty"`
	ClearNodeImages     bool `json:"clearNodeImages,omitempty"`

	NodeSelector        string `json:"nodeSelector,omitempty"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector,omitempty"`
	ServiceAccount      string `json:"serviceAccount,omitempty"`

	OverrideHosts               bool   `json:"overrideHosts,omitempty"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage,omitempty"`

	ClusterDomain string `json:"clusterDomain,omitempty"`

	LeaderElect   bool `json:"leaderElect,omitempty"`
	LeaseDuration int  `json:"leaseDuration,omitempty"`
	RenewDeadline int  `json:"renewDeadline,omitempty"`
	RetryPeriod   int  `json:"retryPeriod,omitempty"`

	PluginListenAddress string   `json:"pluginListenAddress,omitempty"`
	Plugins             []string `json:"plugins,omitempty"`

	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	EnforcePodSecurityStandard string `json:"enforcePodSecurityStandard,omitempty"`

	SyncLabels []string `json:"syncLabels,omitempty"`

	// hostpath mapper options
	// this is only needed if using vcluster-hostpath-mapper component
	// see: https://github.com/loft-sh/vcluster-hostpath-mapper
	MountPhysicalHostPaths bool `json:"mountPhysicalHostPaths,omitempty"`

	HostMetricsBindAddress    string `json:"hostMetricsBindAddress,omitempty"`
	VirtualMetricsBindAddress string `json:"virtualMetricsBindAddress,omitempty"`

	MultiNamespaceMode bool `json:"multiNamespaceMode,omitempty"`
	SyncAllSecrets     bool `json:"syncAllSecrets,omitempty"`
	SyncAllConfigMaps  bool `json:"syncAllConfigMaps,omitempty"`

	ProxyMetricsServer         bool `json:"proxyMetricsServer,omitempty"`
	ServiceAccountTokenSecrets bool `json:"serviceAccountTokenSecrets,omitempty"`

	// DEPRECATED FLAGS
	DeprecatedSyncNodeChanges bool `json:"syncNodeChanges"`
}

type LegacyVirtualClusterProOptions struct {
	RemoteKubeConfig  string `json:"remoteKubeConfig,omitempty"`
	RemoteNamespace   string `json:"remoteNamespace,omitempty"`
	RemoteServiceName string `json:"remoteServiceName,omitempty"`
	EtcdReplicas      int    `json:"etcdReplicas,omitempty"`
	IntegratedCoredns bool   `json:"integratedCoreDNS,omitempty"`
	EtcdEmbedded      bool   `json:"etcdEmbedded,omitempty"`

	NoopSyncer            bool `json:"noopSyncer,omitempty"`
	SyncKubernetesService bool `json:"synck8sService,omitempty"`
}

/*
func AddProFlags(flags *pflag.FlagSet, options *VirtualClusterOptions) {
	flags.StringVar(&options.ProOptions.ProLicenseSecret, "pro-license-secret", "", "If set, vCluster.Pro will try to find this secret to retrieve the vCluster.Pro license.")

	flags.StringVar(&options.ProOptions.RemoteKubeConfig, "remote-kube-config", "", "If set, will use the remote kube-config instead of the local in-cluster one. Expects a kube config to a headless vcluster installation")
	flags.StringVar(&options.ProOptions.RemoteNamespace, "remote-namespace", "", "If set, will use this as the remote namespace")
	flags.StringVar(&options.ProOptions.RemoteServiceName, "remote-service-name", "", "If set, will use this as the remote service name")

	flags.BoolVar(&options.ProOptions.IntegratedCoredns, "integrated-coredns", false, "If enabled vcluster will spin an in memory coreDNS inside the syncer container")
	flags.BoolVar(&options.ProOptions.UseCoreDNSPlugin, "use-coredns-plugin", false, "If enabled, the vcluster plugin for coredns will be used")
	flags.BoolVar(&options.ProOptions.NoopSyncer, "noop-syncer", false, "If enabled will setup a noop Syncer that filters and proxies requests to a specified remote cluster")
	flags.BoolVar(&options.ProOptions.SyncKubernetesService, "sync-k8s-service", false, "If enabled will sync the kubernetes service endpoints in the remote cluster with the load balancer ip of this cluster")

	flags.BoolVar(&options.ProOptions.EtcdEmbedded, "etcd-embedded", false, "If true, will start an embedded etcd within vCluster")
	flags.StringVar(&options.ProOptions.MigrateFrom, "migrate-from", "", "The url (including protocol) of the original database")
	flags.IntVar(&options.ProOptions.EtcdReplicas, "etcd-replicas", 0, "The amount of replicas the etcd has")

	flags.StringArrayVar(&options.ProOptions.EnforceValidatingHooks, "enforce-validating-hook", nil, "A validating hook configuration in yaml format encoded with base64. Can be used multiple times")
	flags.StringArrayVar(&options.ProOptions.EnforceMutatingHooks, "enforce-mutating-hook", nil, "A mutating hook configuration in yaml format encoded with base64. Can be used multiple times")
}

func AddFlags(flags *pflag.FlagSet, options *VirtualClusterOptions) {
	flags.StringVar(&options.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	flags.StringSliceVar(&options.Controllers, "sync", []string{}, "A list of sync controllers to enable. 'foo' enables the sync controller named 'foo', '-foo' disables the sync controller named 'foo'")
	flags.StringVar(&options.RequestHeaderCaCert, "request-header-ca-cert", "/data/server/tls/request-header-ca.crt", "The path to the request header ca certificate")
	flags.StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-ca.crt", "The path to the client ca certificate")
	flags.StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/server-ca.crt", "The path to the server ca certificate")
	flags.StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/server-ca.key", "The path to the server ca key")
	flags.StringVar(&options.KubeConfigPath, "kube-config", "/data/server/cred/admin.kubeconfig", "The path to the virtual cluster admin kube config")
	flags.StringSliceVar(&options.TLSSANs, "tls-san", []string{}, "Add additional hostname or IP as a Subject Alternative Name in the TLS cert")

	flags.StringVar(&options.KubeConfigSecret, "out-kube-config-secret", "", "If specified, the virtual cluster will write the generated kube config to the given secret")
	flags.StringVar(&options.KubeConfigSecretNamespace, "out-kube-config-secret-namespace", "", "If specified, the virtual cluster will write the generated kube config in the given namespace")
	flags.StringVar(&options.KubeConfigServer, "out-kube-config-server", "", "If specified, the virtual cluster will use this server for the generated kube config (e.g. https://my-vcluster.domain.com)")

	flags.StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")
	flags.StringVar(&options.ServiceName, "service-name", "", "The service name where the vcluster proxy will be available")
	flags.BoolVar(&options.SetOwner, "set-owner", true, "If true, will set the same owner the currently running syncer pod has on the synced resources")

	flags.StringVar(&options.Name, "name", "", "The name of the virtual cluster")
	flags.StringVar(&options.BindAddress, "bind-address", "0.0.0.0", "The address to bind the server to")
	flags.IntVar(&options.Port, "port", 8443, "The port to bind to")

	flags.BoolVar(&options.SyncAllNodes, "sync-all-nodes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync all nodes instead of only the needed ones")
	flags.BoolVar(&options.EnableScheduler, "enable-scheduler", false, "If enabled, will expect a scheduler running in the virtual cluster")
	flags.BoolVar(&options.DisableFakeKubelets, "disable-fake-kubelets", false, "If disabled, the virtual cluster will not create fake kubelet endpoints to support metrics-servers")
	flags.BoolVar(&options.FakeKubeletIPs, "fake-kubelet-ips", true, "If enabled, virtual cluster will assign fake ips of type NodeInternalIP to fake the kubelets")
	flags.BoolVar(&options.ClearNodeImages, "node-clear-image-status", false, "If enabled, when syncing real nodes, the status.images data will be removed from the vcluster nodes")

	flags.StringSliceVar(&options.TranslateImages, "translate-image", []string{}, "Translates image names from the virtual pod to the physical pod (e.g. coredns/coredns=mirror.io/coredns/coredns)")
	flags.BoolVar(&options.EnforceNodeSelector, "enforce-node-selector", true, "If enabled and --node-selector is set then the virtual cluster will ensure that no pods are scheduled outside of the node selector")
	flags.StringSliceVar(&options.Tolerations, "enforce-toleration", []string{}, "If set will apply the provided tolerations to all pods in the vcluster")
	flags.StringVar(&options.NodeSelector, "node-selector", "", "If nodes sync is enabled, nodes with the given node selector will be synced to the virtual cluster. If fake nodes are used, and --enforce-node-selector flag is set, then vcluster will ensure that no pods are scheduled outside of the node selector.")
	flags.StringVar(&options.ServiceAccount, "service-account", "", "If set, will set this host service account on the synced pods")

	flags.BoolVar(&options.OverrideHosts, "override-hosts", true, "If enabled, vcluster will override a containers /etc/hosts file if there is a subdomain specified for the pod (spec.subdomain).")
	flags.StringVar(&options.OverrideHostsContainerImage, "override-hosts-container-image", DefaultHostsRewriteImage, "The image for the init container that is used for creating the override hosts file.")

	flags.StringVar(&options.ClusterDomain, "cluster-domain", "cluster.local", "The cluster domain ending that should be used for the virtual cluster")

	flags.BoolVar(&options.LeaderElect, "leader-elect", false, "If enabled, syncer will use leader election")
	flags.Int64Var(&options.LeaseDuration, "lease-duration", 60, "Lease duration of the leader election in seconds")
	flags.Int64Var(&options.RenewDeadline, "renew-deadline", 40, "Renew deadline of the leader election in seconds")
	flags.Int64Var(&options.RetryPeriod, "retry-period", 15, "Retry period of the leader election in seconds")

	flags.BoolVar(&options.DisablePlugins, "disable-plugins", false, "If enabled, vcluster will not load any plugins")
	flags.StringVar(&options.PluginListenAddress, "plugin-listen-address", "localhost:10099", "The plugin address to listen to. If this is changed, you'll need to configure your plugins to connect to the updated port")

	flags.StringVar(&options.DefaultImageRegistry, "default-image-registry", "", "This address will be prepended to all deployed system images by vcluster")

	flags.StringVar(&options.EnforcePodSecurityStandard, "enforce-pod-security-standard", "", "This can be set to 'privileged', 'baseline', or 'restricted' to make vcluster enforce these policies during translation.")
	flags.StringSliceVar(&options.SyncLabels, "sync-labels", []string{}, "The specified labels will be synced to physical resources, in addition to their vcluster translated versions. Supported wildcard - '/*'. e.g --sync-labels=my.company/* will sync all labels that match the given prefix 'my.company/'.")
	flags.StringSliceVar(&options.Plugins, "plugins", []string{}, "The plugins to wait for during startup")

	flags.StringSliceVar(&options.MapVirtualServices, "map-virtual-service", []string{}, "Maps a given service inside the virtual cluster to a service inside the host cluster. E.g. default/test=physical-service")
	flags.StringSliceVar(&options.MapHostServices, "map-host-service", []string{}, "Maps a given service inside the host cluster to a service inside the virtual cluster. E.g. other-namespace/my-service=my-vcluster-namespace/my-service")

	flags.StringVar(&options.HostMetricsBindAddress, "host-metrics-bind-address", "0", "If set, metrics for the controller manager for the resources managed in the host cluster will be exposed at this address")
	flags.StringVar(&options.VirtualMetricsBindAddress, "virtual-metrics-bind-address", "0", "If set, metrics for the controller manager for the resources managed in the virtual cluster will be exposed at this address")

	flags.BoolVar(&options.MountPhysicalHostPaths, "mount-physical-host-paths", false, "If enabled, syncer will rewite hostpaths in synced pod volumes")
	flags.BoolVar(&options.MultiNamespaceMode, "multi-namespace-mode", false, "If enabled, syncer will create a namespace for each virtual namespace and use the original names for the synced namespaced resources")
	flags.StringSliceVar(&options.NamespaceLabels, "namespace-labels", []string{}, "Defines one or more labels that will be added to the namespaces synced in the multi-namespace mode. Format: \"labelKey=labelValue\". Multiple values can be passed in a comma-separated string.")
	flags.BoolVar(&options.SyncAllConfigMaps, "sync-all-configmaps", false, "Sync all configmaps from virtual to host cluster")
	flags.BoolVar(&options.SyncAllSecrets, "sync-all-secrets", false, "Sync all secrets from virtual to host cluster")

	flags.BoolVar(&options.ProxyMetricsServer, "proxy-metrics-server", false, "Proxy the host cluster metrics server")
	flags.BoolVar(&options.ServiceAccountTokenSecrets, "service-account-token-secrets", false, "Create secrets for pod service account tokens instead of injecting it as annotations")

	// Deprecated Flags
	flags.BoolVar(&options.RewriteHostPaths, "rewrite-host-paths", false, "If enabled, syncer will rewite hostpaths in synced pod volumes")
	flags.BoolVar(&options.DeprecatedSyncNodeChanges, "sync-node-changes", false, "If enabled and --fake-nodes is false, the virtual cluster will proxy node updates from the virtual cluster to the host cluster. This is not recommended and should only be used if you know what you are doing.")
	flags.BoolVar(&options.DeprecatedUseFakeKubelets, "fake-kubelets", true, "DEPRECATED: use --disable-fake-kubelets instead")
	flags.BoolVar(&options.DeprecatedUseFakeNodes, "fake-nodes", true, "DEPRECATED: use --sync=-fake-nodes instead")
	flags.BoolVar(&options.DeprecatedUseFakePersistentVolumes, "fake-persistent-volumes", true, "DEPRECATED: use --sync=-fake-persistentvolumes instead")
	flags.BoolVar(&options.DeprecatedEnableStorageClasses, "enable-storage-classes", false, "DEPRECATED: use --sync=storageclasses instead")
	flags.BoolVar(&options.DeprecatedEnablePriorityClasses, "enable-priority-classes", false, "DEPRECATED: use --sync=priorityclasses instead")
	flags.StringVar(&options.DeprecatedSuffix, "suffix", "", "DEPRECATED: use --name instead")
	flags.StringVar(&options.DeprecatedOwningStatefulSet, "owning-statefulset", "", "DEPRECATED: use --set-owner instead")
	flags.StringVar(&options.DeprecatedDisableSyncResources, "disable-sync-resources", "", "DEPRECATED: use --sync instead")
}*/
