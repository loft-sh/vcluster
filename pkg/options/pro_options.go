package options

type VirtualClusterProOptions struct {
	ProLicenseSecret string `json:"proLicenseSecret,omitempty"`

	RemoteKubeConfig       string   `json:"remoteKubeConfig,omitempty"`
	RemoteNamespace        string   `json:"remoteNamespace,omitempty"`
	RemoteServiceName      string   `json:"remoteServiceName,omitempty"`
	EnforceValidatingHooks []string `json:"enforceValidatingHooks"`
	EnforceMutatingHooks   []string `json:"enforceMutatingHooks"`
	EtcdReplicas           int      `json:"etcdReplicas,omitempty"`
	IntegratedCoredns      bool     `json:"integratedCoreDNS,omitempty"`
	UseCoreDNSPlugin       bool     `json:"useCoreDNSPlugin,omitempty"`
	EtcdEmbedded           bool     `json:"etcdEmbedded,omitempty"`
	MigrateFrom            string   `json:"migrateFrom,omitempty"`

	NoopSyncer            bool `json:"noopSyncer,omitempty"`
	SyncKubernetesService bool `json:"synck8sService,omitempty"`
}
