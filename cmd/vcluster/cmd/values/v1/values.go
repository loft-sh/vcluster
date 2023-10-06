package values

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
)

type Helm struct {
	GlobalAnnotations    map[string]interface{} `json:"globalAnnotations,omitempty"`
	Pro                  bool                   `json:"pro,omitempty"`
	EnableHA             bool                   `json:"enableHA,omitempty"`
	Headless             bool                   `json:"headless,omitempty"`
	DefaultImageRegistry string                 `json:"defaultImageRegistry,omitempty"`
	Plugin               map[string]interface{} `json:"plugin,omitempty"`
	Sync                 SyncValues             `json:"sync,omitempty"`
	FallbackHostDns      bool                   `json:"fallbackHostDns,omitempty"`
	MapServices          MapServices
	Proxy                ProxyValues
	Syncer               SyncerValues
	Vcluster             VclusterValues
	Storage              StorageValues
	Volumes              []corev1.Volume
	ServiceAccount       struct {
		Create bool
	}
	WorkloadServiceAccount struct {
		Annotations map[string]interface{}
	}
	Replicas            uint32
	NodeSelector        corev1.NodeSelector
	Affinity            corev1.Affinity
	PriorityClassName   string
	Tolerations         []corev1.Toleration
	Labels              []map[string]string
	PodLabels           []map[string]string
	Annotations         map[string]string
	PodAnnotations      map[string]string
	PodDisruptionbudget PDBValues
	ServerToken         struct {
		Values       string
		SecretKeyRef corev1.SecretKeySelector
	}
	Service ServiceValues
	Ingress IngressValues
}

type SyncValues struct {
	Services               EnabledSwitch  `json:"services,omitempty"`
	Configmaps             SyncConfigMaps `json:"configmaps,omitempty"`
	Secrets                SyncSecrets    `json:"secrets,omitempty"`
	Endpoints              EnabledSwitch  `json:"endpoints,omitempty"`
	Pods                   SyncPods       `json:"pods,omitempty"`
	Events                 EnabledSwitch  `json:"events,omitempty"`
	PersistentVolumeClaims EnabledSwitch  `json:"persistentVolumeClaims,omitempty"`
	Ingresses              EnabledSwitch  `json:"ingresses,omitempty"`
	Ingressclasses         EnabledSwitch  `json:"ingressclasses,omitempty"`
	FakeNodes              EnabledSwitch  `json:"fake-nodes,omitempty"`
	FakePersistentvolumes  EnabledSwitch  `json:"fake-persistentvolumes,omitempty"`
	Nodes                  SyncNodes      `json:"nodes,omitempty"`
	PersistentVolumes      EnabledSwitch  `json:"persistentVolumes,omitempty"`
	StorageClasses         EnabledSwitch  `json:"storageClasses,omitempty"`
	Hoststorageclasses     EnabledSwitch  `json:"hoststorageclasses,omitempty"`
	Priorityclasses        EnabledSwitch  `json:"priorityclasses,omitempty"`
	Networkpolicies        EnabledSwitch  `json:"networkpolicies,omitempty"`
	Volumesnapshots        EnabledSwitch  `json:"volumesnapshots,omitempty"`
	Poddisruptionbudgets   EnabledSwitch  `json:"poddisruptionbudgets,omitempty"`
	Serviceaccounts        EnabledSwitch  `json:"serviceaccounts,omitempty"`
	Generic                SyncGeneric    `json:"generic,omitempty"`
}

type SyncConfigMaps struct {
	Enabled bool `json:"enabled,omitempty"`
	All     bool `json:"all,omitempty"`
}

type SyncSecrets struct {
	Enabled bool `json:"enabled,omitempty"`
	All     bool `json:"all,omitempty"`
}

type SyncPods struct {
	Enabled             bool `json:"enabled,omitempty"`
	EphemeralContainers bool `json:"ephemeralContainers,omitempty"`
	Status              bool `json:"status,omitempty"`
}

type SyncNodes struct {
	FakeKubeletIPs  bool   `json:"fakeKubeletIPs,omitempty"`
	Enabled         bool   `json:"enabled,omitempty"`
	SyncAllNodes    bool   `json:"syncAllNodes,omitempty"`
	NodeSelector    string `json:"nodeSelector,omitempty"`
	EnableScheduler bool   `json:"enableScheduler,omitempty"`
}

type SyncGeneric struct {
	Config string `json:"config,omitempty"`
}

type EnabledSwitch struct {
	Enabled bool `json:"enabled,omitempty"`
}

type MapServices struct {
	FromVirtual []map[string]interface{}
	FromHost    []map[string]interface{}
}

type ProxyValues struct {
	MetricsServer MetricsProxyServerConfig
}

type MetricsProxyServerConfig struct {
	Nodes EnabledSwitch
	Pods  EnabledSwitch
}

type SyncerValues struct {
	ExtraArgs             []string
	Env                   []corev1.EnvVar
	LivenessProbe         EnabledSwitch
	ReadinessProbe        EnabledSwitch
	VolumeMounts          []corev1.VolumeMount
	ExtraVolumeMounts     []corev1.VolumeMount
	Resources             corev1.ResourceRequirements
	KubeConfigContextName string
	ServiceAnnotations    map[string]interface{}
}

type VclusterValues struct {
	Image             string
	Command           []string
	BaseArgs          []string
	ExtraArgs         []string
	ExtraVolumeMounts []corev1.VolumeMount
	VolumeMounts      []corev1.VolumeMount
	Env               []corev1.EnvVar
	Resources         corev1.ResourceRequirements
}

type StorageValues struct {
	Persistence bool
	Size        string
}

type PDBValues struct {
	Enabled bool
	policyv1.PodDisruptionBudgetSpec
}

type ServiceValues struct {
	Type                     corev1.ServiceType
	ExternalIPs              []string
	ExternalTrafficPolicy    corev1.ServiceExternalTrafficPolicy
	LoadBalancerIP           string
	LoadBalancerSourceRanges []string
	LoadBalancerClass        string
}

type IngressValues struct {
	Enabled          bool
	PathType         string
	ApiVersion       string
	IngressClassName string
	Host             string
	Annotations      map[string]string
	Tls              []networkingv1.IngressTLS
}
