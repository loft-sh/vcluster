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
	FallbackHostDNS      bool                   `json:"fallbackHostDns,omitempty"`
	MapServices          MapServices            `json:"mapServices,omitempty"`
	Proxy                ProxyValues            `json:"proxy,omitempty"`
	Syncer               SyncerValues           `json:"syncer,omitempty"`
	Vcluster             VclusterValues         `json:"vcluster,omitempty"`
	Storage              StorageValues          `json:"storage,omitempty"`
	Volumes              []corev1.Volume        `json:"volumes,omitempty"`
	ServiceAccount       struct {
		Create bool `json:"create,omitempty"`
	} `json:"serviceAccount,omitempty"`
	WorkloadServiceAccount struct {
		Annotations map[string]interface{} `json:"annotations,omitempty"`
	} `json:"workloadServiceAccount,omitempty"`
	Rbac                RBACValues          `json:"rbac,omitempty"`
	Replicas            uint32              `json:"replicas,omitempty"`
	NodeSelector        corev1.NodeSelector `json:"nodeSelector,omitempty"`
	Affinity            corev1.Affinity     `json:"affinity,omitempty"`
	PriorityClassName   string              `json:"priorityClassName,omitempty"`
	Tolerations         []corev1.Toleration `json:"tolerations,omitempty"`
	Labels              map[string]string   `json:"labels,omitempty"`
	PodLabels           map[string]string   `json:"podLabels,omitempty"`
	Annotations         map[string]string   `json:"annotations,omitempty"`
	PodAnnotations      map[string]string   `json:"podAnnotations,omitempty"`
	PodDisruptionbudget PDBValues           `json:"podDisruptionbudget,omitempty"`
	ServerToken         struct {
		Value        string                   `json:"value,omitempty"`
		SecretKeyRef corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
	} `json:"serverToken,omitempty"`
	Service ServiceValues `json:"service,omitempty"`
	Ingress IngressValues `json:"ingress,omitempty"`

	SecurityContext    corev1.SecurityContext    `json:"securityContext,omitempty"`
	PodSecurityContext corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	Openshift          struct {
		Enable bool `json:"enable,omitempty"`
	} `json:"openshift,omitempty"`
	Coredns            CoreDNSValues    `json:"coredns,omitempty"`
	Isolation          IsolationValues  `json:"isolation,omitempty"`
	Init               InitValues       `json:"init,omitempty"`
	MultiNamespaceMode EnabledSwitch    `json:"multiNamespaceMode,omitempty"`
	Telemetry          TelemetryValues  `json:"telemetry,omitempty"`
	NoopSyncer         NoopSyncerValues `json:"noopSyncer,omitempty"`
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

	// Deprecated: should be removed from the chart first
	SyncNodeChanges bool `json:"syncNodeChanges,omitempty"`
}

type SyncGeneric struct {
	Config string `json:"config,omitempty"`
}

type EnabledSwitch struct {
	Enabled bool `json:"enabled,omitempty"`
}

type MapServices struct {
	FromVirtual []map[string]interface{} `json:"fromVirtual,omitempty"`
	FromHost    []map[string]interface{} `json:"fromHost,omitempty"`
}

type ProxyValues struct {
	MetricsServer MetricsProxyServerConfig `json:"metricsServer,omitempty"`
}

type MetricsProxyServerConfig struct {
	Nodes EnabledSwitch `json:"nodes,omitempty"`
	Pods  EnabledSwitch `json:"pods,omitempty"`
}

type SyncerValues struct {
	ExtraArgs             []string                    `json:"extraArgs,omitempty"`
	Env                   []corev1.EnvVar             `json:"env,omitempty"`
	LivenessProbe         EnabledSwitch               `json:"livenessProbe,omitempty"`
	ReadinessProbe        EnabledSwitch               `json:"readinessProbe,omitempty"`
	VolumeMounts          []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	ExtraVolumeMounts     []corev1.VolumeMount        `json:"extraVolumeMounts,omitempty"`
	Resources             corev1.ResourceRequirements `json:"resources,omitempty"`
	KubeConfigContextName string                      `json:"kubeConfigContextName,omitempty"`
	ServiceAnnotations    map[string]interface{}      `json:"serviceAnnotations,omitempty"`
}

type VclusterValues struct {
	Image             string                      `json:"image,omitempty"`
	Command           []string                    `json:"command,omitempty"`
	BaseArgs          []string                    `json:"baseArgs,omitempty"`
	ExtraArgs         []string                    `json:"extraArgs,omitempty"`
	ExtraVolumeMounts []corev1.VolumeMount        `json:"extraVolumeMounts,omitempty"`
	VolumeMounts      []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	Env               []corev1.EnvVar             `json:"env,omitempty"`
	Resources         corev1.ResourceRequirements `json:"resources,omitempty"`
}

type StorageValues struct {
	Persistence bool   `json:"persistence,omitempty"`
	Size        string `json:"size,omitempty"`
}

// These should be remove from the chart first as they are deprecated there
type RBACValues struct {
	ClusterRole struct {
		Create bool `json:"create,omitempty"`
	} `json:"clusterRole,omitempty"`
	Role struct {
		Create               bool     `json:"create,omitempty"`
		Extended             bool     `json:"extended,omitempty"`
		ExcludedAPIResources []string `json:"excludedAPIResources,omitempty"`
	} `json:"role,omitempty"`
}

type PDBValues struct {
	Enabled bool `json:"enabled,omitempty"`
	policyv1.PodDisruptionBudgetSpec
}

type ServiceValues struct {
	Type                     corev1.ServiceType                  `json:"type,omitempty"`
	ExternalIPs              []string                            `json:"externalIPs,omitempty"`
	ExternalTrafficPolicy    corev1.ServiceExternalTrafficPolicy `json:"externalTrafficPolicy,omitempty"`
	LoadBalancerIP           string                              `json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges []string                            `json:"loadBalancerSourceRanges,omitempty"`
	LoadBalancerClass        string                              `json:"loadBalancerClass,omitempty"`
}

type IngressValues struct {
	Enabled          bool                      `json:"enabled,omitempty"`
	PathType         string                    `json:"pathType,omitempty"`
	APIVersion       string                    `json:"apiVersion,omitempty"`
	IngressClassName string                    `json:"ingressClassName,omitempty"`
	Host             string                    `json:"host,omitempty"`
	Annotations      map[string]string         `json:"annotations,omitempty"`
	TLS              []networkingv1.IngressTLS `json:"tls,omitempty"`
}

type CoreDNSValues struct {
	Integrated     bool                        `json:"integrated,omitempty"`
	Plugin         CoreDNSPluginValues         `json:"plugin,omitempty"`
	Enabled        bool                        `json:"enabled,omitempty"`
	Replicas       uint32                      `json:"replicas,omitempty"`
	NodeSelector   corev1.NodeSelector         `json:"nodeSelector,omitempty"`
	Image          string                      `json:"image,omitempty"`
	Config         string                      `json:"config,omitempty"`
	Service        CoreDNSServiceValues        `json:"service,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	Manifests      string                      `json:"manifests,omitempty"`
	PodAnnotations map[string]string           `json:"podAnnotations,omitempty"`
	PodLabels      map[string]string           `json:"podLabels,omitempty"`
}

type CoreDNSPluginValues struct {
	Enabled bool
	Config  []DNSMappings
}

type DNSMappings struct {
	Record    Record       `json:"record"`
	Target    Target       `json:"target"`
	AllowedOn []FilterSpec `json:"allowedOn"`
	ExceptOn  []FilterSpec `json:"exceptOn"`
}

type Record struct {
	RecordType RecordType
	FQDN       *string `json:"fqdn"`
	Service    *string `json:"service"`
	Namespace  *string `json:"namespace"`
}

type RecordType string
type TargetMode string

type Target struct {
	Mode      TargetMode `json:"mode"`
	Vcluster  *string    `json:"vcluster"`
	URL       *string    `json:"url"`
	Service   *string    `json:"service"`
	Namespace *string    `json:"namespace"`
}

type FilterSpec struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Labels    []string `json:"labels"`
}

type CoreDNSServiceValues struct {
	Type                  corev1.ServiceType                  `json:"type,omitempty"`
	ExternalIPs           []string                            `json:"externalIPs,omitempty"`
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicy `json:"externalTrafficPolicy,omitempty"`
	Annotations           map[string]string                   `json:"annotations,omitempty"`
}

type IsolationValues struct {
	Enabled             bool          `json:"enabled,omitempty"`
	Namespace           *string       `json:"namespace,omitempty"`
	PodSecurityStandard string        `json:"podSecurityStandard,omitempty"`
	NodeProxyPermission EnabledSwitch `json:"nodeProxyPermission,omitempty"`

	ResourceQuota struct {
		Enabled       bool                        `json:"enabled,omitempty"`
		Quota         map[string]interface{}      `json:"quota,omitempty"`
		ScopeSelector corev1.ScopeSelector        `json:"scopeSelector,omitempty"`
		Scopes        []corev1.ResourceQuotaScope `json:"scopes,omitempty"`
	} `json:"resourceQuota,omitempty"`

	LimitRange    IsolationLimitRangeValues `json:"limitRange,omitempty"`
	NetworkPolicy NetworkPolicyValues       `json:"networkPolicy,omitempty"`
}

type IsolationLimitRangeValues struct {
	Enabled        bool                             `json:"enabled,omitempty"`
	Default        IsolationLimitRangeDefaultValues `json:"default,omitempty"`
	DefaultRequest IsolationLimitRangeDefaultValues `json:"defaultRequest,omitempty"`
}

type IsolationLimitRangeDefaultValues struct {
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`
	Memory           string `json:"memory,omitempty"`
	CPU              string `json:"cpu,omitempty"`
}

type NetworkPolicyValues struct {
	Enabled             bool `json:"enabled,omitempty"`
	OutgoingConnections struct {
		IPBlock struct {
			CIDR   string   `json:"cidr,omitempty"`
			Except []string `json:"except,omitempty"`
		} `json:"ipBlock,omitempty"`
	} `json:"outgoingConnections,omitempty"`
}

type InitValues struct {
	Manifests         string           `json:"manifests,omitempty"`
	ManifestsTemplate string           `json:"manifestsTemplate,omitempty"`
	Helm              []InitHelmCharts `json:"helm,omitempty"`
}

type InitHelmCharts struct {
	Bundle string `json:"bundle,omitempty"`
	Chart  struct {
		Name     string `json:"name,omitempty"`
		Version  string `json:"version,omitempty"`
		Repo     string `json:"repo,omitempty"`
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
		Insecure bool   `json:"insecure,omitempty"`
	} `json:"chart,omitempty"`
	Release struct {
		ReleaseName      string `json:"releaseName,omitempty"`
		ReleaseNamespace string `json:"releaseNamespace,omitempty"`
		Timeout          uint32 `json:"timeout,omitempty"`
	} `json:"release,omitempty"`
	Values         string `json:"values,omitempty"`
	ValuesTemplate string `json:"valuesTemplate,omitempty"`
}

type TelemetryValues struct {
	Disabled           *bool  `json:"disabled,omitempty"`
	InstanceCreator    string `json:"instanceCreator,omitempty"`
	InstanceCreatorUID string `json:"instanceCreatorUID,omitempty"`
}

type NoopSyncerValues struct {
	Enabled        bool `json:"enabled,omitempty"`
	Synck8sService bool `json:"synck8SService,omitempty"`
	Secret         struct {
		ServerCaCert        string `json:"serverCaCert,omitempty"`
		ServerCaKey         string `json:"serverCaKey,omitempty"`
		ClientCaCert        string `json:"clientCaCert,omitempty"`
		RequestHeaderCaCert string `json:"requestHeaderCaCert,omitempty"`
		KubeConfig          string `json:"kubeConfig,omitempty"`
	}
}
