package config

import (
	"github.com/creasty/defaults"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

type Config struct {
	ExportKubeConfig *ExportKubeConfig  `yaml:"exportKubeConfig,omitempty" json:"exportKubeConfig,omitempty"`
	Sync             *SyncConfig        `yaml:"sync,omitempty" json:"sync,omitempty"`
	Observability    *Observability     `yaml:"observability,omitempty" json:"observability,omitempty"`
	Networking       *Networking        `yaml:"networking,omitempty" json:"networking,omitempty"`
	Plugin           map[string]Plugin  `yaml:"plugin,omitempty" json:"plugin,omitempty"`
	Plugins          map[string]Plugins `yaml:"plugins,omitempty" json:"plugins,omitempty"`
	ControlPlane     ControlPlane       `yaml:"controlPlane,omitempty" json:"controlPlane,omitempty"`
	Policies         Policies           `yaml:"policies,omitempty" json:"policies,omitempty"`
	RBAC             RBAC               `yaml:"rbac,omitempty" json:"rbac,omitempty"`
	// Telemetry is the configuration related to telemetry gathered about vcluster usage.
	Telemetry    Telemetry    `yaml:"telemetry,omitempty" json:"telemetry,omitempty"`
	Experimental Experimental `yaml:"experimental,omitempty" json:"experimental,omitempty"`
	License      License      `yaml:"license,omitempty" json:"license,omitempty"`
	Platform     Platform     `yaml:"platform,omitempty" json:"platform,omitempty"`
	Sleep        Sleep        `yaml:"sleep,omitempty" json:"sleep,omitempty" product:"platform"`
	Template     Template     `yaml:"template,omitempty" json:"template,omitempty" product:"platform"`
	Access       Access       `yaml:"access,omitempty" json:"access,omitempty"`
}

type ExportKubeConfig struct {
	Context string                  `yaml:"context" json:"context"`
	Server  string                  `yaml:"server" json:"server"`
	Secret  *corev1.SecretReference `yaml:"secret,omitempty" json:"secret,omitempty"`
}

type SyncConfig struct {
	ToHost   *SyncToHost
	FromHost *SyncFromHost
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
}

type EnableSwitch struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"true"`
}

type SyncAllResource struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	All bool `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"true"`
}

type SyncPods struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	WorkloadServiceAccount string            `yaml:"workloadServiceAccount,omitempty" json:"workloadServiceAccount,omitempty"`
	TranslateImage         map[string]string `yaml:"translateImage,omitempty" json:"translateImage,omitempty"`
	EnforceTolerations     []string          `yaml:"enforceTolerations,omitempty" json:"enforceTolerations,omitempty"` // validate format
	UseSecretsForSATokens  bool              `yaml:"useSecretsForSATokens,omitempty" json:"useSecretsForSATokens,omitempty" default:"true"`
	RewriteHosts           *SyncRewriteHosts `yaml:"rewriteHosts,omitempty" json:"rewriteHosts,omitempty"`
}

type SyncRewriteHosts struct {
	Enabled            bool `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"true"`
	InitContainerImage bool `yaml:"initContainerImage,omitempty" json:"initContainerImage,omitempty" default:"true"`
}

type SyncNodes struct {
	Real   *SyncRealNodes   `yaml:"real,omitempty" json:"real,omitempty"`
	Pseudo *SyncPseudoNodes `yaml:"pseudo,omitempty" json:"pseudo,omitempty"`
}

type SyncRealNodes struct {
	EnableSwitch `yaml:",inline" json:",inline"`

	SyncLabelsTaints bool              `yaml:"syncLabelsTaints,omitempty" json:"syncLabelsTaints,omitempty"`
	ClearImageStatus bool              `yaml:"clearImageStatus,omitempty" json:"clearImageStatus,omitempty"`
	Selector         map[string]string `yaml:"selector,omitempty" json:"selector,omitempty"`
}

type SyncPseudoNodes struct {
	EnableSwitch `yaml:",inline" json:",inline"`
}

type Observability struct {
	ServiceMonitor EnableSwitch         `yaml:"serviceMonitor,omitempty" json:"serviceMonitor,omitempty"`
	Metrics        ObservabilityMetrics `yaml:"metrics,omitempty" json:"metrics,omitempty"`
}

type ObservabilityMetrics struct {
	Proxy ObservabilityMetricsProxy `yaml:"proxy,omitempty" json:"proxy,omitempty"`
}

type ObservabilityMetricsProxy struct {
	Nodes EnableSwitch `yaml:"nodes,omitempty" json:"nodes,omitempty"`
	Pods  EnableSwitch `yaml:"pods,omitempty" json:"pods,omitempty"`
	// TODO Docs, Logs
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
	ClusterDomain string                `yaml:"clusterDomain,omitempty" json:"clusterDomain,omitempty"`
	FallBack      []NetworkDNSFallback  `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	ProxyKubelets *NetworkProxyKubelets `yaml:"proxyKubelets,omitempty" json:"proxyKubelets,omitempty"`
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
	RBAC   *PluginsRBAC           `yaml:"rbac,omitempty" json:"rbac,omitempty"`
}

type PluginsRBAC struct {
	Role        []rbacv1.PolicyRule `yaml:"role,omitempty" json:"role,omitempty"`
	ClusterRole []rbacv1.PolicyRule `yaml:"clusterRole,omitempty" json:"clusterRole,omitempty"`
}

type Plugin struct {
	Plugins `yaml:",inline" json:",inline"`

	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

type ControlPlane struct {
	Distro           Distributions                `yaml:"distro,omitempty" json:"distro,omitempty"`
	BackingStore     BackingStore                 `yaml:"backingStore,omitempty" json:"backingStore,omitempty"`
	HostPathMapper   HostPathMapper               `yaml:"hostPathMapper,omitempty" json:"hostPathMapper,omitempty"`
	CoreDNS          CoreDNS                      `yaml:"coredns,omitempty" json:"coredns,omitempty"`
	VirtualScheduler *EnableSwitch                `yaml:"virtualScheduler,omitempty" json:"virtualScheduler,omitempty" product:"pro"`
	Proxy            *ControlPlaneProxy           `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	Service          ControlPlaneService          `yaml:"service,omitempty" json:"service,omitempty"`
	Ingress          ControlPlaneIngress          `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	HighAvailability ControlPlaneHighAvailability `yaml:"highAvailability,omitempty" json:"highAvailability,omitempty"`
	Advanced         ControlPlaneAdvanced         `yaml:"advanced,omitempty" json:"advanced,omitempty"`
}

type Distributions struct {
	K3S      *DistributionsK3s    `yaml:"k3s,omitempty" json:"k3s,omitempty"`
	K8S      *DistributionsK8s    `yaml:"k8s,omitempty" json:"k8s,omitempty"`
	K0S      *corev1.Container    `yaml:"k0s,omitempty" json:"k0s,omitempty"`
	Advanced DistributionAdvanced `yaml:"advanced,omitempty" json:"advanced,omitempty"`
}

type DistributionsK3s struct {
	corev1.Container `yaml:",inline" json:",inline"`
	Token            string `yaml:"token,omitempty" json:"token,omitempty"`
}

func (d *DistributionsK3s) SetDefaults() {
	if defaults.CanUpdate(d.Container.Resources) {
		d.Container.Resources = corev1.ResourceRequirements{
			Limits: map[corev1.ResourceName]resource.Quantity{
				"cpu":    {Format: "100m"},
				"memory": {Format: "256Mi"},
			},
			Requests: map[corev1.ResourceName]resource.Quantity{
				"cpu":    {Format: "100m"},
				"memory": {Format: "256Mi"},
			},
		}
	}
}

type DistributionsK8s struct {
	APIServer         *corev1.Container `yaml:"apiServer,omitempty" json:"apiServer,omitempty"`
	ControllerManager *corev1.Container `yaml:"controllerManager,omitempty" json:"controllerManager,omitempty"`
	Scheduler         *corev1.Container `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`
	ETCD              *corev1.Container `yaml:"etcd,omitempty" json:"etcd,omitempty"`
}

type DistributionAdvanced struct {
	Paths DistributionPaths `yaml:"paths,omitempty" json:"paths,omitempty"`
}
type DistributionPaths struct {
	KubeConfig          string `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty" default:"/data/server/cred/admin.kubeconfig"`
	ServerCAKey         string `yaml:"serverCAKey,omitempty" json:"serverCAKey,omitempty" default:"/data/server/tls/server-ca.key"`
	ServerCACert        string `yaml:"serverCACert,omitempty" json:"serverCACert,omitempty" default:"/data/server/tls/server-ca.crt"`
	ClientCACert        string `yaml:"clientCACert,omitempty" json:"clientCACert,omitempty" default:"/data/server/tls/client-ca.crt"`
	RequestHeaderCACert string `yaml:"requestHeaderCACert,omitempty" json:"requestHeaderCACert,omitempty" default:"/data/server/tls/request-header-ca.crt"`
}

type BackingStore struct {
	SQLite       EnableSwitch  `yaml:"sqlite,omitempty" json:"sqlite,omitempty"`
	EmbeddedEtcd *EmbeddedEtcd `yaml:"embeddedEtcd,omitempty" json:"embeddedEtcd,omitempty" product:"pro"`
	ExternalEtcd *ExternalEtcd `yaml:"externalEtcd,omitempty" json:"externalEtcd,omitempty"`
}

type EmbeddedEtcd struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	MigrateFrom  EmbeddedEtcdMigrateFrom `yaml:"migrateFrom,omitempty" json:"migrateFrom,omitempty"`
}

type EmbeddedEtcdMigrateFrom struct {
	SQLite bool `yaml:"sqlite,omitempty" json:"sqlite,omitempty" default:"true"`
	Etcd   bool `yaml:"etcd,omitempty" json:"etcd,omitempty" default:"true"`
}

type ExternalEtcd struct {
	Image    string                `yaml:"image,omitempty" json:"image,omitempty" default:"TBD"`
	Replicas uint8                 `yaml:"replicas,omitempty" json:"replicas,omitempty" default:"1"`
	Metadata *ExternalEtcdMetadata `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ExternalEtcdMetadata struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	PodLabels            map[string]string `yaml:"podLabels,omitempty" json:"podLabels,omitempty"`
	PodAnnotations       map[string]string `yaml:"podAnnotations,omitempty" json:"podAnnotations,omitempty"`
}

type HostPathMapper struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Central      bool `yaml:"central,omitempty" json:"central,omitempty" product:"pro" default:"true"`
}

type CoreDNS struct {
	EnableSwitch `yaml:",inline" json:",inline"`
	Embedded     bool              `yaml:"embedded,omitempty" json:"embedded,omitempty" product:"pro" default:"false"`
	Service      CoreDNSService    `yaml:"service,omitempty" json:"service,omitempty"`
	Deployment   CoreDNSDeployment `yaml:"deployment,omitempty" json:"deployment,omitempty"`
}

type CoreDNSService struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	Spec                 *corev1.ServiceSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type CoreDNSDeployment struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`
	Spec                 *appsv1.DeploymentSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneProxy struct {
	BindAddress string               `yaml:"bindAddress,omitempty" json:"bindAddress,omitempty" default:"0.0.0.0" validate:"ip4_addr"`
	Port        int                  `yaml:"port,omitempty" json:"port,omitempty" default:"8443"`
	TLS         ControlPlaneProxyTLS `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type ControlPlaneProxyTLS struct {
	ExtraSANs []string `yaml:"extraSANs,omitempty" json:"extraSANs,omitempty"`
}

type ControlPlaneService struct {
	LabelsAndAnnotations `yaml:",inline" json:",inline"`

	Name string              `yaml:"name,omitempty" json:"name,omitempty"`
	Spec *corev1.ServiceSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneIngress struct {
	EnableSwitch         `yaml:",inline" json:",inline"`
	LabelsAndAnnotations `yaml:",inline" json:",inline"`

	Name string                    `yaml:"name,omitempty" json:"name,omitempty"`
	Spec *networkingv1.IngressSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

type ControlPlaneHighAvailability struct {
	Replicas      *int32 `yaml:"replicas,omitempty" json:"replicas,omitempty" default:"1"`
	LeaseDuration *int32 `yaml:"leaseDuration,omitempty" json:"leaseDuration,omitempty" default:"66"`
	RenewDeadline *int32 `yaml:"renewDeadline,omitempty" json:"renewDeadline,omitempty" default:"40"`
	RetryPeriod   *int32 `yaml:"retryPeriod,omitempty" json:"retryPeriod,omitempty" default:"15"`
}

type ControlPlaneAdvanced struct {
	DefaultImageRegistry   string                             `yaml:"defaultImageRegistry,omitempty" json:"defaultImageRegistry,omitempty"`
	Headless               bool                               `yaml:"headless,omitempty" json:"headless,omitempty" default:"false"`
	Persistence            ControlPlanePersistence            `yaml:"persistence,omitempty" json:"persistence,omitempty"`
	Scheduling             ControlPlaneScheduling             `yaml:"scheduling,omitempty" json:"scheduling,omitempty"`
	ServiceAccounts        ControlPlaneServiceAccounts        `yaml:"serviceAccounts,omitempty" json:"serviceAccounts,omitempty"`
	WorkloadServiceAccount ControlPlaneWorkloadServiceAccount `yaml:"workloadServiceAccount,omitempty" json:"workloadServiceAccount,omitempty"`
	Probes                 ControlPlaneProbes                 `yaml:"probes,omitempty" json:"probes,omitempty"`
	Security               ControlPlaneSecurity               `yaml:"security,omitempty" json:"security,omitempty"`
	Metadata               ControlPlaneMetadata               `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ControlPlanePersistence struct {
	EnableSwitch          `yaml:",inline" json:",inline"`
	RetentionPolicy       string               `yaml:"retentionPolicy,omitempty" json:"retentionPolicy,omitempty" default:"Retain" validate:"oneof=Delete Retain"`
	Size                  *resource.Quantity   `yaml:"size,omitempty" json:"size,omitempty" default:"5Gi"`
	StorageClass          string               `yaml:"storageClass,omitempty" json:"storageClass,omitempty"`
	AddVolumeMounts       []corev1.VolumeMount `yaml:"addVolumeMounts,omitempty" json:"addVolumeMounts,omitempty"`
	OverwriteVolumeMounts []corev1.VolumeMount `yaml:"overwriteVolumeMounts,omitempty" json:"overwriteVolumeMounts,omitempty"`
}

type ControlPlaneScheduling struct {
	NodeSelector      map[string]string   `yaml:"nodeSelector,omitempty" json:"nodeSelector,omitempty"`
	Affinity          *corev1.Affinity    `yaml:"affinity,omitempty" json:"affinity,omitempty"`
	Tolerations       []corev1.Toleration `yaml:"tolerations,omitempty" json:"tolerations,omitempty"`
	PriorityClassName string              `yaml:"priorityClassName,omitempty" json:"priorityClassName,omitempty"`
}

type ControlPlaneServiceAccounts struct {
	Create           bool                          `yaml:"create,omitempty" json:"create,omitempty" default:"true"`
	Name             string                        `yaml:"name,omitempty" json:"name,omitempty" default:"default"`
	ImagePullSecrets []corev1.LocalObjectReference `yaml:"imagePullSecrets,omitempty" json:"imagePullSecrets,omitempty"`
}

type ControlPlaneWorkloadServiceAccount struct {
	Create      bool              `yaml:"create,omitempty" json:"create,omitempty" default:"true"`
	Name        string            `yaml:"name,omitempty" json:"name,omitempty" default:"default"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type ControlPlaneProbes struct {
	LivenessProbe  EnableSwitch `yaml:"livenessProbe,omitempty" json:"livenessProbe,omitempty"`
	ReadinessProbe EnableSwitch `yaml:"readinessProbe,omitempty" json:"readinessProbe,omitempty"`
}

type ControlPlaneSecurity struct {
	PodSecurityContext       *corev1.PodSecurityContext            `yaml:"podSecurityContext,omitempty" json:"podSecurityContext,omitempty"`
	ContainerSecurityContext *ControlPlaneContainerSecurityContext `yaml:"containerSecurityContext,omitempty" json:"containerSecurityContext,omitempty"`
}

type ControlPlaneContainerSecurityContext struct {
	AllowPrivilegeEscalation *bool                `yaml:"allowPrivilegeEscalation,omitempty" json:"allowPrivilegeEscalation,omitempty"`
	Capabilities             *corev1.Capabilities `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	RunAsUser                *int64               `yaml:"runAsUser,omitempty" json:"runAsUser,omitempty" default:"0"`
	RunAsGroup               *int64               `yaml:"runAsUser,omitempty" json:"runAsGroup,omitempty" default:"0"`
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
	PodSecurityStandard string           `yaml:"podSecurityStandard,omitempty" json:"podSecurityStandard,omitempty" default:"baseline"`
	ResourceQuota       ResourceQuota    `yaml:"resourceQuota,omitempty" json:"resourceQuota,omitempty"`
	LimitRange          LimitRange       `yaml:"limitRange,omitempty" json:"limitRange,omitempty"`
	NetworkPolicy       NetworkPolicy    `yaml:"networkPolicy,omitempty" json:"networkPolicy,omitempty"`
	AdmissionControl    AdmissionControl `yaml:"admissionControl,omitempty" json:"admissionControl,omitempty" product:"pro"`
}

type ResourceQuota struct {
	Enabled       bool                        `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"false"`
	Quota         corev1.ResourceList         `yaml:"quota,omitempty" json:"quota,omitempty"`
	ScopeSelector ScopeSelector               `yaml:"scopeSelector,omitempty" json:"scopeSelector,omitempty"`
	Scopes        []corev1.ResourceQuotaScope `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}

type ScopeSelector struct {
	MatchExpressions []metav1.LabelSelectorRequirement `yaml:"matchExpressions,omitempty" json:"matchExpressions,omitempty"`
}

type LimitRange struct {
	Enabled        bool              `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"false"`
	Default        *LimitRangeLimits `yaml:"default,omitempty" json:"default,omitempty"`
	DefaultRequest *LimitRangeLimits `yaml:"defaultRequest,omitempty" json:"defaultRequest,omitempty"`
}

func (l *LimitRange) SetDefaults() {
	if defaults.CanUpdate(l.Default) {
		l.Default = &LimitRangeLimits{
			EphemeralStorage: resource.Quantity{Format: "8Gi"},
			Memory:           resource.Quantity{Format: "512Mi"},
			CPU:              resource.Quantity{Format: "1"},
		}
	}

	if defaults.CanUpdate(l.DefaultRequest) {
		l.DefaultRequest = &LimitRangeLimits{
			EphemeralStorage: resource.Quantity{Format: "3Gi"},
			Memory:           resource.Quantity{Format: "128Mi"},
			CPU:              resource.Quantity{Format: "100m"},
		}
	}
}

type LimitRangeLimits struct {
	EphemeralStorage resource.Quantity `yaml:"ephemeral-storage,omitempty" json:"ephemeral-storage,omitempty"`
	Memory           resource.Quantity `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPU              resource.Quantity `yaml:"cpu,omitempty" json:"cpu,omitempty"`
}

type NetworkPolicy struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"false"`
}

type NetworkPolicyOutgoingConnections struct {
	IPBlock *networkingv1.IPBlock `yaml:"ipBlock,omitempty" json:"ipBlock,omitempty"`
}

func (c *NetworkPolicyOutgoingConnections) SetDefaults() {
	if defaults.CanUpdate(c.IPBlock) {
		c.IPBlock = &networkingv1.IPBlock{
			CIDR: "0.0.0.0/0",
			Except: []string{
				"100.64.0.0/10",
				"127.0.0.0/8",
				"10.0.0.0/8",
				"172.16.0.0/12",
				"192.168.0.0/16",
			},
		}
	}
}

type AdmissionControl struct {
	ValidatingWebhooks []admissionregistrationv1.ValidatingWebhook `yaml:"validatingWebhooks,omitempty" json:"validatingWebhooks,omitempty"`
	MutatingWebhooks   []admissionregistrationv1.MutatingWebhook   `yaml:"mutatingWebhooks,omitempty" json:"mutatingWebhooks,omitempty"`
}

type RBAC struct {
	ClusterRole RBACClusterRole `yaml:"clusterRole,omitempty" json:"clusterRole,omitempty"`
	RBACRole    RBACRole        `yaml:"role,omitempty" json:"role,omitempty"`
}

type RBACClusterRole struct {
	Create     bool        `yaml:"create,omitempty" json:"create,omitempty" default:"false"`
	ExtraRules interface{} `yaml:"extraRules,omitempty" json:"extraRules,omitempty"`
}

type RBACRole struct {
	RBACClusterRole      `yaml:",inline" json:",inline"`
	ExcludedApiResources []string `yaml:"excludedApiResources,omitempty" json:"excludedApiResources,omitempty"`
}

type Telemetry struct {
	Disabled           bool   `yaml:"disabled,omitempty" json:"disabled,omitempty" default:"false"`
	InstanceCreator    string `yaml:"instanceCreators,omitempty" json:"instanceCreators,omitempty" default:"helm"`
	PlatformUserID     string `yaml:"platformUserID,omitempty" json:"platformUserID,omitempty"`
	PlatformInstanceID string `yaml:"platformInstanceID,omitempty" json:"platformInstanceID,omitempty"`
	MachineID          string `yaml:"machineID,omitempty" json:"machineID,omitempty"`
}

type Experimental struct {
	Extended map[string]interface{} `yaml:",inline" json:",inline"`

	SyncSettings ExperimentalSyncSettings `yaml:"syncSettings,omitempty" json:"syncSettings,omitempty"`
	SyncPatches  SyncPatchesConfig        `yaml:"syncPatches,omitempty" json:"syncPatches,omitempty" product:"pro"`
	GenericSync  SyncPatchesConfig        `yaml:"genericSync,omitempty" json:"genericSync,omitempty"`
	Init         ExperimentalInit         `yaml:"init,omitempty" json:"init,omitempty"`
}

type ExperimentalSyncSettings struct {
	DisableSync bool                           `yaml:"disableSync,omitempty" json:"disableSync,omitempty" default:"false"`
	Target      ExperimentalSyncSettingsTarget `yaml:"target,omitempty" json:"target,omitempty"`
}

type ExperimentalSyncSettingsTarget struct {
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

type ExperimentalInit struct {
	Manifests         string                 `yaml:"manifests,omitempty" json:"manifests,omitempty"`
	ManifestsTemplate string                 `yaml:"manifestsTemplate,omitempty" json:"manifestsTemplate,omitempty"`
	Helm              []ExperimentalInitHelm `yaml:"helm,omitempty" json:"helm,omitempty"`
}

type ExperimentalInitHelm struct {
	// TODO pull properties from existing values.yaml
}

type License struct {
	Key          string                  `yaml:"key,omitempty" json:"key,omitempty"`
	KeySecretRef *corev1.SecretReference `yaml:"keySecretRef,omitempty" json:"keySecretRef,omitempty"`
}

func (l *License) SetDefaults() {
	if defaults.CanUpdate(l.KeySecretRef) {
		l.KeySecretRef = &corev1.SecretReference{
			Name: "vcluster-license",
		}
	}
}

type Platform struct {
	Name    string   `yaml:"name,omitempty" json:"name,omitempty"`
	Owner   string   `yaml:"owner,omitempty" json:"owner,omitempty"`
	Project string   `yaml:"project,omitempty" json:"project,omitempty"`
	ApiKey  *License `yaml:"apiKey,omitempty" json:"apiKey,omitempty"`
}

func (l *Platform) SetDefaults() {
	if defaults.CanUpdate(l.ApiKey) {
		l.ApiKey = &License{
			KeySecretRef: &corev1.SecretReference{
				Name: "vcluster-platform-api-key",
			},
		}
	}
}

type Sleep struct {
	After      int64           `yaml:"after,omitempty" json:"after,omitempty" default:"720"`
	Schedule   string          `yaml:"schedule,omitempty" json:"schedule,omitempty"` // TODO validate crontab
	Timezone   string          `yaml:"timezone,omitempty" json:"timezone,omitempty"` // TODO set default
	Ignore     SleepIgnore     `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	AutoDelete SleepAutoDelete `yaml:"autoDelete,omitempty" json:"autoDelete,omitempty"`
}

type SleepIgnore struct {
	Connections   SleepIgnoreConnections `yaml:"connections,omitempty" json:"connections,omitempty"`
	ResourcesWith map[string]string      `yaml:"resourcesWith,omitempty" json:"resourcesWith,omitempty"`
}

type SleepIgnoreConnections struct {
	OlderThan int64 `yaml:"after,omitempty" json:"after,omitempty" default:"3600"`
}

type SleepWakeup struct {
	Schedule string `yaml:"schedule,omitempty" json:"schedule,omitempty"` // TODO validate crontab
}

type SleepAutoDelete struct {
	After int64 `yaml:"after,omitempty" json:"after,omitempty" default:"720"`
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
	Enabled bool              `yaml:"enabled,omitempty" json:"enabled,omitempty" default:"true"`
	Level   int               `yaml:"level,omitempty" json:"level,omitempty" default:"1"` // TODO validate log levels
	Policy  AccessAuditPolicy `yaml:"policy,omitempty" json:"policy,omitempty"`
}

type AccessAuditPolicy struct {
	Rules []auditv1.PolicyRule `yaml:"rules,omitempty" json:"rules,omitempty"`
}
