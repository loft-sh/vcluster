package legacyconfig

import (
	"github.com/loft-sh/vcluster/config"
)

type LegacyK0sAndK3s struct {
	BaseHelm
	AutoDeletePersistentVolumeClaims bool               `json:"autoDeletePersistentVolumeClaims,omitempty"`
	K3sToken                         string             `json:"k3sToken,omitempty"`
	VCluster                         VClusterValues     `json:"vcluster,omitempty"`
	EmbeddedEtcd                     EmbeddedEtcdValues `json:"embeddedEtcd,omitempty"`
	Syncer                           SyncerValues       `json:"syncer,omitempty"`
	Storage                          Storage            `json:"storage,omitempty"`
}

func (c *LegacyK0sAndK3s) UnmarshalYAMLStrict(data []byte) error {
	return config.UnmarshalYAMLStrict(data, c)
}

type LegacyK8s struct {
	BaseHelm
	Syncer       K8sSyncerValues    `json:"syncer,omitempty"`
	API          APIServerValues    `json:"api,omitempty"`
	Controller   ControllerValues   `json:"controller,omitempty"`
	Scheduler    SchedulerValues    `json:"scheduler,omitempty"`
	Etcd         EtcdValues         `json:"etcd,omitempty"`
	EmbeddedEtcd EmbeddedEtcdValues `json:"embeddedEtcd,omitempty"`
	Storage      Storage            `json:"storage,omitempty"`
}

func (c *LegacyK8s) UnmarshalYAMLStrict(data []byte) error {
	return config.UnmarshalYAMLStrict(data, c)
}

type K8sSyncerValues struct {
	SyncerValues
	CommonValues
	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	PodSecurityContext map[string]interface{} `json:"podSecurityContext,omitempty"`
}

type APIServerValues struct {
	SyncerExORCommonValues
	ControlPlaneCommonValues
}

type ControllerValues struct {
	SyncerExORCommonValues
	ControlPlaneCommonValues
}

type SchedulerValues struct {
	SyncerExORCommonValues
	ControlPlaneCommonValues
	Disabled bool `json:"disabled,omitempty"`
}

type EtcdValues struct {
	// Disabled is allowed for k8s & eks
	Disabled bool `json:"disabled,omitempty"`
	CommonValues
	SyncerExORCommonValues
	ControlPlaneCommonValues
	SecurityContext                  map[string]interface{} `json:"securityContext,omitempty"`
	ServiceAnnotations               map[string]string      `json:"serviceAnnotations,omitempty"`
	AutoDeletePersistentVolumeClaims bool                   `json:"autoDeletePersistentVolumeClaims,omitempty"`
	Replicas                         int                    `json:"replicas,omitempty"`
	Labels                           map[string]string      `json:"labels,omitempty"`
	Annotations                      map[string]string      `json:"annotations,omitempty"`
	Storage                          Storage                `json:"storage,omitempty"`
}

type MonitoringValues struct {
	ServiceMonitor ServiceMonitor `json:"serviceMonitor,omitempty"`
}

type ServiceMonitor struct {
	Enabled bool `json:"enabled,omitempty"`
}

type EmbeddedEtcdValues struct {
	Enabled         bool `json:"enabled,omitempty"`
	MigrateFromEtcd bool `json:"migrateFromEtcd,omitempty"`
}

type Storage struct {
	Persistence    *bool                    `json:"persistence,omitempty"`
	Size           string                   `json:"size,omitempty"`
	ClassName      string                   `json:"className,omitempty"`
	BinariesVolume []map[string]interface{} `json:"binariesVolume,omitempty"`
}

type BaseHelm struct {
	GlobalAnnotations    map[string]string        `json:"globalAnnotations,omitempty"`
	Pro                  bool                     `json:"pro,omitempty"`
	ProLicenseSecret     string                   `json:"proLicenseSecret,omitempty"`
	Headless             bool                     `json:"headless,omitempty"`
	DefaultImageRegistry string                   `json:"defaultImageRegistry,omitempty"`
	Plugin               map[string]interface{}   `json:"plugin,omitempty"`
	Sync                 SyncValues               `json:"sync,omitempty"`
	FallbackHostDNS      bool                     `json:"fallbackHostDns,omitempty"`
	MapServices          MapServices              `json:"mapServices,omitempty"`
	Proxy                ProxyValues              `json:"proxy,omitempty"`
	Volumes              []map[string]interface{} `json:"volumes,omitempty"`
	ServiceAccount       struct {
		Create           *bool                        `json:"create,omitempty"`
		Name             string                       `json:"name,omitempty"`
		ImagePullSecrets []config.ImagePullSecretName `json:"imagePullSecrets"`
	} `json:"serviceAccount,omitempty"`
	WorkloadServiceAccount struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	} `json:"workloadServiceAccount,omitempty"`
	Rbac                RBACValues               `json:"rbac,omitempty"`
	NodeSelector        map[string]interface{}   `json:"nodeSelector,omitempty"`
	Affinity            map[string]interface{}   `json:"affinity,omitempty"`
	PriorityClassName   string                   `json:"priorityClassName,omitempty"`
	Tolerations         []map[string]interface{} `json:"tolerations,omitempty"`
	Labels              map[string]string        `json:"labels,omitempty"`
	PodLabels           map[string]string        `json:"podLabels,omitempty"`
	Annotations         map[string]string        `json:"annotations,omitempty"`
	PodAnnotations      map[string]string        `json:"podAnnotations,omitempty"`
	PodDisruptionBudget PDBValues                `json:"podDisruptionBudget,omitempty"`
	Service             ServiceValues            `json:"service,omitempty"`
	Ingress             IngressValues            `json:"ingress,omitempty"`

	SecurityContext    map[string]interface{} `json:"securityContext,omitempty"`
	PodSecurityContext map[string]interface{} `json:"podSecurityContext,omitempty"`
	Openshift          struct {
		Enable bool `json:"enable,omitempty"`
	} `json:"openshift,omitempty"`
	Coredns            CoreDNSValues    `json:"coredns,omitempty"`
	Isolation          IsolationValues  `json:"isolation,omitempty"`
	Init               InitValues       `json:"init,omitempty"`
	MultiNamespaceMode EnabledSwitch    `json:"multiNamespaceMode,omitempty"`
	Telemetry          TelemetryValues  `json:"telemetry,omitempty"`
	ServiceCIDR        string           `json:"serviceCIDR,omitempty"`
	NoopSyncer         NoopSyncerValues `json:"noopSyncer,omitempty"`
	Monitoring         MonitoringValues `json:"monitoring,omitempty"`
	CentralAdmission   AdmissionValues  `json:"centralAdmission,omitempty"`
}

type SyncerValues struct {
	ControlPlaneCommonValues
	ExtraArgs             []string                 `json:"extraArgs,omitempty"`
	Env                   []map[string]interface{} `json:"env,omitempty"`
	LivenessProbe         EnabledSwitch            `json:"livenessProbe,omitempty"`
	ReadinessProbe        EnabledSwitch            `json:"readinessProbe,omitempty"`
	VolumeMounts          []map[string]interface{} `json:"volumeMounts,omitempty"`
	ExtraVolumeMounts     []config.VolumeMount     `json:"extraVolumeMounts,omitempty"`
	Resources             config.Resources         `json:"resources,omitempty"`
	KubeConfigContextName string                   `json:"kubeConfigContextName,omitempty"`
	ServiceAnnotations    map[string]string        `json:"serviceAnnotations,omitempty"`
	Replicas              int32                    `json:"replicas,omitempty"`
	Storage               Storage                  `json:"storage,omitempty"`
	Labels                map[string]string        `json:"labels,omitempty"`
	Annotations           map[string]string        `json:"annotations,omitempty"`
}

type SyncValues struct {
	Services               EnabledSwitch  `json:"services,omitempty"`
	Configmaps             SyncConfigMaps `json:"configmaps,omitempty"`
	Secrets                SyncSecrets    `json:"secrets,omitempty"`
	Endpoints              EnabledSwitch  `json:"endpoints,omitempty"`
	Pods                   SyncPods       `json:"pods,omitempty"`
	Events                 EnabledSwitch  `json:"events,omitempty"`
	PersistentVolumeClaims EnabledSwitch  `json:"persistentvolumeclaims,omitempty"`
	Ingresses              EnabledSwitch  `json:"ingresses,omitempty"`
	Ingressclasses         EnabledSwitch  `json:"ingressclasses,omitempty"`
	FakeNodes              EnabledSwitch  `json:"fake-nodes,omitempty"`
	FakePersistentvolumes  EnabledSwitch  `json:"fake-persistentvolumes,omitempty"`
	Nodes                  SyncNodes      `json:"nodes,omitempty"`
	PersistentVolumes      EnabledSwitch  `json:"persistentvolumes,omitempty"`
	StorageClasses         EnabledSwitch  `json:"storageclasses,omitempty"`
	Hoststorageclasses     EnabledSwitch  `json:"hoststorageclasses,omitempty"`
	Priorityclasses        EnabledSwitch  `json:"priorityclasses,omitempty"`
	Networkpolicies        EnabledSwitch  `json:"networkpolicies,omitempty"`
	Volumesnapshots        EnabledSwitch  `json:"volumesnapshots,omitempty"`
	Poddisruptionbudgets   EnabledSwitch  `json:"poddisruptionbudgets,omitempty"`
	Serviceaccounts        EnabledSwitch  `json:"serviceaccounts,omitempty"`
	CSIStorageCapacities   EnabledSwitch  `json:"csistoragecapacities,omitempty"`
	CSINodes               EnabledSwitch  `json:"csinodes,omitempty"`
	CSIDrivers             EnabledSwitch  `json:"csidrivers,omitempty"`
	Generic                SyncGeneric    `json:"generic,omitempty"`
}

type SyncConfigMaps struct {
	Enabled *bool `json:"enabled,omitempty"`
	All     bool  `json:"all,omitempty"`
}

type SyncSecrets struct {
	Enabled *bool `json:"enabled,omitempty"`
	All     bool  `json:"all,omitempty"`
}

type SyncPods struct {
	Enabled             *bool `json:"enabled,omitempty"`
	EphemeralContainers *bool `json:"ephemeralContainers,omitempty"`
	Status              *bool `json:"status,omitempty"`
}

type SyncNodes struct {
	Enabled *bool `json:"enabled,omitempty"`

	FakeKubeletIPs  *bool  `json:"fakeKubeletIPs,omitempty"`
	SyncAllNodes    *bool  `json:"syncAllNodes,omitempty"`
	NodeSelector    string `json:"nodeSelector,omitempty"`
	EnableScheduler *bool  `json:"enableScheduler,omitempty"`
	SyncNodeChanges *bool  `json:"syncNodeChanges,omitempty"`
}

type SyncGeneric struct {
	RBACValues
	Config string `json:"config,omitempty"`
}

type EnabledSwitch struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type MapServices struct {
	FromVirtual []config.ServiceMapping `json:"fromVirtual,omitempty"`
	FromHost    []config.ServiceMapping `json:"fromHost,omitempty"`
}

type ProxyValues struct {
	MetricsServer MetricsProxyServerConfig `json:"metricsServer,omitempty"`
}

type MetricsProxyServerConfig struct {
	Nodes EnabledSwitch `json:"nodes,omitempty"`
	Pods  EnabledSwitch `json:"pods,omitempty"`
}

type VClusterValues struct {
	Image             string                   `json:"image,omitempty"`
	ImagePullPolicy   string                   `json:"imagePullPolicy,omitempty"`
	Command           []string                 `json:"command,omitempty"`
	BaseArgs          []string                 `json:"baseArgs,omitempty"`
	ExtraArgs         []string                 `json:"extraArgs,omitempty"`
	ExtraVolumeMounts []config.VolumeMount     `json:"extraVolumeMounts,omitempty"`
	VolumeMounts      []map[string]interface{} `json:"volumeMounts,omitempty"`
	Env               []map[string]interface{} `json:"env,omitempty"`
	Resources         map[string]interface{}   `json:"resources,omitempty"`

	// this is only provided in context of k0s right now
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// These should be remove from the chart first as they are deprecated there
type RBACValues struct {
	ClusterRole RBACClusterRoleValues `json:"clusterRole,omitempty"`
	Role        RBACRoleValues        `json:"role,omitempty"`
}

type RBACClusterRoleValues struct {
	Create     *bool                    `json:"create,omitempty"`
	ExtraRules []map[string]interface{} `json:"extraRules,omitempty"`
}

type RBACRoleValues struct {
	Create               *bool                    `json:"create,omitempty"`
	ExtraRules           []map[string]interface{} `json:"extraRules,omitempty"`
	ExcludedAPIResources []string                 `json:"excludedApiResources,omitempty"`
}

type RBACRule struct {
	// Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs.
	Verbs []string `json:"verbs" protobuf:"bytes,1,rep,name=verbs"`
	// APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of
	// the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups.
	// +optional
	APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,2,rep,name=apiGroups"`
	// Resources is a list of resources this rule applies to. '*' represents all resources.
	// +optional
	Resources []string `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`
	// ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.
	// +optional
	ResourceNames []string `json:"resourceNames,omitempty" protobuf:"bytes,4,rep,name=resourceNames"`
	// NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path
	// Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding.
	// Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"),  but not both.
	// +optional
	NonResourceURLs []string `json:"nonResourceURLs,omitempty" protobuf:"bytes,5,rep,name=nonResourceURLs"`
}

type PDBValues struct {
	Enabled        bool        `json:"enabled,omitempty"`
	MinAvailable   interface{} `json:"minAvailable,omitempty"`
	MaxUnavailable interface{} `json:"maxUnavailable,omitempty"`
}

type ServiceValues struct {
	Type                     string            `json:"type,omitempty"`
	ExternalIPs              []string          `json:"externalIPs,omitempty"`
	ExternalTrafficPolicy    string            `json:"externalTrafficPolicy,omitempty"`
	LoadBalancerIP           string            `json:"loadBalancerIP,omitempty"`
	LoadBalancerSourceRanges []string          `json:"loadBalancerSourceRanges,omitempty"`
	LoadBalancerClass        string            `json:"loadBalancerClass,omitempty"`
	LoadBalancerAnnotation   map[string]string `json:"loadBalancerAnnotations,omitempty"`
}

type IngressValues struct {
	Enabled          bool              `json:"enabled,omitempty"`
	PathType         string            `json:"pathType,omitempty"`
	IngressClassName string            `json:"ingressClassName,omitempty"`
	Host             string            `json:"host,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	TLS              []interface{}     `json:"tls,omitempty"`
}

type CoreDNSValues struct {
	Enabled        *bool                `json:"enabled,omitempty"`
	Integrated     bool                 `json:"integrated,omitempty"`
	Fallback       string               `json:"fallback,omitempty"`
	Plugin         CoreDNSPluginValues  `json:"plugin,omitempty"`
	Replicas       int                  `json:"replicas,omitempty"`
	NodeSelector   map[string]string    `json:"nodeSelector,omitempty"`
	Image          string               `json:"image,omitempty"`
	Config         string               `json:"config,omitempty"`
	Service        CoreDNSServiceValues `json:"service,omitempty"`
	Resources      *config.Resources    `json:"resources,omitempty"`
	Manifests      string               `json:"manifests,omitempty"`
	PodAnnotations map[string]string    `json:"podAnnotations,omitempty"`
	PodLabels      map[string]string    `json:"podLabels,omitempty"`
}

type CoreDNSPluginValues struct {
	Enabled bool          `json:"enabled,omitempty"`
	Config  []DNSMappings `json:"config,omitempty"`
}

type DNSMappings struct {
	Record    Record       `json:"record,omitempty"`
	Target    Target       `json:"target,omitempty"`
	AllowedOn []FilterSpec `json:"allowedOn,omitempty"`
	ExceptOn  []FilterSpec `json:"exceptOn,omitempty"`
}

type Record struct {
	RecordType RecordType `json:"recordType,omitempty"`
	FQDN       *string    `json:"fqdn,omitempty"`
	Service    *string    `json:"service,omitempty"`
	Namespace  *string    `json:"namespace,omitempty"`
}

type (
	RecordType string
	TargetMode string
)

type Target struct {
	Mode      TargetMode `json:"mode,omitempty"`
	VCluster  *string    `json:"vcluster,omitempty"`
	URL       *string    `json:"url,omitempty"`
	Service   *string    `json:"service,omitempty"`
	Namespace *string    `json:"namespace,omitempty"`
}

type FilterSpec struct {
	Name      string   `json:"name,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

type CoreDNSServiceValues struct {
	Type                  string            `json:"type,omitempty"`
	ExternalIPs           []string          `json:"externalIPs,omitempty"`
	ExternalTrafficPolicy string            `json:"externalTrafficPolicy,omitempty"`
	Annotations           map[string]string `json:"annotations,omitempty"`
}

type IsolationValues struct {
	Enabled             bool          `json:"enabled,omitempty"`
	Namespace           *string       `json:"namespace,omitempty"`
	PodSecurityStandard string        `json:"podSecurityStandard,omitempty"`
	NodeProxyPermission EnabledSwitch `json:"nodeProxyPermission,omitempty"`

	ResourceQuota struct {
		Enabled       *bool                  `json:"enabled,omitempty"`
		Quota         map[string]interface{} `json:"quota,omitempty"`
		ScopeSelector map[string]interface{} `json:"scopeSelector,omitempty"`
		Scopes        []string               `json:"scopes,omitempty"`
	} `json:"resourceQuota,omitempty"`

	LimitRange    IsolationLimitRangeValues `json:"limitRange,omitempty"`
	NetworkPolicy NetworkPolicyValues       `json:"networkPolicy,omitempty"`
}

type IsolationLimitRangeValues struct {
	Enabled        *bool                  `json:"enabled,omitempty"`
	Default        map[string]interface{} `json:"default,omitempty"`
	DefaultRequest map[string]interface{} `json:"defaultRequest,omitempty"`
}

type NetworkPolicyValues struct {
	Enabled             *bool                      `json:"enabled,omitempty"`
	OutgoingConnections config.OutgoingConnections `json:"outgoingConnections,omitempty"`
}

type InitValues struct {
	Manifests         string                          `json:"manifests,omitempty"`
	ManifestsTemplate string                          `json:"manifestsTemplate,omitempty"`
	Helm              []config.ExperimentalDeployHelm `json:"helm,omitempty"`
}

type TelemetryValues struct {
	Disabled           config.StrBool `json:"disabled,omitempty"`
	InstanceCreator    string         `json:"instanceCreator,omitempty"`
	PlatformUserID     string         `json:"platformUserID,omitempty"`
	PlatformInstanceID string         `json:"platformInstanceID,omitempty"`
	MachineID          string         `json:"machineID,omitempty"`
}

type NoopSyncerValues struct {
	Enabled        bool `json:"enabled,omitempty"`
	Synck8sService bool `json:"synck8sService,omitempty"`
	Secret         struct {
		ServerCaCert        string `json:"serverCaCert,omitempty"`
		ServerCaKey         string `json:"serverCaKey,omitempty"`
		ClientCaCert        string `json:"clientCaCert,omitempty"`
		RequestHeaderCaCert string `json:"requestHeaderCaCert,omitempty"`
		KubeConfig          string `json:"kubeConfig,omitempty"`
	} `json:"secret,omitempty"`
}

type AdmissionValues struct {
	ValidatingWebhooks []config.ValidatingWebhookConfiguration `json:"validatingWebhooks,omitempty"`
	MutatingWebhooks   []config.MutatingWebhookConfiguration   `json:"mutatingWebhooks,omitempty"`
}

type ControlPlaneCommonValues struct {
	Image           string `json:"image,omitempty"`
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

type SyncerExORCommonValues struct {
	ExtraArgs []string          `json:"extraArgs,omitempty"`
	Resources *config.Resources `json:"resources,omitempty"`
}

type CommonValues struct {
	Volumes           []map[string]interface{} `json:"volumes,omitempty"`
	PriorityClassName string                   `json:"priorityClassName,omitempty"`
	NodeSelector      map[string]interface{}   `json:"nodeSelector,omitempty"`
	Affinity          map[string]interface{}   `json:"affinity,omitempty"`
	Tolerations       []map[string]interface{} `json:"tolerations,omitempty"`
	PodAnnotations    map[string]string        `json:"podAnnotations,omitempty"`
	PodLabels         map[string]string        `json:"podLabels,omitempty"`
}
