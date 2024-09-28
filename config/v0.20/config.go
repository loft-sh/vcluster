package config

import (
	_ "embed"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/invopop/jsonschema"
	"sigs.k8s.io/yaml"
)

//go:embed values.yaml
var Values string

var ErrInvalidConfig = errors.New("invalid config")

// NewDefaultConfig creates a new config based on the values.yaml, including all default values.
func NewDefaultConfig() (*Config, error) {
	retConfig := &Config{}
	err := yaml.Unmarshal([]byte(Values), retConfig)
	if err != nil {
		return nil, err
	}

	return retConfig, nil
}

// Config is the vCluster config. This struct describes valid Helm values for vCluster as well as configuration used by the vCluster binary itself.
type Config struct {
	// Global values shared across all (sub)charts
	Global interface{} `json:"global,omitempty"`

	// ExportKubeConfig describes how vCluster should export the vCluster kubeConfig file.
	ExportKubeConfig ExportKubeConfig `json:"exportKubeConfig,omitempty"`

	// Sync describes how to sync resources from the virtual cluster to host cluster and back.
	Sync Sync `json:"sync,omitempty"`

	// Integrations holds config for vCluster integrations with other operators or tools running on the host cluster
	Integrations Integrations `json:"integrations,omitempty"`

	// Networking options related to the virtual cluster.
	Networking Networking `json:"networking,omitempty"`

	// Policies to enforce for the virtual cluster deployment as well as within the virtual cluster.
	Policies Policies `json:"policies,omitempty"`

	// Configure vCluster's control plane components and deployment.
	ControlPlane ControlPlane `json:"controlPlane,omitempty"`

	// RBAC options for the virtual cluster.
	RBAC RBAC `json:"rbac,omitempty"`

	// Define which vCluster plugins to load.
	Plugins map[string]Plugins `json:"plugins,omitempty"`

	// Experimental features for vCluster. Configuration here might change, so be careful with this.
	Experimental Experimental `json:"experimental,omitempty"`

	// External holds configuration for tools that are external to the vCluster.
	External map[string]ExternalConfig `json:"external,omitempty"`

	// Configuration related to telemetry gathered about vCluster usage.
	Telemetry Telemetry `json:"telemetry,omitempty"`

	// ServiceCIDR holds the service cidr for the virtual cluster. Do not use this option anymore.
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// Specifies whether to use vCluster Pro. This is automatically inferred in newer versions. Do not use that option anymore.
	Pro bool `json:"pro,omitempty"`

	// Plugin specifies which vCluster plugins to enable. Use "plugins" instead. Do not use this option anymore.
	Plugin map[string]Plugin `json:"plugin,omitempty"`
}

// Integrations holds config for vCluster integrations with other operators or tools running on the host cluster
type Integrations struct {
	// MetricsServer reuses the metrics server from the host cluster within the vCluster.
	MetricsServer MetricsServer `json:"metricsServer,omitempty"`
}

// MetricsServer reuses the metrics server from the host cluster within the vCluster.
type MetricsServer struct {
	// Enabled signals the metrics server integration should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// APIService holds information about where to find the metrics-server service. Defaults to metrics-server/kube-system.
	APIService APIService `json:"apiService,omitempty"`

	// Nodes defines if metrics-server nodes api should get proxied from host to virtual cluster.
	Nodes bool `json:"nodes,omitempty"`

	// Pods defines if metrics-server pods api should get proxied from host to virtual cluster.
	Pods bool `json:"pods,omitempty"`
}

// APIService holds configuration related to the api server
type APIService struct {
	// Service is a reference to the service for the API server.
	Service APIServiceService `json:"service,omitempty"`
}

// APIServiceService holds the service name and namespace of the host apiservice.
type APIServiceService struct {
	// Name is the name of the host service of the apiservice.
	Name string `json:"name,omitempty"`

	// Namespace is the name of the host service of the apiservice.
	Namespace string `json:"namespace,omitempty"`

	// Port is the target port on the host service to connect to.
	Port int `json:"port,omitempty"`
}

// ExternalConfig holds external tool configuration
type ExternalConfig map[string]interface{}

func (c *Config) UnmarshalYAMLStrict(data []byte) error {
	return UnmarshalYAMLStrict(data, c)
}

func (c *Config) GetPlatformConfig() (*PlatformConfig, error) {
	if c.External == nil {
		return &PlatformConfig{}, nil
	}
	if c.External["platform"] == nil {
		return &PlatformConfig{}, nil
	}

	yamlBytes, err := yaml.Marshal(c.External["platform"])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	retConfig := &PlatformConfig{}
	if err := yaml.Unmarshal(yamlBytes, retConfig); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	return retConfig, nil
}

func (c *Config) SetPlatformConfig(platformConfig *PlatformConfig) error {
	yamlBytes, err := yaml.Marshal(platformConfig)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	setConfig := ExternalConfig{}
	if err := yaml.Unmarshal(yamlBytes, &setConfig); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	if c.External == nil {
		c.External = map[string]ExternalConfig{}
	}
	c.External["platform"] = setConfig
	return nil
}

// BackingStoreType returns the backing store type of the vCluster.
// If no backing store is enabled, it returns StoreTypeUnknown.
func (c *Config) BackingStoreType() StoreType {
	switch {
	case c.ControlPlane.BackingStore.Etcd.Embedded.Enabled:
		return StoreTypeEmbeddedEtcd
	case c.ControlPlane.BackingStore.Etcd.Deploy.Enabled:
		return StoreTypeExternalEtcd
	case c.ControlPlane.BackingStore.Database.Embedded.Enabled:
		return StoreTypeEmbeddedDatabase
	case c.ControlPlane.BackingStore.Database.External.Enabled:
		return StoreTypeExternalDatabase
	default:
		return StoreTypeEmbeddedDatabase
	}
}

func (c *Config) EmbeddedDatabase() bool {
	return !c.ControlPlane.BackingStore.Database.External.Enabled && !c.ControlPlane.BackingStore.Etcd.Embedded.Enabled && !c.ControlPlane.BackingStore.Etcd.Deploy.Enabled
}

func (c *Config) Distro() string {
	if c.ControlPlane.Distro.K3S.Enabled {
		return K3SDistro
	} else if c.ControlPlane.Distro.K0S.Enabled {
		return K0SDistro
	} else if c.ControlPlane.Distro.K8S.Enabled {
		return K8SDistro
	}

	return K8SDistro
}

func (c *Config) IsConfiguredForSleepMode() bool {
	if c != nil && c.External != nil && c.External["platform"] == nil {
		return false
	}

	return c.External["platform"]["autoSleep"] != nil || c.External["platform"]["autoDelete"] != nil
}

// ValidateChanges checks for disallowed config changes.
// Currently only certain backingstore changes are allowed but no distro change.
func ValidateChanges(oldCfg, newCfg *Config) error {
	oldDistro, newDistro := oldCfg.Distro(), newCfg.Distro()
	oldBackingStore, newBackingStore := oldCfg.BackingStoreType(), newCfg.BackingStoreType()

	return ValidateStoreAndDistroChanges(newBackingStore, oldBackingStore, newDistro, oldDistro)
}

// ValidateStoreAndDistroChanges checks whether migrating from one store to the other is allowed.
func ValidateStoreAndDistroChanges(currentStoreType, previousStoreType StoreType, currentDistro, previousDistro string) error {
	if currentDistro != previousDistro && !(previousDistro == "eks" && currentDistro == K8SDistro) {
		return fmt.Errorf("seems like you were using %s as a distro before and now have switched to %s, please make sure to not switch between vCluster distros", previousDistro, currentDistro)
	}

	if currentStoreType != previousStoreType {
		if currentStoreType != StoreTypeEmbeddedEtcd {
			return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s, please make sure to not switch between vCluster stores", previousStoreType, currentStoreType)
		}
		if previousStoreType != StoreTypeExternalEtcd && previousStoreType != StoreTypeEmbeddedDatabase {
			return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s, please make sure to not switch between vCluster stores", previousStoreType, currentStoreType)
		}
	}

	return nil
}

func (c *Config) IsProFeatureEnabled() bool {
	if len(c.Networking.ResolveDNS) > 0 {
		return true
	}

	if c.ControlPlane.CoreDNS.Embedded {
		return true
	}

	if c.Distro() == K8SDistro {
		if c.ControlPlane.BackingStore.Database.External.Enabled {
			return true
		}
	}

	if c.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
		return true
	}

	if len(c.Policies.CentralAdmission.ValidatingWebhooks) > 0 || len(c.Policies.CentralAdmission.MutatingWebhooks) > 0 {
		return true
	}

	if c.ControlPlane.HostPathMapper.Central {
		return true
	}

	if c.Experimental.SyncSettings.DisableSync {
		return true
	}

	if c.Experimental.SyncSettings.RewriteKubernetesService {
		return true
	}

	if c.Experimental.IsolatedControlPlane.Enabled {
		return true
	}

	if len(c.Experimental.DenyProxyRequests) > 0 {
		return true
	}

	if len(c.External["platform"]) > 0 {
		return true
	}

	return false
}

func UnmarshalYAMLStrict(data []byte, i any) error {
	if err := yaml.UnmarshalStrict(data, i); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	return nil
}

// ExportKubeConfig describes how vCluster should export the vCluster kubeconfig.
type ExportKubeConfig struct {
	// Context is the name of the context within the generated kubeconfig to use.
	Context string `json:"context"`

	// Override the default https://localhost:8443 and specify a custom hostname for the generated kubeconfig.
	Server string `json:"server"`

	// Declare in which host cluster secret vCluster should store the generated virtual cluster kubeconfig.
	// If this is not defined, vCluster create it with `vc-NAME`. If you specify another name,
	// vCluster creates the config in this other secret.
	Secret ExportKubeConfigSecretReference `json:"secret,omitempty"`
}

// Declare in which host cluster secret vCluster should store the generated virtual cluster kubeconfig.
// If this is not defined, vCluster create it with `vc-NAME`. If you specify another name,
// vCluster creates the config in this other secret.
type ExportKubeConfigSecretReference struct {
	// Name is the name of the secret where the kubeconfig should get stored.
	Name string `json:"name,omitempty"`

	// Namespace where vCluster should store the kubeconfig secret. If this is not equal to the namespace
	// where you deployed vCluster, you need to make sure vCluster has access to this other namespace.
	Namespace string `json:"namespace,omitempty"`
}

type Sync struct {
	// Configure resources to sync from the virtual cluster to the host cluster.
	ToHost SyncToHost `json:"toHost,omitempty"`

	// Configure what resources vCluster should sync from the host cluster to the virtual cluster.
	FromHost SyncFromHost `json:"fromHost,omitempty"`
}

type SyncToHost struct {
	// Pods defines if pods created within the virtual cluster should get synced to the host cluster.
	Pods SyncPods `json:"pods,omitempty"`

	// Secrets defines if secrets created within the virtual cluster should get synced to the host cluster.
	Secrets SyncAllResource `json:"secrets,omitempty"`

	// ConfigMaps defines if config maps created within the virtual cluster should get synced to the host cluster.
	ConfigMaps SyncAllResource `json:"configMaps,omitempty"`

	// Ingresses defines if ingresses created within the virtual cluster should get synced to the host cluster.
	Ingresses EnableSwitch `json:"ingresses,omitempty"`

	// Services defines if services created within the virtual cluster should get synced to the host cluster.
	Services EnableSwitch `json:"services,omitempty"`

	// Endpoints defines if endpoints created within the virtual cluster should get synced to the host cluster.
	Endpoints EnableSwitch `json:"endpoints,omitempty"`

	// NetworkPolicies defines if network policies created within the virtual cluster should get synced to the host cluster.
	NetworkPolicies EnableSwitch `json:"networkPolicies,omitempty"`

	// PersistentVolumeClaims defines if persistent volume claims created within the virtual cluster should get synced to the host cluster.
	PersistentVolumeClaims EnableSwitch `json:"persistentVolumeClaims,omitempty"`

	// PersistentVolumes defines if persistent volumes created within the virtual cluster should get synced to the host cluster.
	PersistentVolumes EnableSwitch `json:"persistentVolumes,omitempty"`

	// VolumeSnapshots defines if volume snapshots created within the virtual cluster should get synced to the host cluster.
	VolumeSnapshots EnableSwitch `json:"volumeSnapshots,omitempty"`

	// StorageClasses defines if storage classes created within the virtual cluster should get synced to the host cluster.
	StorageClasses EnableSwitch `json:"storageClasses,omitempty"`

	// ServiceAccounts defines if service accounts created within the virtual cluster should get synced to the host cluster.
	ServiceAccounts EnableSwitch `json:"serviceAccounts,omitempty"`

	// PodDisruptionBudgets defines if pod disruption budgets created within the virtual cluster should get synced to the host cluster.
	PodDisruptionBudgets EnableSwitch `json:"podDisruptionBudgets,omitempty"`

	// PriorityClasses defines if priority classes created within the virtual cluster should get synced to the host cluster.
	PriorityClasses EnableSwitch `json:"priorityClasses,omitempty"`
}

type SyncFromHost struct {
	// Nodes defines if nodes should get synced from the host cluster to the virtual cluster, but not back.
	Nodes SyncNodes `json:"nodes,omitempty"`

	// Events defines if events should get synced from the host cluster to the virtual cluster, but not back.
	Events EnableSwitch `json:"events,omitempty"`

	// IngressClasses defines if ingress classes should get synced from the host cluster to the virtual cluster, but not back.
	IngressClasses EnableSwitch `json:"ingressClasses,omitempty"`

	// StorageClasses defines if storage classes should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	StorageClasses EnableAutoSwitch `json:"storageClasses,omitempty"`

	// CSINodes defines if csi nodes should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSINodes EnableAutoSwitch `json:"csiNodes,omitempty"`

	// CSIDrivers defines if csi drivers should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSIDrivers EnableAutoSwitch `json:"csiDrivers,omitempty"`

	// CSIStorageCapacities defines if csi storage capacities should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSIStorageCapacities EnableAutoSwitch `json:"csiStorageCapacities,omitempty"`
}

type EnableAutoSwitch struct {
	// Enabled defines if this option should be enabled.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`
}

type EnableSwitch struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty"`
}

type SyncAllResource struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// All defines if all resources of that type should get synced or only the necessary ones that are needed.
	All bool `json:"all,omitempty"`
}

type SyncPods struct {
	// Enabled defines if pod syncing should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// TranslateImage maps an image to another image that should be used instead. For example this can be used to rewrite
	// a certain image that is used within the virtual cluster to be another image on the host cluster
	TranslateImage map[string]string `json:"translateImage,omitempty"`

	// EnforceTolerations will add the specified tolerations to all pods synced by the virtual cluster.
	EnforceTolerations []string `json:"enforceTolerations,omitempty"`

	// UseSecretsForSATokens will use secrets to save the generated service account tokens by virtual cluster instead of using a
	// pod annotation.
	UseSecretsForSATokens bool `json:"useSecretsForSATokens,omitempty"`

	// RewriteHosts is a special option needed to rewrite statefulset containers to allow the correct FQDN. virtual cluster will add
	// a small container to each stateful set pod that will initially rewrite the /etc/hosts file to match the FQDN expected by
	// the virtual cluster.
	RewriteHosts SyncRewriteHosts `json:"rewriteHosts,omitempty"`
}

type SyncRewriteHosts struct {
	// Enabled specifies if rewriting stateful set pods should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// InitContainer holds extra options for the init container used by vCluster to rewrite the FQDN for stateful set pods.
	InitContainer SyncRewriteHostsInitContainer `json:"initContainer,omitempty"`
}

type SyncRewriteHostsInitContainer struct {
	// Image is the image virtual cluster should use to rewrite this FQDN.
	Image string `json:"image,omitempty"`

	// Resources are the resources that should be assigned to the init container for each stateful set init container.
	Resources Resources `json:"resources,omitempty"`
}

type SyncNodes struct {
	// Enabled specifies if syncing real nodes should be enabled. If this is disabled, vCluster will create fake nodes instead.
	Enabled bool `json:"enabled,omitempty"`

	// SyncBackChanges enables syncing labels and taints from the virtual cluster to the host cluster. If this is enabled someone within the virtual cluster will be able to change the labels and taints of the host cluster node.
	SyncBackChanges bool `json:"syncBackChanges,omitempty"`

	// ClearImageStatus will erase the image status when syncing a node. This allows to hide images that are pulled by the node.
	ClearImageStatus bool `json:"clearImageStatus,omitempty"`

	// Selector can be used to define more granular what nodes should get synced from the host cluster to the virtual cluster.
	Selector SyncNodeSelector `json:"selector,omitempty"`
}

type SyncNodeSelector struct {
	// All specifies if all nodes should get synced by vCluster from the host to the virtual cluster or only the ones where pods are assigned to.
	All bool `json:"all,omitempty"`

	// Labels are the node labels used to sync nodes from host cluster to virtual cluster. This will also set the node selector when syncing a pod from virtual cluster to host cluster to the same value.
	Labels map[string]string `json:"labels,omitempty"`
}

type ServiceMonitor struct {
	// Enabled configures if Helm should create the service monitor.
	Enabled bool `json:"enabled,omitempty"`

	// Labels are the extra labels to add to the service monitor.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the extra annotations to add to the service monitor.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Networking struct {
	// ReplicateServices allows replicating services from the host within the virtual cluster or the other way around.
	ReplicateServices ReplicateServices `json:"replicateServices,omitempty"`

	// ResolveDNS allows to define extra DNS rules. This only works if embedded coredns is configured.
	ResolveDNS []ResolveDNS `json:"resolveDNS,omitempty" product:"pro"`

	// Advanced holds advanced network options.
	Advanced NetworkingAdvanced `json:"advanced,omitempty"`
}

func (n Networking) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(n))
}

type ReplicateServices struct {
	// ToHost defines the services that should get synced from virtual cluster to the host cluster. If services are
	// synced to a different namespace than the virtual cluster is in, additional permissions for the other namespace
	// are required.
	ToHost []ServiceMapping `json:"toHost,omitempty"`

	// FromHost defines the services that should get synced from the host to the virtual cluster.
	FromHost []ServiceMapping `json:"fromHost,omitempty"`
}

type ServiceMapping struct {
	// From is the service that should get synced. Can be either in the form name or namespace/name.
	From string `json:"from,omitempty"`

	// To is the target service that it should get synced to. Can be either in the form name or namespace/name.
	To string `json:"to,omitempty"`
}

type ResolveDNS struct {
	// Hostname is the hostname within the vCluster that should be resolved from.
	Hostname string `json:"hostname"`

	// Service is the virtual cluster service that should be resolved from.
	Service string `json:"service"`

	// Namespace is the virtual cluster namespace that should be resolved from.
	Namespace string `json:"namespace"`

	// Target is the DNS target that should get mapped to
	Target ResolveDNSTarget `json:"target,omitempty"`
}

type ResolveDNSTarget struct {
	// Hostname to use as a DNS target
	Hostname string `json:"hostname,omitempty"`

	// IP to use as a DNS target
	IP string `json:"ip,omitempty"`

	// HostService to target, format is hostNamespace/hostService
	HostService string `json:"hostService,omitempty"`

	// HostNamespace to target
	HostNamespace string `json:"hostNamespace,omitempty"`

	// VClusterService format is hostNamespace/vClusterName/vClusterNamespace/vClusterService
	VClusterService string `json:"vClusterService,omitempty"`
}

type NetworkingAdvanced struct {
	// ClusterDomain is the Kubernetes cluster domain to use within the virtual cluster.
	ClusterDomain string `json:"clusterDomain,omitempty"`

	// FallbackHostCluster allows to fallback dns to the host cluster. This is useful if you want to reach host services without
	// any other modification. You will need to provide a namespace for the service, e.g. my-other-service.my-other-namespace
	FallbackHostCluster bool `json:"fallbackHostCluster,omitempty"`

	// ProxyKubelets allows rewriting certain metrics and stats from the Kubelet to "fake" this for applications such as
	// prometheus or other node exporters.
	ProxyKubelets NetworkProxyKubelets `json:"proxyKubelets,omitempty"`
}

type NetworkProxyKubelets struct {
	// ByHostname will add a special vCluster hostname to the nodes where the node can be reached at. This doesn't work
	// for all applications, e.g. Prometheus requires a node IP.
	ByHostname bool `json:"byHostname,omitempty"`

	// ByIP will create a separate service in the host cluster for every node that will point to virtual cluster and will be used to
	// route traffic.
	ByIP bool `json:"byIP,omitempty"`
}

type Plugin struct {
	Plugins `json:",inline"`

	// Version is the plugin version, this is only needed for legacy plugins.
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
	// Name is the name of the init-container and NOT the plugin name
	Name string `json:"name,omitempty"`

	// Image is the container image that should be used for the plugin
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy to use for the container image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Config is the plugin config to use. This can be arbitrary config used for the plugin.
	Config map[string]interface{} `json:"config,omitempty"`

	// RBAC holds additional rbac configuration for the plugin
	RBAC PluginsRBAC `json:"rbac,omitempty"`

	// Command is the command that should be used for the init container
	Command []string `json:"command,omitempty"`

	// Args are the arguments that should be used for the init container
	Args []string `json:"args,omitempty"`

	// SecurityContext is the container security context used for the init container
	SecurityContext map[string]interface{} `json:"securityContext,omitempty"`

	// Resources are the container resources used for the init container
	Resources map[string]interface{} `json:"resources,omitempty"`

	// VolumeMounts are extra volume mounts for the init container
	VolumeMounts []interface{} `json:"volumeMounts,omitempty"`
}

type PluginsRBAC struct {
	// Role holds extra virtual cluster role permissions for the plugin
	Role PluginsExtraRules `json:"role,omitempty"`

	// ClusterRole holds extra virtual cluster cluster role permissions required for the plugin
	ClusterRole PluginsExtraRules `json:"clusterRole,omitempty"`
}

type PluginsExtraRules struct {
	// ExtraRules are extra rbac permissions roles that will be added to role or cluster role
	ExtraRules []RBACPolicyRule `json:"extraRules,omitempty"`
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
	// Distro holds virtual cluster related distro options. A distro cannot be changed after vCluster is deployed.
	Distro Distro `json:"distro,omitempty"`

	// BackingStore defines which backing store to use for virtual cluster. If not defined will use embedded database as a default backing store.
	BackingStore BackingStore `json:"backingStore,omitempty"`

	// CoreDNS defines everything related to the coredns that is deployed and used within the vCluster.
	CoreDNS CoreDNS `json:"coredns,omitempty"`

	// Proxy defines options for the virtual cluster control plane proxy that is used to do authentication and intercept requests.
	Proxy ControlPlaneProxy `json:"proxy,omitempty"`

	// HostPathMapper defines if vCluster should rewrite host paths.
	HostPathMapper HostPathMapper `json:"hostPathMapper,omitempty" product:"pro"`

	// Ingress defines options for vCluster ingress deployed by Helm.
	Ingress ControlPlaneIngress `json:"ingress,omitempty"`

	// Service defines options for vCluster service deployed by Helm.
	Service ControlPlaneService `json:"service,omitempty"`

	// StatefulSet defines options for vCluster statefulSet deployed by Helm.
	StatefulSet ControlPlaneStatefulSet `json:"statefulSet,omitempty"`

	// ServiceMonitor can be used to automatically create a service monitor for vCluster deployment itself.
	ServiceMonitor ServiceMonitor `json:"serviceMonitor,omitempty"`

	// Advanced holds additional configuration for the vCluster control plane.
	Advanced ControlPlaneAdvanced `json:"advanced,omitempty"`
}

func (c ControlPlane) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(c))
}

type ControlPlaneStatefulSet struct {
	// HighAvailability holds options related to high availability.
	HighAvailability ControlPlaneHighAvailability `json:"highAvailability,omitempty"`

	// Resources are the resource requests and limits for the statefulSet container.
	Resources Resources `json:"resources,omitempty"`

	// Scheduling holds options related to scheduling.
	Scheduling ControlPlaneScheduling `json:"scheduling,omitempty"`

	// Security defines pod or container security context.
	Security ControlPlaneSecurity `json:"security,omitempty"`

	// Probes enables or disables the main container probes.
	Probes ControlPlaneProbes `json:"probes,omitempty"`

	// Persistence defines options around persistence for the statefulSet.
	Persistence ControlPlanePersistence `json:"persistence,omitempty"`

	// EnableServiceLinks for the StatefulSet pod
	EnableServiceLinks *bool `json:"enableServiceLinks,omitempty"`

	LabelsAndAnnotations `json:",inline"`

	// Additional labels or annotations for the statefulSet pods.
	Pods LabelsAndAnnotations `json:"pods,omitempty"`

	// Image is the image for the controlPlane statefulSet container
	Image StatefulSetImage `json:"image,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// WorkingDir specifies in what folder the main process should get started.
	WorkingDir string `json:"workingDir,omitempty"`

	// Command allows you to override the main command.
	Command []string `json:"command,omitempty"`

	// Args allows you to override the main arguments.
	Args []string `json:"args,omitempty"`

	// Env are additional environment variables for the statefulSet container.
	Env []map[string]interface{} `json:"env,omitempty"`
}

type Distro struct {
	// K8S holds K8s relevant configuration.
	K8S DistroK8s `json:"k8s,omitempty"`

	// K3S holds K3s relevant configuration.
	K3S DistroK3s `json:"k3s,omitempty"`

	// K0S holds k0s relevant configuration.
	K0S DistroK0s `json:"k0s,omitempty"`
}

type DistroK3s struct {
	// Enabled specifies if the K3s distro should be enabled. Only one distro can be enabled at the same time.
	Enabled bool `json:"enabled,omitempty"`

	// Token is the K3s token to use. If empty, vCluster will choose one.
	Token string `json:"token,omitempty"`

	DistroCommon    `json:",inline"`
	DistroContainer `json:",inline"`
}

type DistroK8s struct {
	// Enabled specifies if the K8s distro should be enabled. Only one distro can be enabled at the same time.
	Enabled bool `json:"enabled,omitempty"`

	// Version specifies k8s components (scheduler, kube-controller-manager & apiserver) version.
	// It is a shortcut for controlPlane.distro.k8s.apiServer.image.tag,
	// controlPlane.distro.k8s.controllerManager.image.tag and
	// controlPlane.distro.k8s.scheduler.image.tag
	// If e.g. controlPlane.distro.k8s.version is set to v1.30.1 and
	// controlPlane.distro.k8s.scheduler.image.tag
	//(or controlPlane.distro.k8s.controllerManager.image.tag or controlPlane.distro.k8s.apiServer.image.tag)
	// is set to v1.31.0,
	// value from controlPlane.distro.k8s.<controlPlane-component>.image.tag will be used
	// (where <controlPlane-component is apiServer, controllerManager and scheduler).
	Version string `json:"version,omitempty"`

	// APIServer holds configuration specific to starting the api server.
	APIServer DistroContainerEnabled `json:"apiServer,omitempty"`

	// ControllerManager holds configuration specific to starting the controller manager.
	ControllerManager DistroContainerEnabled `json:"controllerManager,omitempty"`

	// Scheduler holds configuration specific to starting the scheduler. Enable this via controlPlane.advanced.virtualScheduler.enabled
	Scheduler DistroContainer `json:"scheduler,omitempty"`

	DistroCommon `json:",inline"`
}

type DistroK0s struct {
	// Enabled specifies if the k0s distro should be enabled. Only one distro can be enabled at the same time.
	Enabled bool `json:"enabled,omitempty"`

	// Config allows you to override the k0s config passed to the k0s binary.
	Config string `json:"config,omitempty"`

	DistroCommon    `json:",inline"`
	DistroContainer `json:",inline"`
}

type DistroCommon struct {
	// Env are extra environment variables to use for the main container and NOT the init container.
	Env []map[string]interface{} `json:"env,omitempty"`

	// Resources for the distro init container
	Resources map[string]interface{} `json:"resources,omitempty"`

	// Security options can be used for the distro init container
	SecurityContext map[string]interface{} `json:"securityContext,omitempty"`
}

type DistroContainer struct {
	// Image is the distro image
	Image Image `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy for the distro image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Command is the command to start the distro binary. This will override the existing command.
	Command []string `json:"command,omitempty"`

	// ExtraArgs are additional arguments to pass to the distro binary.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type DistroContainerEnabled struct {
	// Enabled signals this container should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Image is the distro image
	Image Image `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy for the distro image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Command is the command to start the distro binary. This will override the existing command.
	Command []string `json:"command,omitempty"`

	// ExtraArgs are additional arguments to pass to the distro binary.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type StatefulSetImage struct {
	// Configure the registry of the container image, e.g. my-registry.com or ghcr.io
	// It defaults to ghcr.io and can be overriding either by using this field or controlPlane.advanced.defaultImageRegistry
	Registry string `json:"registry,omitempty"`

	// Configure the repository of the container image, e.g. my-repo/my-image.
	// It defaults to the vCluster pro repository that includes the optional pro modules that are turned off by default.
	// If you still want to use the pure OSS build, use 'loft-sh/vcluster-oss' instead.
	Repository string `json:"repository,omitempty"`

	// Tag is the tag of the container image, e.g. latest
	Tag string `json:"tag,omitempty"`
}

type Image struct {
	// Registry is the registry of the container image, e.g. my-registry.com or ghcr.io. This setting can be globally
	// overridden via the controlPlane.advanced.defaultImageRegistry option. Empty means docker hub.
	Registry string `json:"registry,omitempty"`

	// Repository is the repository of the container image, e.g. my-repo/my-image
	Repository string `json:"repository,omitempty"`

	// Tag is the tag of the container image, e.g. latest
	Tag string `json:"tag,omitempty"`
}

type ImagePullSecretName struct {
	// Name of the image pull secret to use.
	Name string `json:"name,omitempty"`
}

type VirtualClusterKubeConfig struct {
	// KubeConfig is the virtual cluster kubeconfig path.
	KubeConfig string `json:"kubeConfig,omitempty"`

	// ServerCAKey is the server ca key path.
	ServerCAKey string `json:"serverCAKey,omitempty"`

	// ServerCAKey is the server ca cert path.
	ServerCACert string `json:"serverCACert,omitempty"`

	// ServerCAKey is the client ca cert path.
	ClientCACert string `json:"clientCACert,omitempty"`

	// RequestHeaderCACert is the request header ca cert path.
	RequestHeaderCACert string `json:"requestHeaderCACert,omitempty"`
}

type BackingStore struct {
	// Etcd defines that etcd should be used as the backend for the virtual cluster
	Etcd Etcd `json:"etcd,omitempty"`

	// Database defines that a database backend should be used as the backend for the virtual cluster. This uses a project called kine under the hood which is a shim for bridging Kubernetes and relational databases.
	Database Database `json:"database,omitempty"`
}

type Database struct {
	// Embedded defines that an embedded database (sqlite) should be used as the backend for the virtual cluster
	Embedded DatabaseKine `json:"embedded,omitempty"`

	// External defines that an external database should be used as the backend for the virtual cluster
	External DatabaseKine `json:"external,omitempty"`
}

type DatabaseKine struct {
	// Enabled defines if the database should be used.
	Enabled bool `json:"enabled,omitempty"`

	// DataSource is the kine dataSource to use for the database. This depends on the database format.
	// This is optional for the embedded database. Examples:
	// * mysql: mysql://username:password@tcp(hostname:3306)/k3s
	// * postgres: postgres://username:password@hostname:5432/k3s
	DataSource string `json:"dataSource,omitempty"`

	// KeyFile is the key file to use for the database. This is optional.
	KeyFile string `json:"keyFile,omitempty"`

	// CertFile is the cert file to use for the database. This is optional.
	CertFile string `json:"certFile,omitempty"`

	// CaFile is the ca file to use for the database. This is optional.
	CaFile string `json:"caFile,omitempty"`
}

type Etcd struct {
	// Embedded defines to use embedded etcd as a storage backend for the virtual cluster
	Embedded EtcdEmbedded `json:"embedded,omitempty" product:"pro"`

	// Deploy defines to use an external etcd that is deployed by the helm chart
	Deploy EtcdDeploy `json:"deploy,omitempty"`
}

func (e Etcd) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type EtcdEmbedded struct {
	// Enabled defines if the embedded etcd should be used.
	Enabled bool `json:"enabled,omitempty" product:"pro"`

	// MigrateFromDeployedEtcd signals that vCluster should migrate from the deployed external etcd to embedded etcd.
	MigrateFromDeployedEtcd bool `json:"migrateFromDeployedEtcd,omitempty"`
}

func (e EtcdEmbedded) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type EtcdDeploy struct {
	// Enabled defines that an external etcd should be deployed.
	Enabled bool `json:"enabled,omitempty"`

	// StatefulSet holds options for the external etcd statefulSet.
	StatefulSet EtcdDeployStatefulSet `json:"statefulSet,omitempty"`

	// Service holds options for the external etcd service.
	Service EtcdDeployService `json:"service,omitempty"`

	// HeadlessService holds options for the external etcd headless service.
	HeadlessService EtcdDeployHeadlessService `json:"headlessService,omitempty"`
}

type EtcdDeployService struct {
	// Enabled defines if the etcd service should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// Annotations are extra annotations for the external etcd service
	Annotations map[string]string `json:"annotations,omitempty"`
}

type EtcdDeployHeadlessService struct {
	// Enabled defines if the etcd headless service should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// Annotations are extra annotations for the external etcd headless service
	Annotations map[string]string `json:"annotations,omitempty"`
}

type EtcdDeployStatefulSet struct {
	// Enabled defines if the statefulSet should be deployed
	Enabled bool `json:"enabled,omitempty"`

	// EnableServiceLinks for the StatefulSet pod
	EnableServiceLinks *bool `json:"enableServiceLinks,omitempty"`

	// Image is the image to use for the external etcd statefulSet
	Image Image `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy for the external etcd image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Env are extra environment variables
	Env []map[string]interface{} `json:"env,omitempty"`

	// ExtraArgs are appended to the etcd command.
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// Resources the etcd can consume
	Resources Resources `json:"resources,omitempty"`

	// Pods defines extra metadata for the etcd pods.
	Pods LabelsAndAnnotations `json:"pods,omitempty"`

	// HighAvailability are high availability options
	HighAvailability ExternalEtcdHighAvailability `json:"highAvailability,omitempty"`

	// Scheduling options for the etcd pods.
	Scheduling ControlPlaneScheduling `json:"scheduling,omitempty"`

	// Security options for the etcd pods.
	Security ControlPlaneSecurity `json:"security,omitempty"`

	// Persistence options for the etcd pods.
	Persistence ExternalEtcdPersistence `json:"persistence,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type Resources struct {
	// Limits are resource limits for the container
	Limits map[string]interface{} `json:"limits,omitempty"`

	// Requests are minimal resources that will be consumed by the container
	Requests map[string]interface{} `json:"requests,omitempty"`
}

type ExternalEtcdHighAvailability struct {
	// Replicas are the amount of pods to use.
	Replicas int `json:"replicas,omitempty"`
}

type HostPathMapper struct {
	// Enabled specifies if the host path mapper will be used
	Enabled bool `json:"enabled,omitempty"`

	// Central specifies if the central host path mapper will be used
	Central bool `json:"central,omitempty"`
}

type CoreDNS struct {
	// Enabled defines if coredns is enabled
	Enabled bool `json:"enabled,omitempty"`

	// Embedded defines if vCluster will start the embedded coredns service within the control-plane and not as a separate deployment. This is a PRO feature.
	Embedded bool `json:"embedded,omitempty" product:"pro"`

	// Service holds extra options for the coredns service deployed within the virtual cluster
	Service CoreDNSService `json:"service,omitempty"`

	// Deployment holds extra options for the coredns deployment deployed within the virtual cluster
	Deployment CoreDNSDeployment `json:"deployment,omitempty"`

	// OverwriteConfig can be used to overwrite the coredns config
	OverwriteConfig string `json:"overwriteConfig,omitempty"`

	// OverwriteManifests can be used to overwrite the coredns manifests used to deploy coredns
	OverwriteManifests string `json:"overwriteManifests,omitempty"`

	// PriorityClassName specifies the priority class name for the CoreDNS pods.
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

func (c CoreDNS) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(c))
}

type CoreDNSService struct {
	// Spec holds extra options for the coredns service
	Spec map[string]interface{} `json:"spec,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type CoreDNSDeployment struct {
	// Image is the coredns image to use
	Image string `json:"image,omitempty"`

	// Replicas is the amount of coredns pods to run.
	Replicas int `json:"replicas,omitempty"`

	// NodeSelector is the node selector to use for coredns.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Resources are the desired resources for coredns.
	Resources Resources `json:"resources,omitempty"`

	// Pods is additional metadata for the coredns pods.
	Pods LabelsAndAnnotations `json:"pods,omitempty"`

	LabelsAndAnnotations `json:",inline"`

	// TopologySpreadConstraints are the topology spread constraints for the CoreDNS pod.
	TopologySpreadConstraints []interface{} `json:"topologySpreadConstraints,omitempty"`
}

type ControlPlaneProxy struct {
	// BindAddress under which vCluster will expose the proxy.
	BindAddress string `json:"bindAddress,omitempty"`

	// Port under which vCluster will expose the proxy. Changing port is currently not supported.
	Port int `json:"port,omitempty"`

	// ExtraSANs are extra hostnames to sign the vCluster proxy certificate for.
	ExtraSANs []string `json:"extraSANs,omitempty"`
}

type ControlPlaneService struct {
	// Enabled defines if the control plane service should be enabled
	Enabled bool `json:"enabled,omitempty"`

	// Spec allows you to configure extra service options.
	Spec map[string]interface{} `json:"spec,omitempty"`

	// KubeletNodePort is the node port where the fake kubelet is exposed. Defaults to 0.
	KubeletNodePort int `json:"kubeletNodePort,omitempty"`

	// HTTPSNodePort is the node port where https is exposed. Defaults to 0.
	HTTPSNodePort int `json:"httpsNodePort,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type ControlPlaneIngress struct {
	// Enabled defines if the control plane ingress should be enabled
	Enabled bool `json:"enabled,omitempty"`

	// Host is the host where vCluster will be reachable
	Host string `json:"host,omitempty"`

	// PathType is the path type of the ingress
	PathType string `json:"pathType,omitempty"`

	// Spec allows you to configure extra ingress options.
	Spec map[string]interface{} `json:"spec,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type ControlPlaneHighAvailability struct {
	// Replicas is the amount of replicas to use for the statefulSet.
	Replicas int32 `json:"replicas,omitempty"`

	// LeaseDuration is the time to lease for the leader.
	LeaseDuration int `json:"leaseDuration,omitempty"`

	// RenewDeadline is the deadline to renew a lease for the leader.
	RenewDeadline int `json:"renewDeadline,omitempty"`

	// RetryPeriod is the time until a replica will retry to get a lease.
	RetryPeriod int `json:"retryPeriod,omitempty"`
}

type ControlPlaneAdvanced struct {
	// DefaultImageRegistry will be used as a prefix for all internal images deployed by vCluster or Helm. This makes it easy to
	// upload all required vCluster images to a single private repository and set this value. Workload images are not affected by this.
	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	// VirtualScheduler defines if a scheduler should be used within the virtual cluster or the scheduling decision for workloads will be made by the host cluster.
	VirtualScheduler EnableSwitch `json:"virtualScheduler,omitempty"`

	// ServiceAccount specifies options for the vCluster control plane service account.
	ServiceAccount ControlPlaneServiceAccount `json:"serviceAccount,omitempty"`

	// WorkloadServiceAccount specifies options for the service account that will be used for the workloads that run within the virtual cluster.
	WorkloadServiceAccount ControlPlaneWorkloadServiceAccount `json:"workloadServiceAccount,omitempty"`

	// HeadlessService specifies options for the headless service used for the vCluster StatefulSet.
	HeadlessService ControlPlaneHeadlessService `json:"headlessService,omitempty"`

	// GlobalMetadata is metadata that will be added to all resources deployed by Helm.
	GlobalMetadata ControlPlaneGlobalMetadata `json:"globalMetadata,omitempty"`
}

type ControlPlaneHeadlessService struct {
	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are extra labels for this resource.
	Labels map[string]string `json:"labels,omitempty"`
}

type ExternalEtcdPersistence struct {
	// VolumeClaim can be used to configure the persistent volume claim.
	VolumeClaim ExternalEtcdPersistenceVolumeClaim `json:"volumeClaim,omitempty"`

	// VolumeClaimTemplates defines the volumeClaimTemplates for the statefulSet
	VolumeClaimTemplates []map[string]interface{} `json:"volumeClaimTemplates,omitempty"`

	// AddVolumes defines extra volumes for the pod
	AddVolumes []map[string]interface{} `json:"addVolumes,omitempty"`

	// AddVolumeMounts defines extra volume mounts for the container
	AddVolumeMounts []VolumeMount `json:"addVolumeMounts,omitempty"`
}

type ExternalEtcdPersistenceVolumeClaim struct {
	// Enabled enables deploying a persistent volume claim.
	Enabled bool `json:"enabled,omitempty"`

	// AccessModes are the persistent volume claim access modes.
	AccessModes []string `json:"accessModes,omitempty"`

	// RetentionPolicy is the persistent volume claim retention policy.
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// Size is the persistent volume claim storage size.
	Size string `json:"size,omitempty"`

	// StorageClass is the persistent volume claim storage class.
	StorageClass string `json:"storageClass,omitempty"`
}

type ControlPlanePersistence struct {
	// VolumeClaim can be used to configure the persistent volume claim.
	VolumeClaim VolumeClaim `json:"volumeClaim,omitempty"`

	// VolumeClaimTemplates defines the volumeClaimTemplates for the statefulSet
	VolumeClaimTemplates []map[string]interface{} `json:"volumeClaimTemplates,omitempty"`

	// Allows you to override the dataVolume. Only works correctly if volumeClaim.enabled=false.
	DataVolume []map[string]interface{} `json:"dataVolume,omitempty"`

	// BinariesVolume defines a binaries volume that is used to retrieve
	// distro specific executables to be run by the syncer controller.
	// This volume doesn't need to be persistent.
	BinariesVolume []map[string]interface{} `json:"binariesVolume,omitempty"`

	// AddVolumes defines extra volumes for the pod
	AddVolumes []map[string]interface{} `json:"addVolumes,omitempty"`

	// AddVolumeMounts defines extra volume mounts for the container
	AddVolumeMounts []VolumeMount `json:"addVolumeMounts,omitempty"`
}

type VolumeClaim struct {
	// Enabled enables deploying a persistent volume claim. If auto, vCluster will automatically determine
	// based on the chosen distro and other options if this is required.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// AccessModes are the persistent volume claim access modes.
	AccessModes []string `json:"accessModes,omitempty"`

	// RetentionPolicy is the persistent volume claim retention policy.
	RetentionPolicy string `json:"retentionPolicy,omitempty"`

	// Size is the persistent volume claim storage size.
	Size string `json:"size,omitempty"`

	// StorageClass is the persistent volume claim storage class.
	StorageClass string `json:"storageClass,omitempty"`
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
	// NodeSelector is the node selector to apply to the pod.
	NodeSelector map[string]interface{} `json:"nodeSelector,omitempty"`

	// Affinity is the affinity to apply to the pod.
	Affinity map[string]interface{} `json:"affinity,omitempty"`

	// Tolerations are the tolerations to apply to the pod.
	Tolerations []map[string]interface{} `json:"tolerations,omitempty"`

	// PriorityClassName is the priority class name for the the pod.
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// PodManagementPolicy is the statefulSet pod management policy.
	PodManagementPolicy string `json:"podManagementPolicy,omitempty"`

	// TopologySpreadConstraints are the topology spread constraints for the pod.
	TopologySpreadConstraints []interface{} `json:"topologySpreadConstraints,omitempty"`
}

type ControlPlaneServiceAccount struct {
	// Enabled specifies if the service account should get deployed.
	Enabled bool `json:"enabled,omitempty"`

	// Name specifies what name to use for the service account.
	Name string `json:"name,omitempty"`

	// ImagePullSecrets defines extra image pull secrets for the service account.
	ImagePullSecrets []ImagePullSecretName `json:"imagePullSecrets,omitempty"`

	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are extra labels for this resource.
	Labels map[string]string `json:"labels,omitempty"`
}

type ControlPlaneWorkloadServiceAccount struct {
	// Enabled specifies if the service account for the workloads should get deployed.
	Enabled bool `json:"enabled,omitempty"`

	// Name specifies what name to use for the service account for the virtual cluster workloads.
	Name string `json:"name,omitempty"`

	// ImagePullSecrets defines extra image pull secrets for the workload service account.
	ImagePullSecrets []ImagePullSecretName `json:"imagePullSecrets,omitempty"`

	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are extra labels for this resource.
	Labels map[string]string `json:"labels,omitempty"`
}

type ControlPlaneProbes struct {
	// LivenessProbe specifies if the liveness probe for the container should be enabled
	LivenessProbe EnableSwitch `json:"livenessProbe,omitempty"`

	// ReadinessProbe specifies if the readiness probe for the container should be enabled
	ReadinessProbe EnableSwitch `json:"readinessProbe,omitempty"`

	// StartupProbe specifies if the startup probe for the container should be enabled
	StartupProbe EnableSwitch `json:"startupProbe,omitempty"`
}

type ControlPlaneSecurity struct {
	// PodSecurityContext specifies security context options on the pod level.
	PodSecurityContext map[string]interface{} `json:"podSecurityContext,omitempty"`

	// ContainerSecurityContext specifies security context options on the container level.
	ContainerSecurityContext map[string]interface{} `json:"containerSecurityContext,omitempty"`
}

type ControlPlaneGlobalMetadata struct {
	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type LabelsAndAnnotations struct {
	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are extra labels for this resource.
	Labels map[string]string `json:"labels,omitempty"`
}

type Policies struct {
	// NetworkPolicy specifies network policy options.
	NetworkPolicy NetworkPolicy `json:"networkPolicy,omitempty"`

	// PodSecurityStandard that can be enforced can be one of: empty (""), baseline, restricted or privileged
	PodSecurityStandard string `json:"podSecurityStandard,omitempty"`

	// ResourceQuota specifies resource quota options.
	ResourceQuota ResourceQuota `json:"resourceQuota,omitempty"`

	// LimitRange specifies limit range options.
	LimitRange LimitRange `json:"limitRange,omitempty"`

	// CentralAdmission defines what validating or mutating webhooks should be enforced within the virtual cluster.
	CentralAdmission CentralAdmission `json:"centralAdmission,omitempty" product:"pro"`
}

func (p Policies) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(p))
}

type ResourceQuota struct {
	// Enabled defines if the resource quota should be enabled. "auto" means that if limitRange is enabled,
	// the resourceQuota will be enabled as well.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// Quota are the quota options
	Quota map[string]interface{} `json:"quota,omitempty"`

	// ScopeSelector is the resource quota scope selector
	ScopeSelector map[string]interface{} `json:"scopeSelector,omitempty"`

	// Scopes are the resource quota scopes
	Scopes []string `json:"scopes,omitempty"`

	LabelsAndAnnotations `json:",inline"`
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
	// Enabled defines if the limit range should be deployed by vCluster. "auto" means that if resourceQuota is enabled,
	// the limitRange will be enabled as well.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// Default are the default limits for the limit range
	Default map[string]interface{} `json:"default,omitempty"`

	// DefaultRequest are the default request options for the limit range
	DefaultRequest map[string]interface{} `json:"defaultRequest,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type NetworkPolicy struct {
	// Enabled defines if the network policy should be deployed by vCluster.
	Enabled bool `json:"enabled,omitempty"`

	FallbackDNS         string              `json:"fallbackDns,omitempty"`
	OutgoingConnections OutgoingConnections `json:"outgoingConnections,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type OutgoingConnections struct {
	// IPBlock describes a particular CIDR (Ex. "192.168.1.0/24","2001:db8::/64") that is allowed
	// to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs
	// that should not be included within this rule.
	IPBlock IPBlock `json:"ipBlock,omitempty"`

	// Platform enables egress access towards loft platform
	Platform bool `json:"platform,omitempty"`
}

type IPBlock struct {
	// cidr is a string representing the IPBlock
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64"
	CIDR string `json:"cidr,omitempty"`

	// except is a slice of CIDRs that should not be included within an IPBlock
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64"
	// Except values will be rejected if they are outside the cidr range
	// +optional
	Except []string `json:"except,omitempty"`
}

type CentralAdmission struct {
	// ValidatingWebhooks are validating webhooks that should be enforced in the virtual cluster
	ValidatingWebhooks []ValidatingWebhookConfiguration `json:"validatingWebhooks,omitempty"`

	// MutatingWebhooks are mutating webhooks that should be enforced in the virtual cluster
	MutatingWebhooks []MutatingWebhookConfiguration `json:"mutatingWebhooks,omitempty"`
}

type MutatingWebhookConfiguration struct {
	// Kind is a string value representing the REST resource this object represents.
	// Servers may infer this from the endpoint the client submits requests to.
	Kind string `json:"kind,omitempty"`

	// APIVersion defines the versioned schema of this representation of an object.
	// Servers should convert recognized schemas to the latest internal value, and
	// may reject unrecognized values.
	APIVersion string `json:"apiVersion,omitempty"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
	Metadata ObjectMeta `json:"metadata,omitempty"`

	// Webhooks is a list of webhooks and the affected resources and operations.
	Webhooks []MutatingWebhook `json:"webhooks,omitempty"`
}

type MutatingWebhook struct {
	// reinvocationPolicy indicates whether this webhook should be called multiple times as part of a single admission evaluation.
	// Allowed values are "Never" and "IfNeeded".
	ReinvocationPolicy *string `json:"reinvocationPolicy,omitempty" protobuf:"bytes,10,opt,name=reinvocationPolicy,casttype=ReinvocationPolicyType"`

	ValidatingWebhook `json:",inline"`
}

type ValidatingWebhookConfiguration struct {
	// Kind is a string value representing the REST resource this object represents.
	// Servers may infer this from the endpoint the client submits requests to.
	Kind string `json:"kind,omitempty"`

	// APIVersion defines the versioned schema of this representation of an object.
	// Servers should convert recognized schemas to the latest internal value, and
	// may reject unrecognized values.
	APIVersion string `json:"apiVersion,omitempty"`

	// Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
	Metadata ObjectMeta `json:"metadata,omitempty"`

	// Webhooks is a list of webhooks and the affected resources and operations.
	Webhooks []ValidatingWebhook `json:"webhooks,omitempty"`
}

type ValidatingWebhook struct {
	// The name of the admission webhook.
	// Name should be fully qualified, e.g., imagepolicy.kubernetes.io, where
	// "imagepolicy" is the name of the webhook, and kubernetes.io is the name
	// of the organization.
	Name string `json:"name"`

	// ClientConfig defines how to communicate with the hook.
	ClientConfig ValidatingWebhookClientConfig `json:"clientConfig"`

	// Rules describes what operations on what resources/subresources the webhook cares about.
	// The webhook cares about an operation if it matches _any_ Rule.
	Rules []interface{} `json:"rules,omitempty"`

	// FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
	// allowed values are Ignore or Fail. Defaults to Fail.
	FailurePolicy *string `json:"failurePolicy,omitempty"`

	// matchPolicy defines how the "rules" list is used to match incoming requests.
	// Allowed values are "Exact" or "Equivalent".
	MatchPolicy *string `json:"matchPolicy,omitempty"`

	// NamespaceSelector decides whether to run the webhook on an object based
	// on whether the namespace for that object matches the selector. If the
	// object itself is a namespace, the matching is performed on
	// object.metadata.labels. If the object is another cluster scoped resource,
	// it never skips the webhook.
	NamespaceSelector interface{} `json:"namespaceSelector,omitempty"`

	// ObjectSelector decides whether to run the webhook based on if the
	// object has matching labels. objectSelector is evaluated against both
	// the oldObject and newObject that would be sent to the webhook, and
	// is considered to match if either object matches the selector.
	ObjectSelector interface{} `json:"objectSelector,omitempty"`

	// SideEffects states whether this webhook has side effects.
	SideEffects *string `json:"sideEffects"`

	// TimeoutSeconds specifies the timeout for this webhook.
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
	// versions the Webhook expects.
	AdmissionReviewVersions []string `json:"admissionReviewVersions"`

	// MatchConditions is a list of conditions that must be met for a request to be sent to this
	// webhook. Match conditions filter requests that have already been matched by the rules,
	// namespaceSelector, and objectSelector. An empty list of matchConditions matches all requests.
	// There are a maximum of 64 match conditions allowed.
	MatchConditions []interface{} `json:"matchConditions,omitempty"`
}

// ValidatingWebhookClientConfig contains the information to make a TLS
// connection with the webhook
type ValidatingWebhookClientConfig struct {
	// URL gives the location of the webhook, in standard URL form
	// (`scheme://host:port/path`). Exactly one of `url` or `service`
	// must be specified.
	URL *string `json:"url,omitempty"`

	// Service is a reference to the service for this webhook. Either
	// `service` or `url` must be specified.
	//
	// If the webhook is running within the cluster, then you should use `service`.
	Service *ValidatingWebhookServiceReference `json:"service,omitempty"`

	// CABundle is a PEM encoded CA bundle which will be used to validate the webhook's server certificate.
	// If unspecified, system trust roots on the apiserver are used.
	CABundle []byte `json:"caBundle,omitempty"`
}

type ValidatingWebhookServiceReference struct {
	// Namespace is the namespace of the service.
	Namespace string `json:"namespace"`

	// Name is the name of the service.
	Name string `json:"name"`

	// Path is an optional URL path which will be sent in any request to
	// this service.
	Path *string `json:"path,omitempty"`

	// If specified, the port on the service that hosting webhook.
	// Default to 443 for backward compatibility.
	// `port` should be a valid port number (1-65535, inclusive).
	Port *int32 `json:"port,omitempty"`
}

type ObjectMeta struct {
	// Name must be unique within a namespace. Is required when creating resources, although
	// some resources may allow a client to request the generation of an appropriate name
	// automatically. Name is primarily intended for creation idempotence and configuration
	// definition.
	Name string `json:"name,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type RBAC struct {
	// Role holds virtual cluster role configuration
	Role RBACRole `json:"role,omitempty"`

	// ClusterRole holds virtual cluster cluster role configuration
	ClusterRole RBACClusterRole `json:"clusterRole,omitempty"`
}

type RBACClusterRole struct {
	// Enabled defines if the cluster role should be enabled or disabled. If auto, vCluster automatically determines whether the virtual cluster requires a cluster role.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// ExtraRules will add rules to the cluster role.
	ExtraRules []map[string]interface{} `json:"extraRules,omitempty"`

	// OverwriteRules will overwrite the cluster role rules completely.
	OverwriteRules []map[string]interface{} `json:"overwriteRules,omitempty"`
}

type RBACRole struct {
	// Enabled defines if the role should be enabled or disabled.
	Enabled bool `json:"enabled,omitempty"`

	// ExtraRules will add rules to the role.
	ExtraRules []map[string]interface{} `json:"extraRules,omitempty"`

	// OverwriteRules will overwrite the role rules completely.
	OverwriteRules []map[string]interface{} `json:"overwriteRules,omitempty"`
}

type Telemetry struct {
	// Enabled specifies that the telemetry for the vCluster control plane should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	InstanceCreator    string `json:"instanceCreator,omitempty"`
	MachineID          string `json:"machineID,omitempty"`
	PlatformUserID     string `json:"platformUserID,omitempty"`
	PlatformInstanceID string `json:"platformInstanceID,omitempty"`
}

type Experimental struct {
	// Deploy allows you to configure manifests and Helm charts to deploy within the virtual cluster.
	Deploy ExperimentalDeploy `json:"deploy,omitempty"`

	// SyncSettings are advanced settings for the syncer controller.
	SyncSettings ExperimentalSyncSettings `json:"syncSettings,omitempty"`

	// GenericSync holds options to generically sync resources from virtual cluster to host.
	GenericSync ExperimentalGenericSync `json:"genericSync,omitempty"`

	// MultiNamespaceMode tells virtual cluster to sync to multiple namespaces instead of a single one. This will map each virtual cluster namespace to a single namespace in the host cluster.
	MultiNamespaceMode ExperimentalMultiNamespaceMode `json:"multiNamespaceMode,omitempty"`

	// IsolatedControlPlane is a feature to run the vCluster control plane in a different Kubernetes cluster than the workloads themselves.
	IsolatedControlPlane ExperimentalIsolatedControlPlane `json:"isolatedControlPlane,omitempty" product:"pro"`

	// VirtualClusterKubeConfig allows you to override distro specifics and specify where vCluster will find the required certificates and vCluster config.
	VirtualClusterKubeConfig VirtualClusterKubeConfig `json:"virtualClusterKubeConfig,omitempty"`

	// DenyProxyRequests denies certain requests in the vCluster proxy.
	DenyProxyRequests []DenyRule `json:"denyProxyRequests,omitempty" product:"pro"`
}

func (e Experimental) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type ExperimentalMultiNamespaceMode struct {
	// Enabled specifies if multi namespace mode should get enabled
	Enabled bool `json:"enabled,omitempty"`

	// NamespaceLabels are extra labels that will be added by vCluster to each created namespace.
	NamespaceLabels map[string]string `json:"namespaceLabels,omitempty"`
}

type ExperimentalIsolatedControlPlane struct {
	// Enabled specifies if the isolated control plane feature should be enabled.
	Enabled bool `json:"enabled,omitempty" product:"pro"`

	// Headless states that Helm should deploy the vCluster in headless mode for the isolated control plane.
	Headless bool `json:"headless,omitempty"`

	// KubeConfig is the path where to find the remote workload cluster kubeconfig.
	KubeConfig string `json:"kubeConfig,omitempty"`

	// Namespace is the namespace where to sync the workloads into.
	Namespace string `json:"namespace,omitempty"`

	// Service is the vCluster service in the remote cluster.
	Service string `json:"service,omitempty"`
}

type ExperimentalSyncSettings struct {
	// DisableSync will not sync any resources and disable most control plane functionality.
	DisableSync bool `json:"disableSync,omitempty" product:"pro"`

	// RewriteKubernetesService will rewrite the Kubernetes service to point to the vCluster service if disableSync is enabled
	RewriteKubernetesService bool `json:"rewriteKubernetesService,omitempty" product:"pro"`

	// TargetNamespace is the namespace where the workloads should get synced to.
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// SetOwner specifies if vCluster should set an owner reference on the synced objects to the vCluster service. This allows for easy garbage collection.
	SetOwner bool `json:"setOwner,omitempty"`

	// SyncLabels are labels that should get not rewritten when syncing from the virtual cluster.
	SyncLabels []string `json:"syncLabels,omitempty"`

	// HostMetricsBindAddress is the bind address for the local manager
	HostMetricsBindAddress string `json:"hostMetricsBindAddress,omitempty"`

	// VirtualMetricsBindAddress is the bind address for the virtual manager
	VirtualMetricsBindAddress string `json:"virtualMetricsBindAddress,omitempty"`
}

func (e ExperimentalSyncSettings) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type ExperimentalDeploy struct {
	// Host defines what manifests to deploy into the host cluster
	Host ExperimentalDeployHost `json:"host,omitempty"`

	// VCluster defines what manifests and charts to deploy into the vCluster
	VCluster ExperimentalDeployVCluster `json:"vcluster,omitempty"`
}

type ExperimentalDeployHost struct {
	// Manifests are raw Kubernetes manifests that should get applied within the virtual cluster.
	Manifests string `json:"manifests,omitempty"`

	// ManifestsTemplate is a Kubernetes manifest template that will be rendered with vCluster values before applying it within the virtual cluster.
	ManifestsTemplate string `json:"manifestsTemplate,omitempty"`
}

type ExperimentalDeployVCluster struct {
	// Manifests are raw Kubernetes manifests that should get applied within the virtual cluster.
	Manifests string `json:"manifests,omitempty"`

	// ManifestsTemplate is a Kubernetes manifest template that will be rendered with vCluster values before applying it within the virtual cluster.
	ManifestsTemplate string `json:"manifestsTemplate,omitempty"`

	// Helm are Helm charts that should get deployed into the virtual cluster
	Helm []ExperimentalDeployHelm `json:"helm,omitempty"`
}

type ExperimentalDeployHelm struct {
	// Chart defines what chart should get deployed.
	Chart ExperimentalDeployHelmChart `json:"chart,omitempty"`

	// Release defines what release should get deployed.
	Release ExperimentalDeployHelmRelease `json:"release,omitempty"`

	// Values defines what values should get used.
	Values string `json:"values,omitempty"`

	// Timeout defines the timeout for Helm
	Timeout string `json:"timeout,omitempty"`

	// Bundle allows to compress the Helm chart and specify this instead of an online chart
	Bundle string `json:"bundle,omitempty"`
}

type ExperimentalDeployHelmRelease struct {
	// Name of the release
	Name string `json:"name,omitempty"`

	// Namespace of the release
	Namespace string `json:"namespace,omitempty"`
}

type ExperimentalDeployHelmChart struct {
	Name     string `json:"name,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`
	Version  string `json:"version,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type PlatformConfig struct {
	// APIKey defines where to find the platform access key and host. By default, vCluster will search in the following locations in this precedence:
	// * environment variable called LICENSE
	// * secret specified under external.platform.apiKey.secretName
	// * secret called "vcluster-platform-api-key" in the vCluster namespace
	APIKey PlatformAPIKey `json:"apiKey,omitempty"`
}

// PlatformAPIKey defines where to find the platform access key. The secret key name doesn't matter as long as the secret only contains a single key.
type PlatformAPIKey struct {
	// SecretName is the name of the secret where the platform access key is stored. This defaults to vcluster-platform-api-key if undefined.
	SecretName string `json:"secretName,omitempty"`

	// Namespace defines the namespace where the access key secret should be retrieved from. If this is not equal to the namespace
	// where the vCluster instance is deployed, you need to make sure vCluster has access to this other namespace.
	Namespace string `json:"namespace,omitempty"`

	// CreateRBAC will automatically create the necessary RBAC roles and role bindings to allow vCluster to read the secret specified
	// in the above namespace, if specified.
	// This defaults to true.
	CreateRBAC *bool `json:"createRBAC,omitempty"`
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

	ClusterRole ExperimentalGenericSyncExtraRules `json:"clusterRole,omitempty"`
	Role        ExperimentalGenericSyncExtraRules `json:"role,omitempty"`
}

type ExperimentalGenericSyncExtraRules struct {
	ExtraRules []interface{} `json:"extraRules,omitempty"`
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

type DenyRule struct {
	// The name of the check.
	Name string `json:"name,omitempty"`

	// Namespace describe a list of namespaces that will be affected by the check.
	// An empty list means that all namespaces will be affected.
	// In case of ClusterScoped rules, only the Namespace resource is affected.
	Namespaces []string `json:"namespaces,omitempty"`

	// Rules describes on which verbs and on what resources/subresources the webhook is enforced.
	// The webhook is enforced if it matches any Rule.
	// The version of the request must match the rule version exactly. Equivalent matching is not supported.
	Rules []RuleWithVerbs `json:"rules,omitempty"`

	// ExcludedUsers describe a list of users for which the checks will be skipped.
	// Impersonation attempts on these users will still be subjected to the checks.
	ExcludedUsers []string `json:"excludedUsers,omitempty"`
}

type RuleWithVerbs struct {
	// APIGroups is the API groups the resources belong to. '*' is all groups.
	APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,1,rep,name=apiGroups"`

	// APIVersions is the API versions the resources belong to. '*' is all versions.
	APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,2,rep,name=apiVersions"`

	// Resources is a list of resources this rule applies to.
	Resources []string `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`

	// Scope specifies the scope of this rule.
	Scope *string `json:"scope,omitempty" protobuf:"bytes,4,rep,name=scope"`

	// Verb is the kube verb associated with the request for API requests, not the http verb. This includes things like list and watch.
	// For non-resource requests, this is the lowercase http verb.
	// If '*' is present, the length of the slice must be one.
	Verbs []string `json:"operations,omitempty"`
}

// addProToJSONSchema looks for fields with the `product:"pro"` tag and adds the pro tag to the central field.
// Requires `json:""` tag to be set as well.
func addProToJSONSchema(base *jsonschema.Schema, t reflect.Type) {
	proFields := []string{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("product")
		jsonName := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if tag == "" {
			continue
		}

		proFields = append(proFields, jsonName)
	}
	if len(proFields) == 0 {
		return
	}
	for _, field := range proFields {
		central, ok := base.Properties.Get(field)
		if !ok {
			continue
		}
		if central.Extras == nil {
			central.Extras = map[string]interface{}{}
		}
		central.Extras["pro"] = true
	}
}
