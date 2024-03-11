package config

import "regexp"

type Config struct {
	ExportKubeConfig ExportKubeConfig   `json:"exportKubeConfig,omitempty"`
	ControlPlane     ControlPlane       `json:"controlPlane,omitempty"`
	Sync             Sync               `json:"sync,omitempty"`
	Observability    Observability      `json:"observability,omitempty"`
	Networking       Networking         `json:"networking,omitempty"`
	Plugin           map[string]Plugin  `json:"plugin,omitempty"`
	Plugins          map[string]Plugins `json:"plugins,omitempty"`
	Policies         Policies           `json:"policies,omitempty"`
	RBAC             RBAC               `json:"rbac,omitempty"`

	// Telemetry is the configuration related to telemetry gathered about vcluster usage.
	Telemetry    Telemetry    `json:"telemetry,omitempty"`
	Experimental Experimental `json:"experimental,omitempty"`
	Platform     Platform     `json:"platform,omitempty"`

	// legacy for compatibility
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
	Pro         bool   `json:"pro,omitempty"`
}

type ExportKubeConfig struct {
	Context string          `json:"context"`
	Server  string          `json:"server"`
	Secret  SecretReference `json:"secret,omitempty"`
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
	ToHost   SyncToHost   `json:"toHost,omitempty"`
	FromHost SyncFromHost `json:"fromHost,omitempty"`
}

type SyncToHost struct {
	Services               EnableSwitch    `json:"services,omitempty"`
	Endpoints              EnableSwitch    `json:"endpoints,omitempty"`
	Ingresses              EnableSwitch    `json:"ingresses,omitempty"`
	PriorityClasses        EnableSwitch    `json:"priorityClasses,omitempty"`
	NetworkPolicies        EnableSwitch    `json:"networkPolicies,omitempty"`
	VolumeSnapshots        EnableSwitch    `json:"volumeSnapshots,omitempty"`
	PodDisruptionBudgets   EnableSwitch    `json:"podDisruptionBudgets,omitempty"`
	ServiceAccounts        EnableSwitch    `json:"serviceAccounts,omitempty"`
	StorageClasses         EnableSwitch    `json:"storageClasses,omitempty"`
	PersistentVolumes      EnableSwitch    `json:"persistentVolumes,omitempty"`
	PersistentVolumeClaims EnableSwitch    `json:"persistentVolumeClaims,omitempty"`
	ConfigMaps             SyncAllResource `json:"configMaps,omitempty"`
	Secrets                SyncAllResource `json:"secrets,omitempty"`
	Pods                   SyncPods        `json:"pods,omitempty"`
}

type SyncFromHost struct {
	CSIDrivers           EnableSwitch `json:"csiDrivers,omitempty"`
	CSINodes             EnableSwitch `json:"csiNodes,omitempty"`
	CSIStorageCapacities EnableSwitch `json:"csiStorageCapacities,omitempty"`
	IngressClasses       EnableSwitch `json:"ingressClasses,omitempty"`
	Events               EnableSwitch `json:"events,omitempty"`
	StorageClasses       EnableSwitch `json:"storageClasses,omitempty"`
	Nodes                SyncNodes    `json:"nodes,omitempty"`
}

type EnableSwitch struct {
	Enabled bool `json:"enabled,omitempty"`
}

type SyncAllResource struct {
	Enabled bool `json:"enabled,omitempty"`
	All     bool `json:"all,omitempty"`
}

type SyncPods struct {
	Enabled bool `json:"enabled,omitempty"`

	TranslateImage        map[string]string `json:"translateImage,omitempty"`
	EnforceTolerations    []string          `json:"enforceTolerations,omitempty"` // validate format
	UseSecretsForSATokens bool              `json:"useSecretsForSATokens,omitempty"`
	RewriteHosts          SyncRewriteHosts  `json:"rewriteHosts,omitempty"`
}

type SyncRewriteHosts struct {
	Enabled            bool   `json:"enabled,omitempty"`
	InitContainerImage string `json:"initContainerImage,omitempty"`
}

type SyncNodes struct {
	Real   SyncRealNodes `json:"real,omitempty"`
	Pseudo EnableSwitch  `json:"pseudo,omitempty"`
}

type SyncRealNodes struct {
	Enabled bool `json:"enabled,omitempty"`

	SyncAll          bool             `json:"syncAll,omitempty"`
	SyncLabelsTaints bool             `json:"syncLabelsTaints,omitempty"`
	ClearImageStatus bool             `json:"clearImageStatus,omitempty"`
	Selector         SyncNodeSelector `json:"selector,omitempty"`
}

type SyncNodeSelector struct {
	Labels map[string]string `json:"labels,omitempty"`
}

type Observability struct {
	Metrics ObservabilityMetrics `json:"metrics,omitempty"`
}

type ControlPlaneObservability struct {
	ServiceMonitor EnableSwitch `json:"serviceMonitor,omitempty"`
}

type ObservabilityMetrics struct {
	Proxy MetricsProxy `json:"proxy,omitempty"`
}

type MetricsProxy struct {
	Nodes EnableSwitch `json:"nodes,omitempty"`
	Pods  EnableSwitch `json:"pods,omitempty"`
}

type Networking struct {
	ReplicateServices ReplicateServices  `json:"replicateServices,omitempty"`
	ResolveServices   []ResolveServices  `json:"resolveServices,omitempty"`
	Advanced          NetworkingAdvanced `json:"advanced,omitempty"`
}

type ReplicateServices struct {
	ToHost   []ServiceMapping `json:"toHost,omitempty"`
	FromHost []ServiceMapping `json:"fromHost,omitempty"`
}

type ServiceMapping struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type ResolveServices struct {
	Service string               `json:"service,omitempty"`
	Target  ResolveServiceTarget `json:"target,omitempty"`
}

type ResolveServiceTarget struct {
	VCluster ResolveServiceService  `json:"vcluster,omitempty"`
	Host     ResolveServiceService  `json:"host,omitempty"`
	External ResolveServiceHostname `json:"external,omitempty"`
}

type ResolveServiceService struct {
	Service string `json:"service,omitempty"`
}

type ResolveServiceHostname struct {
	Hostname string `json:"hostname,omitempty"`
}

type NetworkingAdvanced struct {
	ClusterDomain string               `json:"clusterDomain,omitempty"`
	FallBack      []NetworkDNSFallback `json:"fallback,omitempty"`
	ProxyKubelets NetworkProxyKubelets `json:"proxyKubelets,omitempty"`
}

type NetworkDNSFallback struct {
	IP          string `json:"ip,omitempty"`
	HostCluster bool   `json:"hostCluster,omitempty"`
}

type NetworkProxyKubelets struct {
	ByHostname bool `json:"byHostname,omitempty"`
	ByIP       bool `json:"byIP,omitempty"`
}

type Plugin struct {
	Plugins `json:",inline"`

	Version        string                 `json:"version,omitempty"`
	Env            []interface{}          `json:"env,omitempty"`
	EnvFrom        []interface{}          `json:"envFrom,omitempty"`
	Lifecycle      map[string]interface{} `json:"lifecycle,omitempty"`
	LivenessProbe  map[string]interface{} `json:"livenessProbe,omitempty"`
	ReadinessProbe map[string]interface{} `json:"readinessProbe,omitempty"`
	StartupProbe   map[string]interface{} `json:"startupProbe,omitempty"`
	WorkingDir     string                 `json:"workingDir,omitempty"`
	Optional       bool                   `json:"optional,omitempty"`
}

type Plugins struct {
	Name            string                 `json:"name,omitempty"`
	Command         []string               `json:"command,omitempty"`
	Args            []string               `json:"args,omitempty"`
	Image           string                 `json:"image,omitempty"`
	ImagePullPolicy string                 `json:"imagePullPolicy,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	SecurityContext map[string]interface{} `json:"securityContext,omitempty"`
	Resources       map[string]interface{} `json:"resources,omitempty"`
	VolumeMounts    []interface{}          `json:"volumeMounts,omitempty"`
	RBAC            PluginsRBAC            `json:"rbac,omitempty"`
}

type PluginsRBAC struct {
	Role        []RBACPolicyRule `json:"role,omitempty"`
	ClusterRole []RBACPolicyRule `json:"clusterRole,omitempty"`
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

type ControlPlane struct {
	Distro           Distro                    `json:"distro,omitempty"`
	HostPathMapper   HostPathMapper            `json:"hostPathMapper,omitempty"`
	CoreDNS          CoreDNS                   `json:"coredns,omitempty"`
	BackingStore     BackingStore              `json:"backingStore,omitempty"`
	VirtualScheduler EnableSwitch              `json:"virtualScheduler,omitempty" product:"pro"`
	Proxy            ControlPlaneProxy         `json:"proxy,omitempty"`
	Service          ControlPlaneService       `json:"service,omitempty"`
	Ingress          ControlPlaneIngress       `json:"ingress,omitempty"`
	StatefulSet      ControlPlaneStatefulSet   `json:"statefulSet,omitempty"`
	Observability    ControlPlaneObservability `json:"observability,omitempty"`
	Advanced         ControlPlaneAdvanced      `json:"advanced,omitempty"`
}

type ControlPlaneStatefulSet struct {
	Image            Image                        `json:"image,omitempty"`
	ImagePullPolicy  string                       `json:"imagePullPolicy,omitempty"`
	Probes           ControlPlaneProbes           `json:"probes,omitempty"`
	Security         ControlPlaneSecurity         `json:"security,omitempty"`
	Persistence      ControlPlanePersistence      `json:"persistence,omitempty"`
	Scheduling       ControlPlaneScheduling       `json:"scheduling,omitempty"`
	HighAvailability ControlPlaneHighAvailability `json:"highAvailability,omitempty"`
}

type Distro struct {
	K3S DistroK3s `json:"k3s,omitempty"`
	K8S DistroK8s `json:"k8s,omitempty"`
	K0S DistroK0s `json:"k0s,omitempty"`
	EKS DistroK8s `json:"eks,omitempty"`
}

type DistroK3s struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Token           string `json:"token,omitempty"`
	DistroCommon    `json:",inline"`
	DistroContainer `json:",inline"`
}

type DistroK8s struct {
	Enabled           bool `json:"enabled,omitempty"`
	DistroCommon      `json:",inline"`
	APIServer         DistroContainerDisabled `json:"apiServer,omitempty"`
	ControllerManager DistroContainerDisabled `json:"controllerManager,omitempty"`
	Scheduler         DistroContainer         `json:"scheduler,omitempty"`
}

type DistroK0s struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Config          string `json:"config,omitempty"`
	DistroCommon    `json:",inline"`
	DistroContainer `json:",inline"`
}

type DistroCommon struct {
	Env             []map[string]interface{} `json:"env,omitempty"`
	SecurityContext map[string]interface{}   `json:"securityContext,omitempty"`
	Resources       map[string]interface{}   `json:"resources,omitempty"`
}

type DistroContainer struct {
	Image           Image    `json:"image,omitempty"`
	ImagePullPolicy string   `json:"imagePullPolicy,omitempty"`
	Command         []string `json:"command,omitempty"`
	ExtraArgs       []string `json:"extraArgs,omitempty"`
}

type DistroContainerDisabled struct {
	Disabled        bool     `json:"disabled,omitempty"`
	Image           Image    `json:"image,omitempty"`
	ImagePullPolicy string   `json:"imagePullPolicy,omitempty"`
	Command         []string `json:"command,omitempty"`
	ExtraArgs       []string `json:"extraArgs,omitempty"`
}

type Image struct {
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// LocalObjectReference contains enough information to let you locate the
// referenced object inside the same namespace.
type LocalObjectReference struct {
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name,omitempty"`
}

type VirtualClusterKubeConfig struct {
	KubeConfig          string `json:"kubeConfig,omitempty"`
	ServerCAKey         string `json:"serverCAKey,omitempty"`
	ServerCACert        string `json:"serverCACert,omitempty"`
	ClientCACert        string `json:"clientCACert,omitempty"`
	RequestHeaderCACert string `json:"requestHeaderCACert,omitempty"`
}

type BackingStore struct {
	EmbeddedEtcd EmbeddedEtcd `json:"embeddedEtcd,omitempty" product:"pro"`
	ExternalEtcd ExternalEtcd `json:"externalEtcd,omitempty"`
}

type EmbeddedEtcd struct {
	Enabled                 bool `json:"enabled,omitempty"`
	MigrateFromExternalEtcd bool `json:"migrateFromExternalEtcd,omitempty"`
}

type ExternalEtcd struct {
	Enabled     bool                    `json:"enabled,omitempty"`
	Image       string                  `json:"image,omitempty"`
	Replicas    uint8                   `json:"replicas,omitempty"`
	Security    ControlPlaneSecurity    `json:"security,omitempty"`
	Scheduling  ControlPlaneScheduling  `json:"scheduling,omitempty"`
	Persistence ControlPlanePersistence `json:"persistence,omitempty"`
	Metadata    ExternalEtcdMetadata    `json:"metadata,omitempty"`
}

type ExternalEtcdMetadata struct {
	LabelsAndAnnotations `json:",inline"`
	PodLabels            map[string]string `json:"podLabels,omitempty"`
	PodAnnotations       map[string]string `json:"podAnnotations,omitempty"`
}

type HostPathMapper struct {
	Enabled bool `json:"enabled,omitempty"`
	Central bool `json:"central,omitempty" product:"pro"`
}

type CoreDNS struct {
	Enabled    bool              `json:"enabled,omitempty"`
	Embedded   bool              `json:"embedded,omitempty" product:"pro"`
	Service    CoreDNSService    `json:"service,omitempty"`
	Deployment CoreDNSDeployment `json:"deployment,omitempty"`
}

type CoreDNSService struct {
	LabelsAndAnnotations `json:",inline"`
	Spec                 map[string]interface{} `json:"spec,omitempty"`
}

type CoreDNSDeployment struct {
	LabelsAndAnnotations `json:",inline"`
	Spec                 map[string]interface{} `json:"spec,omitempty"`
}

type ControlPlaneProxy struct {
	BindAddress string   `json:"bindAddress,omitempty"`
	Port        int      `json:"port,omitempty"`
	ExtraSANs   []string `json:"extraSANs,omitempty"`
}

type ControlPlaneService struct {
	LabelsAndAnnotations `json:",inline"`

	Name string                 `json:"name,omitempty"`
	Spec map[string]interface{} `json:"spec,omitempty"`
}

type ControlPlaneIngress struct {
	EnableSwitch         `json:",inline"`
	LabelsAndAnnotations `json:",inline"`

	Name string                 `json:"name,omitempty"`
	Spec map[string]interface{} `json:"spec,omitempty"`
}

type ControlPlaneHighAvailability struct {
	Replicas      int32 `json:"replicas,omitempty"`
	LeaseDuration int   `json:"leaseDuration,omitempty"`
	RenewDeadline int   `json:"renewDeadline,omitempty"`
	RetryPeriod   int   `json:"retryPeriod,omitempty"`
}

type ControlPlaneAdvanced struct {
	DefaultImageRegistry   string                             `json:"defaultImageRegistry,omitempty"`
	ServiceAccount         ControlPlaneServiceAccount         `json:"serviceAccount,omitempty"`
	WorkloadServiceAccount ControlPlaneWorkloadServiceAccount `json:"workloadServiceAccount,omitempty"`
	Metadata               ControlPlaneMetadata               `json:"metadata,omitempty"`
}

type ControlPlanePersistence struct {
	EnableSwitch          `json:",inline"`
	RetentionPolicy       string        `json:"retentionPolicy,omitempty"`
	Size                  string        `json:"size,omitempty"`
	StorageClass          string        `json:"storageClass,omitempty"`
	AddVolumeMounts       []VolumeMount `json:"addVolumeMounts,omitempty"`
	OverwriteVolumeMounts []VolumeMount `json:"overwriteVolumeMounts,omitempty"`
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
	NodeSelector      map[string]interface{} `json:"nodeSelector,omitempty"`
	Affinity          map[string]interface{} `json:"affinity,omitempty"`
	Tolerations       []interface{}          `json:"tolerations,omitempty"`
	PriorityClassName string                 `json:"priorityClassName,omitempty"`
}

type ControlPlaneServiceAccount struct {
	Enabled          bool                   `json:"enabled,omitempty"`
	Name             string                 `json:"name,omitempty"`
	ImagePullSecrets []LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`
}

type ControlPlaneWorkloadServiceAccount struct {
	Enabled          bool                   `json:"enabled,omitempty"`
	Name             string                 `json:"name,omitempty"`
	ImagePullSecrets []LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`
}

type ControlPlaneProbes struct {
	LivenessProbe  EnableSwitch `json:"livenessProbe,omitempty"`
	ReadinessProbe EnableSwitch `json:"readinessProbe,omitempty"`
	StartupProbe   EnableSwitch `json:"startupProbe,omitempty"`
}

type ControlPlaneSecurity struct {
	PodSecurityContext       map[string]interface{}               `json:"podSecurityContext,omitempty"`
	ContainerSecurityContext ControlPlaneContainerSecurityContext `json:"containerSecurityContext,omitempty"`
}

type ControlPlaneContainerSecurityContext struct {
	AllowPrivilegeEscalation bool                   `json:"allowPrivilegeEscalation,omitempty"`
	Capabilities             map[string]interface{} `json:"capabilities,omitempty"`
	RunAsUser                int64                  `json:"runAsUser,omitempty"`
	RunAsGroup               int64                  `json:"runAsGroup,omitempty"`
}

type ControlPlaneMetadata struct {
	StatefulSet  LabelsAndAnnotations `json:"statefulSet,omitempty"`
	Pods         LabelsAndAnnotations `json:"pods,omitempty"`
	AllResources LabelsAndAnnotations `json:"allResources,omitempty"`
}

type LabelsAndAnnotations struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Policies struct {
	PodSecurityStandard     string                  `json:"podSecurityStandard,omitempty"`
	ResourceQuota           ResourceQuota           `json:"resourceQuota,omitempty"`
	LimitRange              LimitRange              `json:"limitRange,omitempty"`
	NetworkPolicy           NetworkPolicy           `json:"networkPolicy,omitempty"`
	CentralAdmissionControl CentralAdmissionControl `json:"centralAdmissionControl,omitempty" product:"pro"`
}

type ResourceQuota struct {
	EnableSwitch  `json:",inline"`
	Quota         map[string]string `json:"quota,omitempty"`
	ScopeSelector ScopeSelector     `json:"scopeSelector,omitempty"`
	Scopes        []string          `json:"scopes,omitempty"`
}

type ScopeSelector struct {
	MatchExpressions []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
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
	EnableSwitch   `json:",inline"`
	Default        LimitRangeLimits `json:"default,omitempty"`
	DefaultRequest LimitRangeLimits `json:"defaultRequest,omitempty"`
}

type LimitRangeLimits struct {
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`
	Memory           string `json:"memory,omitempty"`
	CPU              string `json:"cpu,omitempty"`
}

type NetworkPolicy struct {
	EnableSwitch        `json:",inline"`
	OutgoingConnections OutgoingConnections `json:"outgoingConnections,omitempty"`
}

type OutgoingConnections struct {
	IPBlock IPBlock `json:"ipBlock,omitempty"`
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

type CentralAdmissionControl struct {
	ValidatingWebhooks []interface{} `json:"validatingWebhooks,omitempty"`
	MutatingWebhooks   []interface{} `json:"mutatingWebhooks,omitempty"`
}

type RBAC struct {
	ClusterRole RBACClusterRole `json:"clusterRole,omitempty"`
	RBACRole    RBACRole        `json:"role,omitempty"`
}

type RBACClusterRole struct {
	Create     bool        `json:"create,omitempty"`
	ExtraRules interface{} `json:"extraRules,omitempty"`
}

type RBACRole struct {
	RBACClusterRole      `json:",inline"`
	ExcludedApiResources []string `json:"excludedApiResources,omitempty"`
}

type Telemetry struct {
	Disabled           bool   `json:"disabled,omitempty"`
	InstanceCreator    string `json:"instanceCreators,omitempty"`
	PlatformUserID     string `json:"platformUserID,omitempty"`
	PlatformInstanceID string `json:"platformInstanceID,omitempty"`
	MachineID          string `json:"machineID,omitempty"`
}

type Experimental struct {
	Extended map[string]interface{} `json:",inline"`

	IsolatedControlPlane ExperimentalIsolatedControlPlane `json:"isolatedControlPlane,omitempty"`
	SyncSettings         ExperimentalSyncSettings         `json:"syncSettings,omitempty"`
	GenericSync          ExperimentalGenericSync          `json:"genericSync,omitempty"`
	Deploy               ExperimentalDeploy               `json:"deploy,omitempty"`
	MultiNamespaceMode   ExperimentalMultiNamespaceMode   `json:"multiNamespaceMode,omitempty"`

	VirtualClusterKubeConfig VirtualClusterKubeConfig `json:"virtualClusterKubeConfig,omitempty"`
}

type ExperimentalMultiNamespaceMode struct {
	Enabled bool `json:"enabled,omitempty"`

	NamespaceLabels map[string]string `json:"namespaceLabels,omitempty"`
}

type ExperimentalIsolatedControlPlane struct {
	Enabled bool `json:"enabled,omitempty"`

	KubeConfig string `json:"kubeConfig,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Service    string `json:"service,omitempty"`
}

type ExperimentalSyncSettings struct {
	DisableSync              bool `json:"disableSync,omitempty"`
	RewriteKubernetesService bool `json:"rewriteKubernetesService,omitempty"`

	TargetNamespace string   `json:"targetNamespace,omitempty"`
	SetOwner        bool     `json:"setOwner,omitempty"`
	SyncLabels      []string `json:"syncLabels,omitempty"`
}

type ExperimentalDeploy struct {
	Manifests         string        `json:"manifests,omitempty"`
	ManifestsTemplate string        `json:"manifestsTemplate,omitempty"`
	Helm              []interface{} `json:"helm,omitempty"`
}

type Platform struct {
	Name    string             `json:"name,omitempty"`
	Owner   string             `json:"owner,omitempty"`
	Project string             `json:"project,omitempty"`
	ApiKey  SecretKeyReference `json:"apiKey,omitempty"`
}

type SecretKeyReference struct {
	Value     string          `json:"value,omitempty"`
	SecretRef SecretReference `json:"secretRef,omitempty"`
}

type Template struct {
	// Name is the name of the template used to populate the virtual cluster
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type Access struct {
	Audit AccessAudit `json:"audit,omitempty" product:"platform"`
}

type AccessAudit struct {
	Enabled bool              `json:"enabled,omitempty"`
	Level   int               `json:"level,omitempty"`
	Policy  AccessAuditPolicy `json:"policy,omitempty"`
}

type AccessAuditPolicy struct {
	Rules []AuditPolicyRule `json:"rules,omitempty"`
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

type ExperimentalGenericSync struct {
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
	Regex       string         `json:"regex,omitempty" yaml:"regex,omitempty"`
	ParsedRegex *regexp.Regexp `json:"-"               yaml:"-"`

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
