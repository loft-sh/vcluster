package config

type Config struct {
	ExportKubeConfig ExportKubeConfig   `yaml:"exportKubeConfig,omitempty" json:"exportKubeConfig,omitempty"`
	Sync             Sync               `yaml:"sync,omitempty" json:"sync,omitempty"`
	Observability    Observability      `yaml:"observability,omitempty" json:"observability,omitempty"`
	Networking       Networking         `yaml:"networking,omitempty" json:"networking,omitempty"`
	Plugin           map[string]Plugin  `yaml:"plugin,omitempty" json:"plugin,omitempty"`
	Plugins          map[string]Plugins `yaml:"plugins,omitempty" json:"plugins,omitempty"`
	ControlPlane     ControlPlane       `yaml:"controlPlane,omitempty" json:"controlPlane,omitempty"`
	Policies         Policies           `yaml:"policies,omitempty" json:"policies,omitempty"`
	RBAC             RBAC               `yaml:"rbac,omitempty" json:"rbac,omitempty"`

	// Telemetry is the configuration related to telemetry gathered about vcluster usage.
	Telemetry    Telemetry          `yaml:"telemetry,omitempty" json:"telemetry,omitempty"`
	Experimental Experimental       `yaml:"experimental,omitempty" json:"experimental,omitempty"`
	License      SecretKeyReference `yaml:"license,omitempty" json:"license,omitempty"`
	Platform     Platform           `yaml:"platform,omitempty" json:"platform,omitempty"`
}

type ExportKubeConfig struct {
	Context string          `yaml:"context" json:"context"`
	Server  string          `yaml:"server" json:"server"`
	Secret  SecretReference `yaml:"secret,omitempty" json:"secret,omitempty"`
}

// SecretReference represents a Secret Reference. It has enough information to retrieve secret
// in any namespace
type SecretReference struct {
	// name is unique within a namespace to reference a secret resource.
	Name string `json:"name,omitempty"`

	// namespace defines the space within which the secret name must be unique.
	Namespace string `json:"namespace,omitempty"`
}

type Sync struct {
	ToHost   SyncToHost
	FromHost SyncFromHost
}

type SyncToHost struct {
	Services               EnableSwitch    `yaml:"services,omitempty" json:"services,omitempty"`
	Endpoints              EnableSwitch    `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`
	Ingresses              EnableSwitch    `yaml:"ingresses,omitempty" json:"ingresses,omitempty"`
	PriorityClasses        EnableSwitch    `yaml:"priorityClasses,omitempty" json:"priorityClasses,omitempty"`
	NetworkPolicies        EnableSwitch    `yaml:"networkPolicies,omitempty" json:"networkPolicies,omitempty"`
	VolumeSnapshots        EnableSwitch    `yaml:"volumeSnapshots,omitempty" json:"volumeSnapshots,omitempty"`
	PodDisruptionBudgets   EnableSwitch    `yaml:"podDisruptionBudgets,omitempty" json:"podDisruptionBudgets,omitempty"`
	ServiceAccounts        EnableSwitch    `yaml:"serviceAccounts,omitempty" json:"serviceAccounts,omitempty"`
	StorageClasses         EnableSwitch    `yaml:"storageClasses,omitempty" json:"storageClasses,omitempty"`
	PersistentVolumes      EnableSwitch    `yaml:"persistentVolumes,omitempty" json:"persistentVolumes,omitempty"`
	PersistentVolumeClaims EnableSwitch    `yaml:"persistentVolumeClaims,omitempty" json:"persistentVolumeClaims,omitempty"`
	ConfigMaps             SyncAllResource `yaml:"configMaps,omitempty" json:"configMaps,omitempty"`
	Secrets                SyncAllResource `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Pods                   SyncPods        `yaml:"pods,omitempty" json:"pods,omitempty"`
}

type SyncFromHost struct {
	CSIDrivers           EnableSwitch `yaml:"csiDrivers,omitempty" json:"csiDrivers,omitempty"`
	CSINodes             EnableSwitch `yaml:"csiNodes,omitempty" json:"csiNodes,omitempty"`
	CSIStorageCapacities EnableSwitch `yaml:"csiStorageCapacities,omitempty" json:"csiStorageCapacities,omitempty"`
	IngressClasses       EnableSwitch `yaml:"ingressClasses,omitempty" json:"ingressClasses,omitempty"`
	Events               EnableSwitch `yaml:"events,omitempty" json:"events,omitempty"`
	StorageClasses       EnableSwitch `yaml:"storageClasses,omitempty" json:"storageClasses,omitempty"`
	Nodes                SyncNodes    `yaml:"nodes,omitempty" json:"nodes,omitempty"`
}

type EnableSwitch struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

type SyncAllResource struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	All bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

type SyncPods struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	WorkloadServiceAccount string            `yaml:"workloadServiceAccount,omitempty" json:"workloadServiceAccount,omitempty"`
	TranslateImage         map[string]string `yaml:"translateImage,omitempty" json:"translateImage,omitempty"`
	EnforceTolerations     []string          `yaml:"enforceTolerations,omitempty" json:"enforceTolerations,omitempty"` // validate format
	UseSecretsForSATokens  bool              `yaml:"useSecretsForSATokens,omitempty" json:"useSecretsForSATokens,omitempty"`
	RewriteHosts           *SyncRewriteHosts `yaml:"rewriteHosts,omitempty" json:"rewriteHosts,omitempty"`
}

type SyncRewriteHosts struct {
	Enabled            bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	InitContainerImage bool `yaml:"initContainerImage,omitempty" json:"initContainerImage,omitempty"`
}

type SyncNodes struct {
	Real   SyncRealNodes `yaml:"real,omitempty" json:"real,omitempty"`
	Pseudo EnableSwitch  `yaml:"pseudo,omitempty" json:"pseudo,omitempty"`
}

type SyncRealNodes struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	SyncLabelsTaints bool             `yaml:"syncLabelsTaints,omitempty" json:"syncLabelsTaints,omitempty"`
	ClearImageStatus bool             `yaml:"clearImageStatus,omitempty" json:"clearImageStatus,omitempty"`
	Selector         SyncNodeSelector `yaml:"selector,omitempty" json:"selector,omitempty"`
}

type SyncNodeSelector struct {
	Label map[string]string `yaml:"label,omitempty" json:"label,omitempty"`
}

type Observability struct {
	ServiceMonitor EnableSwitch         `yaml:"serviceMonitor,omitempty" json:"serviceMonitor,omitempty"`
	Metrics        ObservabilityMetrics `yaml:"metrics,omitempty" json:"metrics,omitempty"`
}

type ObservabilityMetrics struct {
	Proxy MetricsProxy `yaml:"proxy,omitempty" json:"proxy,omitempty"`
}

type MetricsProxy struct {
	Nodes EnableSwitch `yaml:"nodes,omitempty" json:"nodes,omitempty"`
	Pods  EnableSwitch `yaml:"pods,omitempty" json:"pods,omitempty"`
}

type Networking struct {
	ReplicateServices ReplicateServices  `yaml:"replicateServices,omitempty" json:"replicateServices,omitempty"`
	ResolveServices   ResolveServices    `yaml:"resolveServices,omitempty" json:"resolveServices,omitempty"`
	Advanced          NetworkingAdvanced `yaml:"advanced,omitempty" json:"advanced,omitempty"`
}

type ReplicateServices struct {
	ToHost   ServiceMapping `yaml:"toHost,omitempty" json:"toHost,omitempty"`
	FromHost ServiceMapping `yaml:"fromHost,omitempty" json:"fromHost,omitempty"`
}

type ServiceMapping struct {
	From string `yaml:"from,omitempty" json:"from,omitempty"`
	To   string `yaml:"to,omitempty" json:"to,omitempty"`
}

type ResolveServices struct {
	Service string               `yaml:"service,omitempty" json:"service,omitempty"`
	Target  ResolveServiceTarget `yaml:"target,omitempty" json:"target,omitempty"`
}

type ResolveServiceTarget struct {
	VCluster ResolveServiceService  `yaml:"vcluster,omitempty" json:"vcluster,omitempty"`
	Host     ResolveServiceService  `yaml:"host,omitempty" json:"host,omitempty"`
	External ResolveServiceHostname `yaml:"external,omitempty" json:"external,omitempty"`
}

type ResolveServiceService struct {
	Service string `yaml:"service,omitempty" json:"service,omitempty"`
}

type ResolveServiceHostname struct {
	Hostname string `yaml:"hostname,omitempty" json:"hostname,omitempty"`
}

type NetworkingAdvanced struct {
	ClusterDomain string               `yaml:"clusterDomain,omitempty" json:"clusterDomain,omitempty"`
	FallBack      []NetworkDNSFallback `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	ProxyKubelets NetworkProxyKubelets `yaml:"proxyKubelets,omitempty" json:"proxyKubelets,omitempty"`
}

type NetworkDNSFallback struct {
	IP          string `yaml:"ip,omitempty" json:"ip,omitempty"`
	HostCluster bool   `yaml:"hostCluster,omitempty" json:"hostCluster,omitempty"`
}

type NetworkProxyKubelets struct {
	ByHostname bool `yaml:"byHostname,omitempty" json:"byHostname,omitempty"`
	ByIP       bool `yaml:"byIP,omitempty" json:"byIP,omitempty"`
}

type Plugins struct {
	Image  string                 `yaml:"image,omitempty" json:"image,omitempty"`
	Config map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	RBAC   PluginsRBAC            `yaml:"rbac,omitempty" json:"rbac,omitempty"`
}

type PluginsRBAC struct {
	Role        []RBACPolicyRule `yaml:"role,omitempty" json:"role,omitempty"`
	ClusterRole []RBACPolicyRule `yaml:"clusterRole,omitempty" json:"clusterRole,omitempty"`
}

type RBACPolicyRule struct {
	// Verbs is a list of Verbs that apply to ALL the ResourceKinds contained in this rule. '*' represents all verbs.
	Verbs []string `json:"verbs"`

	// APIGroups is the name of the APIGroup that contains the resources.  If multiple API groups are specified, any action requested against one of
	// the enumerated resources in any API group will be allowed. "" represents the core API group and "*" represents all API groups.
	APIGroups []string `json:"apiGroups,omitempty"`
	// Resources is a list of resources this rule applies to. '*' represents all resources.
	Resources []string `json:"resources,omitempty"`
	// ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.
	ResourceNames []string `json:"resourceNames,omitempty"`

	// NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path
	// Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding.
	// Rules can either apply to API resources (such as "pods" or "secrets") or non-resource URL paths (such as "/api"),  but not both.
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
}

type Plugin struct {
	Plugins `yaml:",inline" json:",inline"`

	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type ControlPlane struct {
	Distro           Distro                       `yaml:"distro,omitempty" json:"distro,omitempty"`
	HostPathMapper   HostPathMapper               `yaml:"hostPathMapper,omitempty" json:"hostPathMapper,omitempty"`
	CoreDNS          CoreDNS                      `yaml:"coredns,omitempty" json:"coredns,omitempty"`
	VirtualScheduler EnableSwitch                 `yaml:"virtualScheduler,omitempty" json:"virtualScheduler,omitempty" product:"pro"`
	Proxy            ControlPlaneProxy            `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Service          ControlPlaneService          `yaml:"service,omitempty" json:"service,omitempty"`
	Ingress          ControlPlaneIngress          `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	HighAvailability ControlPlaneHighAvailability `yaml:"highAvailability,omitempty" json:"highAvailability,omitempty"`
	Advanced         ControlPlaneAdvanced         `yaml:"advanced,omitempty" json:"advanced,omitempty"`
}

type Distro struct {
	K3S      DistroK3s      `yaml:"k3s,omitempty" json:"k3s,omitempty"`
	K8S      DistroK8s      `yaml:"k8s,omitempty" json:"k8s,omitempty"`
	K0S      DistroK0s      `yaml:"k0s,omitempty" json:"k0s,omitempty"`
	Advanced DistroAdvanced `yaml:"advanced,omitempty" json:"advanced,omitempty"`
}

type DistroK3s struct {
	EnableSwitch    `yaml:",inline" json:",inline"`
	DistroContainer `yaml:",inline" json:",inline"`
	Token           string       `yaml:"token,omitempty" json:"token,omitempty"`
	BackingStore    BackingStore `yaml:"backingStore,omitempty" json:"backingStore,omitempty"`
}

type DistroK8s struct {
	EnableSwitch      `yaml:",inline" json:",inline"`
	APIServer         DistroContainer `yaml:"apiServer,omitempty" json:"apiServer,omitempty"`
	ControllerManager DistroContainer `yaml:"controllerManager,omitempty" json:"controllerManager,omitempty"`
	Scheduler         DistroContainer `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
	BackingStore      BackingStore    `yaml:"backingStore,omitempty" json:"backingStore,omitempty"`
}

type DistroK0s struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	BackingStore BackingStore `yaml:"backingStore,omitempty" json:"backingStore,omitempty"`
}

type DistroContainer struct {
	Spec            map[string]interface{}   `yaml:"spec,omitempty" json:"spec,omitempty"`
	Image           string                   `yaml:"image,omitempty" json:"image,omitempty"`
	ImagePullPolicy string                   `yaml:"imagePullPolicy,omitempty" json:"imagePullPolicy,omitempty"`
	Command         []string                 `yaml:"command,omitempty" json:"command,omitempty"`
	Args            []string                 `yaml:"args,omitempty" json:"args,omitempty"`
	ExtraArgs       []string                 `yaml:"extraArgs,omitempty" json:"extraArgs,omitempty"`
	Env             []map[string]interface{} `yaml:"env,omitempty" json:"env,omitempty"`
}

// LocalObjectReference contains enough information to let you locate the
// referenced object inside the same namespace.
type LocalObjectReference struct {
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name,omitempty"`
}

type DistroAdvanced struct {
	Paths DistroPaths `yaml:"paths,omitempty" json:"paths,omitempty"`
}

type DistroPaths struct {
	KubeConfig          string `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty"`
	ServerCAKey         string `yaml:"serverCAKey,omitempty" json:"serverCAKey,omitempty"`
	ServerCACert        string `yaml:"serverCACert,omitempty" json:"serverCACert,omitempty"`
	ClientCACert        string `yaml:"clientCACert,omitempty" json:"clientCACert,omitempty"`
	RequestHeaderCACert string `yaml:"requestHeaderCACert,omitempty" json:"requestHeaderCACert,omitempty"`
}

type BackingStore struct {
	EmbeddedEtcd EmbeddedEtcd `yaml:"embeddedEtcd,omitempty" json:"embeddedEtcd,omitempty" product:"pro"`
	ExternalEtcd ExternalEtcd `yaml:"externalEtcd,omitempty" json:"externalEtcd,omitempty"`
}

type EmbeddedEtcd struct {
	EnableSwitch      `yaml:",inline" json:",inline"`
	MigrateFromSqlite bool `yaml:"migrateFromSqlite,omitempty" json:"migrateFromSqlite,omitempty"`
}

type ExternalEtcd struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Image        string                  `yaml:"image,omitempty" json:"image,omitempty"`
	Replicas     uint8                   `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Security     ControlPlaneSecurity    `yaml:"security,omitempty" json:"security,omitempty"`
	Scheduling   ControlPlaneScheduling  `yaml:"scheduling,omitempty" json:"scheduling,omitempty"`
	Persistence  ControlPlanePersistence `yaml:"persistence,omitempty" json:"persistence,omitempty"`
	Metadata     ExternalEtcdMetadata    `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ExternalEtcdMetadata struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	PodLabels            map[string]string `yaml:"podLabels,omitempty" json:"podLabels,omitempty"`
	PodAnnotations       map[string]string `yaml:"podAnnotations,omitempty" json:"podAnnotations,omitempty"`
}

type HostPathMapper struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Central      bool `yaml:"central,omitempty" json:"central,omitempty" product:"pro"`
}

type CoreDNS struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Embedded     bool              `yaml:"embedded,omitempty" json:"embedded,omitempty" product:"pro"`
	Service      CoreDNSService    `yaml:"service,omitempty" json:"service,omitempty"`
	Deployment   CoreDNSDeployment `yaml:"deployment,omitempty" json:"deployment,omitempty"`
}

type CoreDNSService struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	Spec                 map[string]interface{} `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type CoreDNSDeployment struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	Spec                 map[string]interface{} `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneProxy struct {
	BindAddress string               `yaml:"bindAddress,omitempty" json:"bindAddress,omitempty"`
	Port        int                  `yaml:"port,omitempty" json:"port,omitempty"`
	TLS         ControlPlaneProxyTLS `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type ControlPlaneProxyTLS struct {
	ExtraSANs []string `yaml:"extraSANs,omitempty" json:"extraSANs,omitempty"`
}

type ControlPlaneService struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`

	Name string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Spec map[string]interface{} `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneIngress struct {
	EnableSwitch         `yaml:",inline" json:",inline"`
	LabelsAndAnnotations `yaml:",inline" json:",inline"`

	Name string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Spec map[string]interface{} `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneHighAvailability struct {
	Replicas int32 `yaml:"replicas,omitempty" json:"replicas,omitempty"`
}

type ControlPlaneAdvanced struct {
	DefaultImageRegistry   string                             `yaml:"defaultImageRegistry,omitempty" json:"defaultImageRegistry,omitempty"`
	Image                  ImageRef                           `yaml:"image,omitempty" json:"image,omitempty"`
	Headless               bool                               `yaml:"headless,omitempty" json:"headless,omitempty"`
	Persistence            ControlPlanePersistence            `yaml:"persistence,omitempty" json:"persistence,omitempty"`
	Scheduling             ControlPlaneScheduling             `yaml:"scheduling,omitempty" json:"scheduling,omitempty"`
	ServiceAccounts        ControlPlaneServiceAccounts        `yaml:"serviceAccounts,omitempty" json:"serviceAccounts,omitempty"`
	WorkloadServiceAccount ControlPlaneWorkloadServiceAccount `yaml:"workloadServiceAccount,omitempty" json:"workloadServiceAccount,omitempty"`
	Probes                 ControlPlaneProbes                 `yaml:"probes,omitempty" json:"probes,omitempty"`
	Security               ControlPlaneSecurity               `yaml:"security,omitempty" json:"security,omitempty"`
	Metadata               ControlPlaneMetadata               `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ImageRef struct {
	Repository string `yaml:"repository" json:"repository"`
	Tag        string `yaml:"tag,omitempty" json:"tag,omitempty"`
}

type ControlPlanePersistence struct {
	EnableSwitch          `yaml:",inline" json:",inline"`
	RetentionPolicy       string        `yaml:"retentionPolicy,omitempty" json:"retentionPolicy,omitempty"`
	Size                  string        `yaml:"size,omitempty" json:"size,omitempty"`
	StorageClass          string        `yaml:"storageClass,omitempty" json:"storageClass,omitempty"`
	AddVolumeMounts       []VolumeMount `yaml:"addVolumeMounts,omitempty" json:"addVolumeMounts,omitempty"`
	OverwriteVolumeMounts []VolumeMount `yaml:"overwriteVolumeMounts,omitempty" json:"overwriteVolumeMounts,omitempty"`
}

// VolumeMount describes a mounting of a Volume within a container.
type VolumeMount struct {
	// This must match the Name of a Volume.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	// Defaults to false.
	ReadOnly bool `json:"readOnly,omitempty" protobuf:"varint,2,opt,name=readOnly"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `json:"mountPath" protobuf:"bytes,3,opt,name=mountPath"`
	// Path within the volume from which the container's volume should be mounted.
	// Defaults to "" (volume's root).
	SubPath string `json:"subPath,omitempty" protobuf:"bytes,4,opt,name=subPath"`
	// mountPropagation determines how mounts are propagated from the host
	// to container and the other way around.
	// When not set, MountPropagationNone is used.
	// This field is beta in 1.10.
	MountPropagation *string `json:"mountPropagation,omitempty" protobuf:"bytes,5,opt,name=mountPropagation,casttype=MountPropagationMode"`
	// Expanded path within the volume from which the container's volume should be mounted.
	// Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.
	// Defaults to "" (volume's root).
	// SubPathExpr and SubPath are mutually exclusive.
	SubPathExpr string `json:"subPathExpr,omitempty" protobuf:"bytes,6,opt,name=subPathExpr"`
}

type ControlPlaneScheduling struct {
	NodeSelector      map[string]interface{} `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`
	Affinity          map[string]interface{} `yaml:"affinity,omitempty" json:"affinity,omitempty"`
	Tolerations       map[string]interface{} `yaml:"tolerations,omitempty" json:"tolerations,omitempty"`
	PriorityClassName string                 `yaml:"priorityClassName,omitempty" json:"priorityClassName,omitempty"`
}

type ControlPlaneServiceAccounts struct {
	EnableSwitch     `yaml:",inline" json:",inline"`
	Name             string                 `yaml:"name,omitempty" json:"name,omitempty"`
	ImagePullSecrets []LocalObjectReference `yaml:"imagePullSecrets,omitempty" json:"imagePullSecrets,omitempty"`
}

type ControlPlaneWorkloadServiceAccount struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Name         string            `yaml:"name,omitempty" json:"name,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type ControlPlaneProbes struct {
	LivenessProbe  EnableSwitch `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe EnableSwitch `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
}

type ControlPlaneSecurity struct {
	PodSecurityContext       map[string]interface{}               `yaml:"podSecurityContext,omitempty" json:"podSecurityContext,omitempty"`
	ContainerSecurityContext ControlPlaneContainerSecurityContext `yaml:"containerSecurityContext,omitempty" json:"containerSecurityContext,omitempty"`
}

type ControlPlaneContainerSecurityContext struct {
	AllowPrivilegeEscalation bool                   `yaml:"allowPrivilegeEscalation,omitempty" json:"allowPrivilegeEscalation,omitempty"`
	Capabilities             map[string]interface{} `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	RunAsUser                int64                  `yaml:"runAsUser,omitempty" json:"runAsUser,omitempty"`
	RunAsGroup               int64                  `yaml:"runAsUser,omitempty" json:"runAsGroup,omitempty"`
}

type ControlPlaneMetadata struct {
	StatefulSet  LabelsAndAnnotations `yaml:"statefulSet,omitempty" json:"statefulSet,omitempty"`
	Pods         LabelsAndAnnotations `yaml:"pods,omitempty" json:"pods,omitempty"`
	AllResources LabelsAndAnnotations `yaml:"allResources,omitempty" json:"allResources,omitempty"`
}

type LabelsAndAnnotations struct {
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type Policies struct {
	PodSecurityStandard string           `yaml:"podSecurityStandard,omitempty" json:"podSecurityStandard,omitempty"`
	ResourceQuota       ResourceQuota    `yaml:"resourceQuota,omitempty" json:"resourceQuota,omitempty"`
	LimitRange          LimitRange       `yaml:"limitRange,omitempty" json:"limitRange,omitempty"`
	NetworkPolicy       NetworkPolicy    `yaml:"networkPolicy,omitempty" json:"networkPolicy,omitempty"`
	AdmissionControl    AdmissionControl `yaml:"admissionControl,omitempty" json:"admissionControl,omitempty" product:"pro"`
}

type ResourceQuota struct {
	EnableSwitch  `yaml:",inline" json:",inline"`
	Quota         map[string]string `yaml:"quota,omitempty" json:"quota,omitempty"`
	ScopeSelector ScopeSelector     `yaml:"scopeSelector,omitempty" json:"scopeSelector,omitempty"`
	Scopes        []string          `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

type ScopeSelector struct {
	MatchExpressions []LabelSelectorRequirement `yaml:"matchExpressions,omitempty" json:"matchExpressions,omitempty"`
}

type LabelSelectorRequirement struct {
	// key is the label key that the selector applies to.
	Key string `json:"key"`
	// operator represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists and DoesNotExist.
	Operator string `json:"operator"`
	// values is an array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. This array is replaced during a strategic
	// merge patch.
	Values []string `json:"values,omitempty"`
}

type LimitRange struct {
	EnableSwitch   `yaml:",inline" json:",inline"`
	Default        LimitRangeLimits `yaml:"default,omitempty" json:"default,omitempty"`
	DefaultRequest LimitRangeLimits `yaml:"defaultRequest,omitempty" json:"defaultRequest,omitempty"`
}

type LimitRangeLimits struct {
	EphemeralStorage string `yaml:"ephemeral-storage,omitempty" json:"ephemeral-storage,omitempty"`
	Memory           string `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPU              string `yaml:"cpu,omitempty" json:"cpu,omitempty"`
}

type NetworkPolicy struct {
	EnableSwitch        `yaml:",inline" json:",inline"`
	OutgoingConnections OutgoingConnections `yaml:"outgoingConnections,omitempty" json:"outgoingConnections,omitempty"`
}

type OutgoingConnections struct {
	IPBlock IPBlock `yaml:"ipBlock,omitempty" json:"ipBlock,omitempty"`
}

// IPBlock describes a particular CIDR (Ex. "192.168.1.0/24","2001:db8::/64") that is allowed
// to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs
// that should not be included within this rule.
type IPBlock struct {
	// cidr is a string representing the IPBlock
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64"
	CIDR string `json:"cidr"`

	// except is a slice of CIDRs that should not be included within an IPBlock
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64"
	// Except values will be rejected if they are outside the cidr range
	// +optional
	Except []string `json:"except,omitempty"`
}

type AdmissionControl struct {
	ValidatingWebhooks []interface{} `yaml:"validatingWebhooks,omitempty" json:"validatingWebhooks,omitempty"`
	MutatingWebhooks   []interface{} `yaml:"mutatingWebhooks,omitempty" json:"mutatingWebhooks,omitempty"`
}

type RBAC struct {
	ClusterRole RBACClusterRole `yaml:"clusterRole,omitempty" json:"clusterRole,omitempty"`
	RBACRole    RBACRole        `yaml:"role,omitempty" json:"role,omitempty"`
}

type RBACClusterRole struct {
	Create     bool        `yaml:"create,omitempty" json:"create,omitempty"`
	ExtraRules interface{} `yaml:"extraRules,omitempty" json:"extraRules,omitempty"`
}

type RBACRole struct {
	RBACClusterRole      `yaml:",inline" json:",inline"`
	ExcludedApiResources []string `yaml:"excludedApiResources,omitempty" json:"excludedApiResources,omitempty"`
}

type Telemetry struct {
	Disabled           bool   `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	InstanceCreator    string `yaml:"instanceCreators,omitempty" json:"instanceCreators,omitempty"`
	PlatformUserID     string `yaml:"platformUserID,omitempty" json:"platformUserID,omitempty"`
	PlatformInstanceID string `yaml:"platformInstanceID,omitempty" json:"platformInstanceID,omitempty"`
	MachineID          string `yaml:"machineID,omitempty" json:"machineID,omitempty"`
}

type Experimental struct {
	Extended map[string]interface{} `yaml:",inline" json:",inline"`

	ControlPlaneSettings ExperimentalControlPlaneSettings `yaml:"controlPlaneSettings,omitempty" json:"controlPlaneSettings,omitempty"`
	SyncSettings         ExperimentalSyncSettings         `yaml:"syncSettings,omitempty" json:"syncSettings,omitempty"`
	SyncPatches          SyncPatches                      `yaml:"syncPatches,omitempty" json:"syncPatches,omitempty" product:"pro"`
	GenericSync          SyncPatches                      `yaml:"genericSync,omitempty" json:"genericSync,omitempty"`
	Deploy               ExperimentalDeploy               `yaml:"deploy,omitempty" json:"deploy,omitempty"`
}

type ExperimentalControlPlaneSettings struct {
	RewriteKubernetesService bool `yaml:"rewriteKubernetesService,omitempty" json:"rewriteKubernetesService,omitempty"`
}

type ExperimentalSyncSettings struct {
	DisableSync bool                           `yaml:"disableSync,omitempty" json:"disableSync,omitempty"`
	Target      ExperimentalSyncSettingsTarget `yaml:"target,omitempty" json:"target,omitempty"`
}

type ExperimentalSyncSettingsTarget struct {
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

type ExperimentalDeploy struct {
	Manifests         string        `yaml:"manifests,omitempty" json:"manifests,omitempty"`
	ManifestsTemplate string        `yaml:"manifestsTemplate,omitempty" json:"manifestsTemplate,omitempty"`
	Helm              []interface{} `yaml:"helm,omitempty" json:"helm,omitempty"`
}

type Platform struct {
	Name    string             `yaml:"name,omitempty" json:"name,omitempty"`
	Owner   string             `yaml:"owner,omitempty" json:"owner,omitempty"`
	Project string             `yaml:"project,omitempty" json:"project,omitempty"`
	ApiKey  SecretKeyReference `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`
}

type SecretKeyReference struct {
	Key          string          `yaml:"key,omitempty" json:"key,omitempty"`
	KeySecretRef SecretReference `yaml:"keySecretRef,omitempty" json:"keySecretRef,omitempty"`
}

type Template struct {
	// Name is the name of the template used to populate the virtual cluster
	Name    string `yaml:"name,omitempty" json:"name,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type Access struct {
	Audit AccessAudit `yaml:"audit,omitempty" json:"audit,omitempty" product:"platform"`
}

type AccessAudit struct {
	Enabled bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Level   int               `yaml:"level,omitempty" json:"level,omitempty"`
	Policy  AccessAuditPolicy `yaml:"policy,omitempty" json:"policy,omitempty"`
}

type AccessAuditPolicy struct {
	Rules []AuditPolicyRule `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// AuditPolicyRule maps requests based off metadata to an audit Level.
// Requests must match the rules of every field (an intersection of rules).
type AuditPolicyRule struct {
	// The Level that requests matching this rule are recorded at.
	Level string `json:"level"`

	// The users (by authenticated user name) this rule applies to.
	// An empty list implies every user.
	Users []string `json:"users,omitempty"`
	// The user groups this rule applies to. A user is considered matching
	// if it is a member of any of the UserGroups.
	// An empty list implies every user group.
	UserGroups []string `json:"userGroups,omitempty"`

	// The verbs that match this rule.
	// An empty list implies every verb.
	Verbs []string `json:"verbs,omitempty"`

	// Rules can apply to API resources (such as "pods" or "secrets"),
	// non-resource URL paths (such as "/api"), or neither, but not both.
	// If neither is specified, the rule is treated as a default for all URLs.

	// Resources that this rule matches. An empty list implies all kinds in all API groups.
	Resources []AuditGroupResources `json:"resources,omitempty"`
	// Namespaces that this rule matches.
	// The empty string "" matches non-namespaced resources.
	// An empty list implies every namespace.
	Namespaces []string `json:"namespaces,omitempty"`

	// NonResourceURLs is a set of URL paths that should be audited.
	// `*`s are allowed, but only as the full, final step in the path.
	// Examples:
	// - `/metrics` - Log requests for apiserver metrics
	// - `/healthz*` - Log all health checks
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`

	// OmitStages is a list of stages for which no events are created. Note that this can also
	// be specified policy wide in which case the union of both are omitted.
	// An empty list means no restrictions will apply.
	OmitStages []string `json:"omitStages,omitempty"`

	// OmitManagedFields indicates whether to omit the managed fields of the request
	// and response bodies from being written to the API audit log.
	// - a value of 'true' will drop the managed fields from the API audit log
	// - a value of 'false' indicates that the managed fileds should be included
	//   in the API audit log
	// Note that the value, if specified, in this rule will override the global default
	// If a value is not specified then the global default specified in
	// Policy.OmitManagedFields will stand.
	OmitManagedFields *bool `json:"omitManagedFields,omitempty"`
}

// AuditGroupResources represents resource kinds in an API group.
type AuditGroupResources struct {
	// Group is the name of the API group that contains the resources.
	// The empty string represents the core API group.
	Group string `json:"group,omitempty"`
	// Resources is a list of resources this rule applies to.
	//
	// For example:
	// - `pods` matches pods.
	// - `pods/log` matches the log subresource of pods.
	// - `*` matches all resources and their subresources.
	// - `pods/*` matches all subresources of pods.
	// - `*/scale` matches all scale subresources.
	//
	// If wildcard is present, the validation rule will ensure resources do not
	// overlap with each other.
	//
	// An empty list implies all resources and subresources in this API groups apply.
	Resources []string `json:"resources,omitempty"`
	// ResourceNames is a list of resource instance names that the policy matches.
	// Using this field requires Resources to be specified.
	// An empty list implies that every instance of the resource is matched.
	ResourceNames []string `json:"resourceNames,omitempty"`
}

type SyncPatches struct {
	// Version is the config version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Exports syncs a resource from the virtual cluster to the host
	Exports []*Export `json:"export,omitempty" yaml:"export,omitempty"`

	// Imports syncs a resource from the host cluster to virtual cluster
	Imports []*Import `json:"import,omitempty" yaml:"import,omitempty"`

	// Hooks are hooks that can be used to inject custom patches before syncing
	Hooks *Hooks `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

type Hooks struct {
	// HostToVirtual is a hook that is executed before syncing from the host to the virtual cluster
	HostToVirtual []*Hook `json:"hostToVirtual,omitempty" yaml:"hostToVirtual,omitempty"`

	// VirtualToHost is a hook that is executed before syncing from the virtual to the host cluster
	VirtualToHost []*Hook `json:"virtualToHost,omitempty" yaml:"virtualToHost,omitempty"`
}

type Hook struct {
	TypeInformation

	// Verbs are the verbs that the hook should mutate
	Verbs []string `json:"verbs,omitempty" yaml:"verbs,omitempty"`

	// Patches are the patches to apply on the object to be synced
	Patches []*Patch `json:"patches,omitempty" yaml:"patches,omitempty"`
}

type Import struct {
	SyncBase `json:",inline" yaml:",inline"`
}

type SyncBase struct {
	TypeInformation `json:",inline" yaml:",inline"`

	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// ReplaceWhenInvalid determines if the controller should try to recreate the object
	// if there is a problem applying
	ReplaceWhenInvalid bool `json:"replaceOnConflict,omitempty" yaml:"replaceOnConflict,omitempty"`

	// Patches are the patches to apply on the virtual cluster objects
	// when syncing them from the host cluster
	Patches []*Patch `json:"patches,omitempty" yaml:"patches,omitempty"`

	// ReversePatches are the patches to apply to host cluster objects
	// after it has been synced to the virtual cluster
	ReversePatches []*Patch `json:"reversePatches,omitempty" yaml:"reversePatches,omitempty"`
}

type Export struct {
	SyncBase `json:",inline" yaml:",inline"`

	// Selector is a label selector to select the synced objects in the virtual cluster.
	// If empty, all objects will be synced.
	Selector *Selector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

type TypeInformation struct {
	// APIVersion of the object to sync
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	// Kind of the object to sync
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
}

type Selector struct {
	// LabelSelector are the labels to select the object from
	LabelSelector map[string]string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
}

type Patch struct {
	// Operation is the type of the patch
	Operation PatchType `json:"op,omitempty" yaml:"op,omitempty"`

	// FromPath is the path from the other object
	FromPath string `json:"fromPath,omitempty" yaml:"fromPath,omitempty"`

	// Path is the path of the patch
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// NamePath is the path to the name of a child resource within Path
	NamePath string `json:"namePath,omitempty" yaml:"namePath,omitempty"`

	// NamespacePath is path to the namespace of a child resource within Path
	NamespacePath string `json:"namespacePath,omitempty" yaml:"namespacePath,omitempty"`

	// Value is the new value to be set to the path
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`

	// Regex - is regular expresion used to identify the Name,
	// and optionally Namespace, parts of the field value that
	// will be replaced with the rewritten Name and/or Namespace
	Regex string `json:"regex,omitempty" yaml:"regex,omitempty"`
	//ParsedRegex *regexp.Regexp `json:"-"               yaml:"-"`

	// Conditions are conditions that must be true for
	// the patch to get executed
	Conditions []*PatchCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

	// Ignore determines if the path should be ignored if handled as a reverse patch
	Ignore *bool `json:"ignore,omitempty" yaml:"ignore,omitempty"`

	// Sync defines if a specialized syncer should be initialized using values
	// from the rewriteName operation as Secret/Configmap names to be synced
	Sync *PatchSync `json:"sync,omitempty" yaml:"sync,omitempty"`
}

type PatchType string

const (
	PatchTypeRewriteName                     PatchType = "rewriteName"
	PatchTypeRewriteLabelKey                 PatchType = "rewriteLabelKey"
	PatchTypeRewriteLabelSelector            PatchType = "rewriteLabelSelector"
	PatchTypeRewriteLabelExpressionsSelector PatchType = "rewriteLabelExpressionsSelector"

	PatchTypeCopyFromObject PatchType = "copyFromObject"
	PatchTypeAdd            PatchType = "add"
	PatchTypeReplace        PatchType = "replace"
	PatchTypeRemove         PatchType = "remove"
)

type PatchCondition struct {
	// Path is the path within the object to select
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// SubPath is the path below the selected object to select
	SubPath string `json:"subPath,omitempty" yaml:"subPath,omitempty"`

	// Equal is the value the path should be equal to
	Equal interface{} `json:"equal,omitempty" yaml:"equal,omitempty"`

	// NotEqual is the value the path should not be equal to
	NotEqual interface{} `json:"notEqual,omitempty" yaml:"notEqual,omitempty"`

	// Empty means that the path value should be empty or unset
	Empty *bool `json:"empty,omitempty" yaml:"empty,omitempty"`
}

type PatchSync struct {
	Secret    *bool `json:"secret,omitempty"    yaml:"secret,omitempty"`
	ConfigMap *bool `json:"configmap,omitempty" yaml:"configmap,omitempty"`
}
