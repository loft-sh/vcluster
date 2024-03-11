package config

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
