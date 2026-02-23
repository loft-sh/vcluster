package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/loft-sh/api/v4/pkg/vclusterconfig"
	yamlv3 "gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:embed values.yaml
var Values string

var ErrInvalidConfig = errors.New("invalid config")

// NewDefaultConfig creates a new config based on the values.yaml, including all default values.
func NewDefaultConfig() (*Config, error) {
	retConfig := &Config{}
	if err := yaml.Unmarshal([]byte(Values), retConfig); err != nil {
		return nil, err
	}

	platformConfig := vclusterconfig.NewDefaultPlatformConfig()
	retConfig.Sleep = platformConfig.Sleep
	retConfig.Snapshots = platformConfig.Snapshots
	retConfig.Deletion = platformConfig.Deletion
	retConfig.Platform = platformConfig.Platform

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

	// Deploy holds configuration for the deployment of vCluster.
	Deploy Deploy `json:"deploy,omitempty"`

	// Networking options related to the virtual cluster.
	Networking Networking `json:"networking,omitempty"`

	// Policies to enforce for the virtual cluster deployment as well as within the virtual cluster.
	Policies Policies `json:"policies,omitempty"`

	// Configure vCluster's control plane components and deployment.
	ControlPlane ControlPlane `json:"controlPlane,omitempty"`

	// PrivateNodes holds configuration for vCluster private nodes mode.
	PrivateNodes PrivateNodes `json:"privateNodes,omitempty"`

	// RBAC options for the virtual cluster.
	RBAC RBAC `json:"rbac,omitempty"`

	// Define which vCluster plugins to load.
	Plugins map[string]Plugins `json:"plugins,omitempty"`

	// Experimental features for vCluster. Configuration here might change, so be careful with this.
	Experimental Experimental `json:"experimental,omitempty"`

	// Configuration related to telemetry gathered about vCluster usage.
	Telemetry Telemetry `json:"telemetry,omitempty"`

	// ServiceCIDR holds the service cidr for the virtual cluster. Do not use this option anymore.
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// Specifies whether to use vCluster Pro. This is automatically inferred in newer versions. Do not use that option anymore.
	Pro bool `json:"pro,omitempty"`

	// Plugin specifies which vCluster plugins to enable. Use "plugins" instead. Do not use this option anymore.
	Plugin map[string]Plugin `json:"plugin,omitempty"`

	// Logging provides structured logging options
	Logging *Logging `json:"logging,omitempty"`

	// Sleep holds configuration for automatically putting the virtual cluster to sleep.
	Sleep *vclusterconfig.Sleep `json:"sleep,omitempty"`

	// Snapshots holds configuration for automatic vCluster snapshots.
	Snapshots *vclusterconfig.Snapshots `json:"snapshots,omitempty"`

	// Deletion holds configuration for automatic vCluster deletion.
	Deletion *vclusterconfig.Deletion `json:"deletion,omitempty"`

	// Platform holds vCluster Platform specific configuration.
	Platform *vclusterconfig.Platform `json:"platform,omitempty"`
}

// PrivateNodes enables private nodes for vCluster. When turned on, vCluster will not sync resources to the host cluster
// and instead act as a hosted control plane into which actual worker nodes can be joined via kubeadm or cluster api.
type PrivateNodes struct {
	// Enabled defines if dedicated nodes should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Kubelet holds kubelet configuration that is used for all nodes.
	Kubelet Kubelet `json:"kubelet,omitempty"`

	// AutoUpgrade holds configuration for auto upgrade.
	AutoUpgrade AutoUpgrade `json:"autoUpgrade,omitempty"`

	// JoinNode holds configuration specifically used during joining the node (see "kubeadm join").
	JoinNode JoinConfiguration `json:"joinNode,omitempty"`

	// AutoNodes stores auto nodes configuration.
	AutoNodes []PrivateNodesAutoNodes `json:"autoNodes,omitempty"`

	// VPN holds configuration for the private nodes vpn. This can be used to connect the private nodes to the control plane or
	// connect the private nodes to each other if they are not running in the same network. Platform connection is required for the vpn to work.
	VPN PrivateNodesVPN `json:"vpn,omitempty"`
}

type CloudControllerManager struct {
	// Enabled defines if the embedded cloud controller manager should be enabled. This defaults to true, but can be disabled if you want to use
	// an external cloud controller manager such as AWS or GCP. The cloud controller manager is responsible for setting the node's ip addresses as well
	// as the provider id for the node and other node metadata.
	Enabled bool `json:"enabled,omitempty"`
}

type PrivateNodesVPN struct {
	// Enabled defines if the private nodes vpn should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// NodeToNode holds configuration for the node to node vpn. This can be used to connect the private nodes to each other if they are not running in the same network.
	NodeToNode PrivateNodesVPNNodeToNode `json:"nodeToNode,omitempty"`
}

type PrivateNodesVPNNodeToNode struct {
	// Enabled defines if the node to node vpn should be enabled.
	Enabled bool `json:"enabled,omitempty"`
}

// PrivateNodesAutoNodes defines auto nodes
type PrivateNodesAutoNodes struct {
	// Provider is the node provider of the nodes in this pool.
	Provider string `json:"provider,omitempty" jsonschema:"required"`

	// Properties are the node provider properties. This is a simple key value map and can contain things
	// like region, subscription, etc. that is then used by the node provider to create the nodes and node environment.
	Properties map[string]string `json:"properties,omitempty"`

	// Static defines static node pools. Static node pools have a fixed size and are not scaled automatically.
	Static []StaticNodePool `json:"static,omitempty"`

	// Dynamic defines dynamic node pools. Dynamic node pools are scaled automatically based on the requirements within the cluster.
	// Karpenter is used under the hood to handle the scheduling of the nodes.
	Dynamic []DynamicNodePool `json:"dynamic,omitempty"`
}

type DynamicNodePool struct {
	// Name is the name of this NodePool
	Name string `json:"name" jsonschema:"required"`

	// NodeTypeSelector filters the types of nodes that can be provisioned by this pool.
	// All requirements must be met for a node type to be eligible.
	NodeTypeSelector []Requirement `json:"nodeTypeSelector,omitempty"`

	// Taints are the taints to apply to the nodes in this pool.
	Taints []KubeletJoinTaint `json:"taints,omitempty"`

	// NodeLabels are the labels to apply to the nodes in this pool.
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// Limits specify the maximum resources that can be provisioned by this node pool,
	// mapping to the 'limits' field in Karpenter's NodePool API.
	Limits map[string]string `json:"limits,omitempty"`

	// Disruption contains the parameters that relate to Karpenter's disruption logic
	Disruption DynamicNodePoolDisruption `json:"disruption,omitempty"`

	// TerminationGracePeriod is the maximum duration the controller will wait before forcefully deleting the pods on a node, measured from when deletion is first initiated.
	//
	// Warning: this feature takes precedence over a Pod's terminationGracePeriodSeconds value, and bypasses any blocked PDBs or the karpenter.sh/do-not-disrupt annotation.
	//
	// This field is intended to be used by cluster administrators to enforce that nodes can be cycled within a given time period.
	// When set, drifted nodes will begin draining even if there are pods blocking eviction. Draining will respect PDBs and the do-not-disrupt annotation until the TGP is reached.
	//
	// Karpenter will preemptively delete pods so their terminationGracePeriodSeconds align with the node's terminationGracePeriod.
	// If a pod would be terminated without being granted its full terminationGracePeriodSeconds prior to the node timeout,
	// that pod will be deleted at T = node timeout - pod terminationGracePeriodSeconds.
	//
	// The feature can also be used to allow maximum time limits for long-running jobs which can delay node termination with preStop hooks.
	// Defaults to 30s. Set to Never to wait indefinitely for pods to be drained.
	TerminationGracePeriod string `json:"terminationGracePeriod,omitempty"`

	// The amount of time a Node can live on the cluster before being removed
	ExpireAfter string `json:"expireAfter,omitempty"`

	// Weight is the weight of this node pool.
	Weight int `json:"weight,omitempty"`
}

type DynamicNodePoolDisruption struct {
	// ConsolidateAfter is the duration the controller will wait
	// before attempting to terminate nodes that are underutilized.
	// Refer to ConsolidationPolicy for how underutilization is considered.
	ConsolidateAfter string `json:"consolidateAfter,omitempty"`

	// ConsolidationPolicy describes which nodes Karpenter can disrupt through its consolidation
	// algorithm. This policy defaults to "WhenEmptyOrUnderutilized" if not specified
	ConsolidationPolicy string `json:"consolidationPolicy,omitempty"`

	// Budgets is a list of Budgets.
	// If there are multiple active budgets, Karpenter uses
	// the most restrictive value. If left undefined,
	// this will default to one budget with a value to 10%.
	Budgets []DynamicNodePoolDisruptionBudget `json:"budgets,omitempty"`
}
type DynamicNodePoolDisruptionBudget struct {
	// Nodes dictates the maximum number of NodeClaims owned by this NodePool
	// that can be terminating at once. This is calculated by counting nodes that
	// have a deletion timestamp set, or are actively being deleted by Karpenter.
	// This field is required when specifying a budget.
	Nodes string `json:"nodes,omitempty"`

	// Schedule specifies when a budget begins being active, following
	// the upstream cronjob syntax. If omitted, the budget is always active.
	// Timezones are not supported.
	Schedule string `json:"schedule,omitempty"`

	// Duration determines how long a Budget is active since each Schedule hit.
	// Only minutes and hours are accepted, as cron does not work in seconds.
	// If omitted, the budget is always active.
	// This is required if Schedule is set.
	Duration string `json:"duration,omitempty"`
}

type StaticNodePool struct {
	// Name is the name of this static nodePool
	Name string `json:"name" jsonschema:"required"`

	// NodeTypeSelector filters the types of nodes that can be provisioned by this pool.
	// All requirements must be met for a node type to be eligible.
	NodeTypeSelector []Requirement `json:"nodeTypeSelector,omitempty"`

	// Taints are the taints to apply to the nodes in this pool.
	Taints []KubeletJoinTaint `json:"taints,omitempty"`

	// NodeLabels are the labels to apply to the nodes in this pool.
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// TerminationGracePeriod is the maximum duration the controller will wait before forcefully deleting the pods on a node, measured from when deletion is first initiated.
	//
	// Warning: this feature takes precedence over a Pod's terminationGracePeriodSeconds value, and bypasses any blocked PDBs or the karpenter.sh/do-not-disrupt annotation.
	//
	// This field is intended to be used by cluster administrators to enforce that nodes can be cycled within a given time period.
	// When set, drifted nodes will begin draining even if there are pods blocking eviction. Draining will respect PDBs and the do-not-disrupt annotation until the TGP is reached.
	//
	// Karpenter will preemptively delete pods so their terminationGracePeriodSeconds align with the node's terminationGracePeriod.
	// If a pod would be terminated without being granted its full terminationGracePeriodSeconds prior to the node timeout,
	// that pod will be deleted at T = node timeout - pod terminationGracePeriodSeconds.
	//
	// The feature can also be used to allow maximum time limits for long-running jobs which can delay node termination with preStop hooks.
	// Defaults to 30s. Set to Never to wait indefinitely for pods to be drained.
	TerminationGracePeriod string `json:"terminationGracePeriod,omitempty"`

	// Quantity is the number of desired nodes in this pool.
	Quantity int `json:"quantity" jsonschema:"required"`
}

// KarpenterRequirement defines a scheduling requirement for a dynamic node pool.
// It corresponds to an entry in the 'requirements' list of a Karpenter NodePool.
type Requirement struct {
	// Property is the property on the node type to select.
	Property string `json:"property" jsonschema:"required"`

	// Operator is the comparison operator, such as "In", "NotIn", "Exists". If empty, defaults to "In".
	Operator string `json:"operator,omitempty"`

	// Values is the list of values to use for comparison. This is mutually exclusive with value.
	Values []string `json:"values,omitempty"`

	// Value is the value to use for comparison. This is mutually exclusive with values.
	Value string `json:"value,omitempty"`
}

type Deploy struct {
	// KubeProxy holds dedicated kube proxy configuration.
	KubeProxy KubeProxy `json:"kubeProxy,omitempty"`

	// Metallb holds dedicated metallb configuration.
	Metallb Metallb `json:"metallb,omitempty"`

	// CNI holds dedicated CNI configuration.
	CNI CNI `json:"cni,omitempty"`

	// LocalPathProvisioner holds dedicated local path provisioner configuration.
	LocalPathProvisioner LocalPathProvisioner `json:"localPathProvisioner,omitempty"`

	// IngressNginx holds dedicated ingress-nginx configuration.
	// Deprecated: We do not deploy ingress nginx and the project is being deprecated.
	IngressNginx IngressNginx `json:"ingressNginx,omitempty"`

	// MetricsServer holds dedicated metrics server configuration.
	MetricsServer DeployMetricsServer `json:"metricsServer,omitempty"`

	// VolumeSnapshotController holds dedicated CSI snapshot-controller configuration.
	VolumeSnapshotController VolumeSnapshotController `json:"volumeSnapshotController,omitempty"`
}

type DeployMetricsServer struct {
	// Enabled defines if metrics server should be enabled.
	Enabled bool `json:"enabled,omitempty"`
}

// VolumeSnapshotController defines CSI volumes snapshot-controller configuration.
type VolumeSnapshotController struct {
	// Enabled defines if the CSI volumes snapshot-controller should be enabled.
	Enabled bool `json:"enabled,omitempty"`
}

type IngressNginx struct {
	// Enabled defines if ingress-nginx should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// DefaultIngressClass defines if the deployed ingress class should be the default ingress class.
	DefaultIngressClass bool `json:"defaultIngressClass,omitempty"`
}

type Metallb struct {
	// Enabled defines if metallb should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// ControllerImage is the image for metallb controller.
	ControllerImage string `json:"controllerImage,omitempty"`

	// SpeakerImage is the image for metallb speaker.
	SpeakerImage string `json:"speakerImage,omitempty"`

	// IPAddressPool is the IP address pool to use for metallb.
	IPAddressPool MetallbIPAddressPool `json:"ipAddressPool,omitempty"`
}

type MetallbIPAddressPool struct {
	// Addresses is a list of IP addresses to use for the IP address pool.
	Addresses []string `json:"addresses,omitempty"`

	// L2Advertisement defines if L2 advertisement should be enabled for the IP address pool.
	L2Advertisement bool `json:"l2Advertisement,omitempty"`
}

type Standalone struct {
	// Enabled defines if standalone mode should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// DataDir defines the data directory for the standalone mode.
	DataDir string `json:"dataDir,omitempty"`

	// AutoNodes automatically deploys nodes for standalone mode.
	AutoNodes StandaloneAutoNodes `json:"autoNodes,omitempty"`

	// JoinNode holds configuration for the standalone control plane node.
	JoinNode StandaloneJoinNode `json:"joinNode,omitempty"`
}

type StandaloneAutoNodes struct {
	// Provider is the node provider of the nodes in this pool.
	Provider string `json:"provider,omitempty"`

	// Quantity is the number of nodes to deploy for standalone mode.
	Quantity int `json:"quantity,omitempty"`

	// NodeTypeSelector filters the types of nodes that can be provisioned by this pool.
	// All requirements must be met for a node type to be eligible.
	NodeTypeSelector []Requirement `json:"nodeTypeSelector,omitempty"`
}

type StandaloneJoinNode struct {
	// Enabled defines if the standalone node should be joined into the cluster. If false, only the control plane binaries will be executed and no node will show up in the actual cluster.
	Enabled bool `json:"enabled,omitempty"`

	JoinConfiguration `json:",inline"`
}

type JoinConfiguration struct {
	// PreInstallCommands are commands that will be executed before containerd, kubelet etc. is installed.
	PreInstallCommands []string `json:"preInstallCommands,omitempty"`

	// PreJoinCommands are commands that will be executed before kubeadm join is executed.
	PreJoinCommands []string `json:"preJoinCommands,omitempty"`

	// PostJoinCommands are commands that will be executed after kubeadm join is executed.
	PostJoinCommands []string `json:"postJoinCommands,omitempty"`

	// Containerd holds configuration for the containerd join process.
	Containerd ContainerdJoin `json:"containerd,omitempty"`

	// CACertPath is the path to the SSL certificate authority used to
	// secure communications between node and control-plane.
	// Defaults to "/etc/kubernetes/pki/ca.crt".
	CACertPath string `json:"caCertPath,omitempty"`

	// SkipPhases is a list of phases to skip during command execution.
	// The list of phases can be obtained with the "kubeadm join --help" command.
	SkipPhases []string `json:"skipPhases,omitempty"`

	// NodeRegistration holds configuration for the node registration similar to the kubeadm node registration.
	NodeRegistration NodeRegistration `json:"nodeRegistration,omitempty"`
}

type ContainerdJoin struct {
	// Enabled defines if containerd should be installed and configured by vCluster.
	Enabled bool `json:"enabled,omitempty"`

	// Registry holds configuration for how containerd should be configured to use a registries.
	Registry ContainerdRegistry `json:"registry,omitempty"`

	// PauseImage is the image for the pause container.
	PauseImage string `json:"pauseImage,omitempty"`
}

type ContainerdRegistry struct {
	// ConfigPath is the path to the containerd registry config.
	ConfigPath string `json:"configPath,omitempty"`

	// Mirrors holds configuration for the containerd registry mirrors. E.g. myregistry.io:5000 or docker.io. See https://github.com/containerd/containerd/blob/main/docs/hosts.md for more details.
	Mirrors map[string]ContainerdMirror `json:"mirrors,omitempty"`

	// Auth holds configuration for the containerd registry auth. See https://github.com/containerd/containerd/blob/main/docs/cri/registry.md#configure-registry-credentials for more details.
	Auth map[string]ContainerdRegistryAuth `json:"auth,omitempty"`
}

type ContainerdRegistryAuth struct {
	// Username is the username for the containerd registry.
	Username string `json:"username,omitempty"`

	// Password is the password for the containerd registry.
	Password string `json:"password,omitempty"`

	// IdentityToken is the token for the containerd registry.
	IdentityToken string `json:"identityToken,omitempty"`

	// Auth is the auth config for the containerd registry.
	Auth string `json:"auth,omitempty"`
}

type ContainerdMirror struct {
	// Server is the fallback server to use for the containerd registry mirror. E.g. https://registry-1.docker.io. See https://github.com/containerd/containerd/blob/main/docs/hosts.md for more details.
	Server string `json:"server,omitempty"`

	// CACert are paths to CA certificates to use for the containerd registry mirror.
	CACert []string `json:"caCert,omitempty"`

	// SkipVerify is a boolean to skip the certificate verification for the containerd registry mirror and allows http connections.
	SkipVerify bool `json:"skipVerify,omitempty"`

	// Capabilities is a list of capabilities to enable for the containerd registry mirror. If empty, will use pull and resolve capabilities.
	Capabilities []string `json:"capabilities,omitempty"`

	// OverridePath is a boolean to override the path for the containerd registry mirror.
	OverridePath bool `json:"overridePath,omitempty"`

	// Hosts holds configuration for the containerd registry mirror hosts. See https://github.com/containerd/containerd/blob/main/docs/hosts.md for more details.
	Hosts []ContainerdMirrorHost `json:"hosts,omitempty"`
}

type ContainerdMirrorHost struct {
	// Server is the server to use for the containerd registry mirror host. E.g. http://192.168.31.250:5000.
	Server string `json:"server,omitempty"`

	// CACert are paths to CA certificates to use for the containerd registry mirror host.
	CACert []string `json:"caCert,omitempty"`

	// SkipVerify is a boolean to skip the certificate verification for the containerd registry mirror and allows http connections.
	SkipVerify bool `json:"skipVerify,omitempty"`

	// Capabilities is a list of capabilities to enable for the containerd registry mirror. If empty, will use pull and resolve capabilities.
	Capabilities []string `json:"capabilities,omitempty"`

	// OverridePath is a boolean to override the path for the containerd registry mirror.
	OverridePath bool `json:"overridePath,omitempty"`
}

type NodeRegistration struct {
	// CRI socket is the socket for the CRI.
	CRISocket string `json:"criSocket,omitempty"`

	// KubeletExtraArgs passes through extra arguments to the kubelet. The arguments here are passed to the kubelet command line via the environment file
	// kubeadm writes at runtime for the kubelet to source. This overrides the generic base-level configuration in the kubelet-config ConfigMap
	// Flags have higher priority when parsing. These values are local and specific to the node kubeadm is executing on.
	// An argument name in this list is the flag name as it appears on the command line except without leading dash(es).
	// Extra arguments will override existing default arguments. Duplicate extra arguments are allowed.
	KubeletExtraArgs []KubeletExtraArg `json:"kubeletExtraArgs,omitempty"`

	// Taints are additional taints to set for the kubelet.
	Taints []KubeletJoinTaint `json:"taints,omitempty"`

	// IgnorePreflightErrors provides a slice of pre-flight errors to be ignored when the current node is registered, e.g. 'IsPrivilegedUser,Swap'.
	// Value 'all' ignores errors from all checks.
	IgnorePreflightErrors []string `json:"ignorePreflightErrors,omitempty"`

	// ImagePullPolicy specifies the policy for image pulling during kubeadm "init" and "join" operations.
	// The value of this field must be one of "Always", "IfNotPresent" or "Never".
	// If this field is unset kubeadm will default it to "IfNotPresent", or pull the required images if not present on the host.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

// KubeletExtraArg represents an argument with a name and a value.
type KubeletExtraArg struct {
	// Name is the name of the argument.
	Name string `json:"name"`
	// Value is the value of the argument.
	Value string `json:"value"`
}

type KubeletJoinTaint struct {
	// Required. The taint key to be applied to a node.
	Key string `json:"key"`
	// The taint value corresponding to the taint key.
	// +optional
	Value string `json:"value,omitempty"`
	// Required. The effect of the taint on pods
	// that do not tolerate the taint.
	// Valid effects are NoSchedule, PreferNoSchedule and NoExecute.
	Effect string `json:"effect"`
}

type LocalPathProvisioner struct {
	// Enabled defines if LocalPathProvisioner should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Image is the image for local path provisioner.
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// NodePath is the path on the node where to create the persistent volume directories.
	NodePath string `json:"nodePath,omitempty"`
}

type CNI struct {
	// Flannel holds dedicated Flannel configuration.
	Flannel CNIFlannel `json:"flannel,omitempty"`
}

type CNIFlannel struct {
	// Enabled defines if Flannel should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Image is the image for Flannel main container.
	Image string `json:"image,omitempty"`

	// InitImage is the image for Flannel init container.
	InitImage string `json:"initImage,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

type AutoUpgrade struct {
	// Enabled defines if auto upgrade should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Image is the image for the auto upgrade pod started by vCluster. If empty defaults to the controlPlane.statefulSet.image.
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// NodeSelector is the node selector for the auto upgrade. If empty will select all worker nodes.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// BinariesPath is the base path for the kubeadm binaries. Defaults to /usr/local/bin
	BinariesPath string `json:"binariesPath,omitempty"`

	// CNIBinariesPath is the base path for the CNI binaries. Defaults to /opt/cni/bin
	CNIBinariesPath string `json:"cniBinariesPath,omitempty"`

	// Concurrency is the number of nodes that can be upgraded at the same time.
	Concurrency int `json:"concurrency,omitempty"`
}

type Kubelet struct {
	// Config is the config for the kubelet that will be merged into the default kubelet config. More information can be found here:
	// https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/#kubelet-config-k8s-io-v1beta1-KubeletConfiguration
	Config map[string]interface{} `json:"config,omitempty"`
}

type KubeProxy struct {
	// Enabled defines if the kube proxy should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Image is the image for the kube-proxy.
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// NodeSelector is the node selector for the kube-proxy.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// PriorityClassName is the priority class name for the kube-proxy.
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Tolerations is the tolerations for the kube-proxy.
	Tolerations []interface{} `json:"tolerations,omitempty"`

	// ExtraEnv is the extra environment variables for the kube-proxy.
	ExtraEnv []interface{} `json:"extraEnv,omitempty"`

	// ExtraArgs are additional arguments to pass to the kube-proxy.
	ExtraArgs []string `json:"extraArgs,omitempty"`

	// Config is the config for the kube-proxy that will be merged into the default kube-proxy config. More information can be found here:
	// https://kubernetes.io/docs/reference/config-api/kube-proxy-config.v1alpha1/#kubeproxy-config-k8s-io-v1alpha1-KubeProxyConfiguration
	Config map[string]interface{} `json:"config,omitempty"`
}

type Konnectivity struct {
	// Server holds configuration for the konnectivity server.
	Server KonnectivityServer `json:"server,omitempty"`

	// Agent holds configuration for the konnectivity agent.
	Agent KonnectivityAgent `json:"agent,omitempty"`
}

type KonnectivityServer struct {
	// Enabled defines if the konnectivity server should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// ExtraArgs are additional arguments to pass to the konnectivity server.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type KonnectivityAgent struct {
	// Enabled defines if the konnectivity agent should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Replicas is the number of replicas for the konnectivity agent.
	Replicas int `json:"replicas,omitempty"`

	// Image is the image for the konnectivity agent.
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the policy how to pull the image.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// NodeSelector is the node selector for the konnectivity agent.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// PriorityClassName is the priority class name for the konnectivity agent.
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Tolerations is the tolerations for the konnectivity agent.
	Tolerations []interface{} `json:"tolerations,omitempty"`

	// ExtraEnv is the extra environment variables for the konnectivity agent.
	ExtraEnv []interface{} `json:"extraEnv,omitempty"`

	// ExtraArgs are additional arguments to pass to the konnectivity agent.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

// Integrations holds config for vCluster integrations with other operators or tools running on the host cluster
type Integrations struct {
	// MetricsServer reuses the metrics server from the host cluster within the vCluster.
	MetricsServer MetricsServer `json:"metricsServer,omitempty"`

	// KubeVirt reuses a host kubevirt and makes certain CRDs from it available inside the vCluster
	KubeVirt KubeVirt `json:"kubeVirt,omitempty"`

	// ExternalSecrets reuses a host external secret operator and makes certain CRDs from it available inside the vCluster.
	// - ExternalSecrets will be synced from the virtual cluster to the host cluster.
	// - SecretStores will be synced from the virtual cluster to the host cluster and then bi-directionally.
	// - ClusterSecretStores will be synced from the host cluster to the virtual cluster.
	ExternalSecrets ExternalSecrets `json:"externalSecrets,omitempty"`

	// CertManager reuses a host cert-manager and makes its CRDs from it available inside the vCluster.
	// - Certificates and Issuers will be synced from the virtual cluster to the host cluster.
	// - ClusterIssuers will be synced from the host cluster to the virtual cluster.
	CertManager CertManager `json:"certManager,omitempty"`

	// Istio syncs DestinationRules, Gateways and VirtualServices from virtual cluster to the host.
	Istio Istio `json:"istio,omitempty"`

	// Netris integration helps configuring netris networking for vCluster.
	Netris vclusterconfig.NetrisIntegration `json:"netris,omitempty"`
}

// CertManager reuses a host cert-manager and makes its CRDs from it available inside the vCluster
type CertManager struct {
	EnableSwitch

	// Sync contains advanced configuration for syncing cert-manager resources.
	Sync CertManagerSync `json:"sync,omitempty"`
}

type CertManagerSync struct {
	ToHost   CertManagerSyncToHost   `json:"toHost,omitempty"`
	FromHost CertManagerSyncFromHost `json:"fromHost,omitempty"`
}

type CertManagerSyncToHost struct {
	// Certificates defines if certificates should get synced from the virtual cluster to the host cluster.
	Certificates EnableSwitch `json:"certificates,omitempty"`

	// Issuers defines if issuers should get synced from the virtual cluster to the host cluster.
	Issuers EnableSwitch `json:"issuers,omitempty"`
}

type CertManagerSyncFromHost struct {
	// ClusterIssuers defines if (and which) cluster issuers should get synced from the host cluster to the virtual cluster.
	ClusterIssuers ClusterIssuersSyncConfig `json:"clusterIssuers,omitempty"`
}

type ClusterIssuersSyncConfig struct {
	EnableSwitch
	// Selector defines what cluster issuers should be imported.
	Selector LabelSelector `json:"selector,omitempty"`
}

type Istio struct {
	EnableSwitch
	Sync IstioSync `json:"sync,omitempty"`
}

type IstioSync struct {
	ToHost IstioSyncToHost `json:"toHost,omitempty"`
}

type IstioSyncToHost struct {
	DestinationRules EnableSwitch `json:"destinationRules,omitempty"`

	Gateways EnableSwitch `json:"gateways,omitempty"`

	VirtualServices EnableSwitch `json:"virtualServices,omitempty"`
}

// ExternalSecrets reuses a host external secret operator and makes certain CRDs from it available inside the vCluster
type ExternalSecrets struct {
	// Enabled defines whether the external secret integration is enabled or not
	Enabled bool `json:"enabled,omitempty"`
	// Version defines the version of the external secrets operator to use. If empty, the storage version will be used.
	Version string `json:"version,omitempty"`
	// Webhook defines whether the host webhooks are reused or not
	Webhook EnableSwitch `json:"webhook,omitempty"`
	// Sync defines the syncing behavior for the integration
	Sync ExternalSecretsSync `json:"sync,omitempty"`
}

type ExternalSecretsSync struct {
	// ToHost defines what resources are synced from the virtual cluster to the host
	ToHost ExternalSecretsSyncToHostConfig `json:"toHost,omitempty"`
	// FromHost defines what resources are synced from the host cluster to the virtual cluster
	FromHost ExternalSecretsSyncFromHostConfig `json:"fromHost,omitempty"`
}

type ExternalSecretsSyncToHostConfig struct {
	// ExternalSecrets allows to configure if only a subset of ExternalSecrets matching a label selector should get synced from the virtual cluster to the host cluster.
	ExternalSecrets SelectorConfig `json:"externalSecrets,omitempty"`
	// Stores defines if secret stores should get synced from the virtual cluster to the host cluster and then bi-directionally.
	Stores EnableSwitchSelector `json:"stores,omitempty"`
}

type ExternalSecretsSyncFromHostConfig struct {
	// ClusterStores defines if cluster secrets stores should get synced from the host cluster to the virtual cluster.
	ClusterStores EnableSwitchSelector `json:"clusterStores,omitempty"`
}

type SelectorConfig struct {
	Selector StandardLabelSelector `json:"selector,omitempty"`
}

type EnableSwitchSelector struct {
	SelectorConfig
	EnableSwitch
}

type LabelSelector struct {
	// Labels defines what labels should be looked for
	Labels map[string]string `json:"labels,omitempty"`
}

// KubeVirt reuses a host kubevirt and makes certain CRDs from it available inside the vCluster
type KubeVirt struct {
	// Enabled signals if the integration should be enabled
	Enabled bool `json:"enabled,omitempty"`
	// APIService holds information about where to find the virt-api service. Defaults to virt-api/kubevirt.
	APIService APIService `json:"apiService,omitempty"`
	// Webhook holds configuration for enabling the webhook within the vCluster
	Webhook EnableSwitch `json:"webhook,omitempty"`
	// Sync holds configuration on what resources to sync
	Sync KubeVirtSync `json:"sync,omitempty"`
}

// KubeVirtSync are the crds that are supported by this integration
type KubeVirtSync struct {
	// If DataVolumes should get synced
	DataVolumes EnableSwitch `json:"dataVolumes,omitempty"`
	// If VirtualMachineInstanceMigrations should get synced
	VirtualMachineInstanceMigrations EnableSwitch `json:"virtualMachineInstanceMigrations,omitempty"`
	// If VirtualMachineInstances should get synced
	VirtualMachineInstances EnableSwitch `json:"virtualMachineInstances,omitempty"`
	// If VirtualMachines should get synced
	VirtualMachines EnableSwitch `json:"virtualMachines,omitempty"`
	// If VirtualMachineClones should get synced
	VirtualMachineClones EnableSwitch `json:"virtualMachineClones,omitempty"`
	// If VirtualMachinePools should get synced
	VirtualMachinePools EnableSwitch `json:"virtualMachinePools,omitempty"`
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

func (c *Config) UnmarshalYAMLStrict(data []byte) error {
	return UnmarshalYAMLStrict(data, c)
}

// GetPlatform returns the platform configuration from the new top-level field.
func (c *Config) GetPlatformConfig() *vclusterconfig.Platform {
	if c.Platform != nil {
		return c.Platform
	}

	return &vclusterconfig.Platform{}
}

func (c *Config) SetPlatformConfig(platformConfig *PlatformConfig) error {
	c.Platform = platformConfig
	return nil
}

// BackingStoreType returns the backing store type of the vCluster.
// If no backing store is enabled, it returns StoreTypeUnknown.
func (c *Config) BackingStoreType() StoreType {
	switch {
	case c.ControlPlane.BackingStore.Etcd.Embedded.Enabled:
		return StoreTypeEmbeddedEtcd
	case c.ControlPlane.BackingStore.Etcd.External.Enabled:
		return StoreTypeExternalEtcd
	case c.ControlPlane.BackingStore.Etcd.Deploy.Enabled:
		return StoreTypeDeployedEtcd
	case c.ControlPlane.BackingStore.Database.Embedded.Enabled:
		return StoreTypeEmbeddedDatabase
	case c.ControlPlane.BackingStore.Database.External.Enabled:
		return StoreTypeExternalDatabase
	default:
		return StoreTypeEmbeddedDatabase
	}
}

func (c *Config) EmbeddedDatabase() bool {
	return !c.ControlPlane.BackingStore.Database.External.Enabled && !c.ControlPlane.BackingStore.Etcd.Embedded.Enabled && !c.ControlPlane.BackingStore.Etcd.Deploy.Enabled && !c.ControlPlane.BackingStore.Etcd.External.Enabled
}

func (c *Config) IsDockerRegistryDaemonEnabled() bool {
	return c.Experimental.Docker.Enabled && c.Experimental.Docker.RegistryProxy.Enabled
}

func (c *Config) IsVirtualSchedulerEnabled() bool {
	return c.ControlPlane.Distro.K8S.Scheduler.Enabled || c.ControlPlane.Advanced.VirtualScheduler.Enabled
}

func (c *Config) IsConfiguredForSleepMode() bool {
	if c == nil {
		return false
	}

	return c.Sleep != nil
}

func (c *Config) IsConfiguredForAutoDeletion() bool {
	if c == nil || c.Deletion == nil {
		return false
	}

	return c.Deletion.Auto != nil
}

// ValidateChanges checks for disallowed config changes.
func ValidateChanges(oldCfg, newCfg *Config) error {
	if err := ValidateStoreChanges(newCfg.BackingStoreType(), oldCfg.BackingStoreType()); err != nil {
		return err
	}
	if err := ValidateNamespaceSyncChanges(oldCfg, newCfg); err != nil { //nolint:revive
		return err
	}
	if err := ValidateVPNChanges(oldCfg, newCfg); err != nil {
		return err
	}

	return nil
}

// ValidateStoreChanges checks whether migrating from one store to the other is allowed.
func ValidateStoreChanges(currentStoreType, previousStoreType StoreType) error {
	if currentStoreType == previousStoreType {
		return nil
	}

	switch currentStoreType {
	case StoreTypeDeployedEtcd:
		fallthrough
	case StoreTypeEmbeddedEtcd:
		// switching from external ETCD, deploy ETCD, or embedded (SQLite) to deployed or embedded ETCD is valid
		if previousStoreType == StoreTypeExternalEtcd || previousStoreType == StoreTypeDeployedEtcd || previousStoreType == StoreTypeEmbeddedDatabase {
			return nil
		}
	case StoreTypeExternalDatabase:
		// switching from embedded to external ETCD is allowed because of a bug that labeled store types as embedded but used
		// external info if provided when the external "enabled" flag was not used. Now, using the "enabled" flag is required or
		// SQLite is used. The exception to allow this switch is necessary so they can toggle the "enabled" flag if the cluster
		// was previously using external. Otherwise, after upgrade the vCluster will start using a fresh SQLite database.
		if previousStoreType == StoreTypeEmbeddedDatabase {
			return nil
		}
	default:
	}
	return fmt.Errorf("seems like you were using %s as a store before and now have switched to %s,"+
		" please make sure to not switch between vCluster stores", previousStoreType, currentStoreType)
}

func ValidateNamespaceSyncChanges(oldCfg, newCfg *Config) error {
	oldNamespaceConf := oldCfg.Sync.ToHost.Namespaces
	newNamespaceConf := newCfg.Sync.ToHost.Namespaces

	if oldNamespaceConf.Enabled != newNamespaceConf.Enabled {
		return fmt.Errorf("sync.toHost.namespaces.enabled is not allowed to be changed")
	}

	if oldNamespaceConf.MappingsOnly != newNamespaceConf.MappingsOnly {
		return fmt.Errorf("sync.toHost.namespaces.mappingsOnly is not allowed to be changed")
	}

	if !reflect.DeepEqual(oldNamespaceConf.Mappings.ByName, newNamespaceConf.Mappings.ByName) {
		return fmt.Errorf("sync.toHost.namespaces.mappings.byName is not allowed to be changed")
	}

	if !reflect.DeepEqual(oldNamespaceConf.Patches, newNamespaceConf.Patches) {
		return fmt.Errorf("sync.toHost.namespaces.patches is not allowed to be changed")
	}

	return nil
}

func ValidateVPNChanges(oldCfg *Config, newCfg *Config) error {
	if oldCfg.PrivateNodes.VPN.Enabled != newCfg.PrivateNodes.VPN.Enabled {
		return fmt.Errorf("privateNodes.vpn.enabled is not allowed to be changed")
	}

	if oldCfg.PrivateNodes.VPN.NodeToNode.Enabled != newCfg.PrivateNodes.VPN.NodeToNode.Enabled {
		return fmt.Errorf("privateNodes.vpn.nodeToNode.enabled is not allowed to be changed")
	}

	return nil
}

func (c *Config) IsProFeatureEnabled() bool {
	if os.Getenv("SKIP_VALIDATE_PRO_FEATURES") == "true" {
		return false
	}

	if len(c.Networking.ResolveDNS) > 0 {
		return true
	}

	if c.ControlPlane.CoreDNS.Embedded {
		return true
	}

	if c.ControlPlane.BackingStore.Database.External.Enabled {
		return true
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

	if len(c.Experimental.DenyProxyRequests) > 0 {
		return true
	}

	if len(c.Sync.ToHost.CustomResources) > 0 || len(c.Sync.FromHost.CustomResources) > 0 {
		return true
	}

	if c.Sync.ToHost.Namespaces.Enabled {
		return true
	}

	if c.Sync.ToHost.Pods.HybridScheduling.Enabled {
		return true
	}

	// private nodes is allowed in standalone mode
	if c.PrivateNodes.Enabled && !c.ControlPlane.Standalone.Enabled {
		return true
	}

	if c.Sync.ToHost.ResourceClaimTemplates.Enabled {
		return true
	}

	if c.Sync.ToHost.ResourceClaims.Enabled {
		return true
	}

	if c.Sync.FromHost.DeviceClasses.Enabled {
		return true
	}

	if c.Sleep != nil {
		return true
	}

	if c.Snapshots != nil {
		return true
	}

	if c.Deletion != nil {
		return true
	}

	if c.Platform != nil {
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
	ExportKubeConfigProperties

	// Declare in which host cluster secret vCluster should store the generated virtual cluster kubeconfig.
	// If this is not defined, vCluster will create it with `vc-NAME`. If you specify another name,
	// vCluster creates the config in this other secret.
	//
	// Deprecated: Use AdditionalSecrets instead.
	Secret ExportKubeConfigSecretReference `json:"secret,omitempty"`

	// AdditionalSecrets specifies the additional host cluster secrets in which vCluster will store the
	// generated virtual cluster kubeconfigs.
	AdditionalSecrets []ExportKubeConfigAdditionalSecretReference `json:"additionalSecrets,omitempty"`
}

// GetAdditionalSecrets returns optional additional kubeconfig Secrets.
//
// If the deprecated Secret property is set, GetAdditionalSecrets only returns that secret, and
// AdditionalSecrets is ignored. On the other hand, if the AdditionalSecrets property is set,
// GetAdditionalSecrets returns the secrets config from the AdditionalSecrets, and Secret property
// is ignored.
func (e *ExportKubeConfig) GetAdditionalSecrets() []ExportKubeConfigAdditionalSecretReference {
	if e.Secret.IsSet() {
		return []ExportKubeConfigAdditionalSecretReference{
			{
				ExportKubeConfigProperties: e.ExportKubeConfigProperties,
				Namespace:                  e.Secret.Namespace,
				Name:                       e.Secret.Name,
			},
		}
	}

	return e.AdditionalSecrets
}

type ExportKubeConfigProperties struct {
	// Context is the name of the context within the generated kubeconfig to use.
	Context string `json:"context,omitempty"`

	// Override the default https://localhost:8443 and specify a custom hostname for the generated kubeconfig.
	Server string `json:"server,omitempty"`

	// If tls should get skipped for the server
	Insecure bool `json:"insecure,omitempty"`

	// ServiceAccount can be used to generate a service account token instead of the default certificates.
	ServiceAccount ExportKubeConfigServiceAccount `json:"serviceAccount,omitempty"`
}

type ExportKubeConfigServiceAccount struct {
	// Name of the service account to be used to generate a service account token instead of the default certificates.
	Name string `json:"name,omitempty"`

	// Namespace of the service account to be used to generate a service account token instead of the default certificates.
	// If omitted, will use the kube-system namespace.
	Namespace string `json:"namespace,omitempty"`

	// ClusterRole to assign to the service account.
	ClusterRole string `json:"clusterRole,omitempty"`
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

// IsSet checks if at least one ExportKubeConfigSecretReference property is set.
func (s *ExportKubeConfigSecretReference) IsSet() bool {
	return *s != (ExportKubeConfigSecretReference{})
}

// ExportKubeConfigAdditionalSecretReference defines the additional host cluster secret in which
// vCluster stores the generated virtual cluster kubeconfigs.
type ExportKubeConfigAdditionalSecretReference struct {
	ExportKubeConfigProperties

	// Name is the name of the secret where the kubeconfig is stored.
	Name string `json:"name,omitempty"`

	// Namespace where vCluster stores the kubeconfig secret. If this is not equal to the namespace
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
	Ingresses EnableSwitchWithPatches `json:"ingresses,omitempty"`

	// Services defines if services created within the virtual cluster should get synced to the host cluster.
	Services EnableSwitchWithPatches `json:"services,omitempty"`

	// Endpoints defines if endpoints created within the virtual cluster should get synced to the host cluster.
	Endpoints EnableSwitchWithPatches `json:"endpoints,omitempty"`

	// EndpointSlices defines if endpointslices created within the virtual cluster should get synced to the host cluster.
	EndpointSlices EnableSwitchWithPatches `json:"endpointSlices,omitempty"`

	// NetworkPolicies defines if network policies created within the virtual cluster should get synced to the host cluster.
	NetworkPolicies EnableSwitchWithPatches `json:"networkPolicies,omitempty"`

	// PersistentVolumeClaims defines if persistent volume claims created within the virtual cluster should get synced to the host cluster.
	PersistentVolumeClaims EnableSwitchWithPatches `json:"persistentVolumeClaims,omitempty"`

	// PersistentVolumes defines if persistent volumes created within the virtual cluster should get synced to the host cluster.
	PersistentVolumes EnableSwitchWithPatches `json:"persistentVolumes,omitempty"`

	// VolumeSnapshots defines if volume snapshots created within the virtual cluster should get synced to the host cluster.
	VolumeSnapshots EnableSwitchWithPatches `json:"volumeSnapshots,omitempty"`

	// VolumeSnapshotContents defines if volume snapshot contents created within the virtual cluster should get synced to the host cluster.
	VolumeSnapshotContents EnableSwitchWithPatches `json:"volumeSnapshotContents,omitempty"`

	// StorageClasses defines if storage classes created within the virtual cluster should get synced to the host cluster.
	StorageClasses EnableSwitchWithPatches `json:"storageClasses,omitempty"`

	// ServiceAccounts defines if service accounts created within the virtual cluster should get synced to the host cluster.
	ServiceAccounts EnableSwitchWithPatches `json:"serviceAccounts,omitempty"`

	// PodDisruptionBudgets defines if pod disruption budgets created within the virtual cluster should get synced to the host cluster.
	PodDisruptionBudgets EnableSwitchWithPatches `json:"podDisruptionBudgets,omitempty"`

	// PriorityClasses defines if priority classes created within the virtual cluster should get synced to the host cluster.
	PriorityClasses EnableSwitchWithPatches `json:"priorityClasses,omitempty"`

	// CustomResources defines what custom resources should get synced from the virtual cluster to the host cluster. vCluster will copy the definition automatically from host cluster to virtual cluster on startup.
	// vCluster will also automatically add any required RBAC permissions to the vCluster role for this to work.
	CustomResources map[string]SyncToHostCustomResource `json:"customResources,omitempty"`

	// Namespaces defines if namespaces created within the virtual cluster should get synced to the host cluster.
	Namespaces SyncToHostNamespaces `json:"namespaces,omitempty"`

	// ResourceClaim defines if resource claims created within the virtual cluster should get synced to the host cluster.
	ResourceClaims EnableSwitchWithPatches `json:"resourceClaims,omitempty"`

	// ResourceClaimTemplates defines if resourceClaimTemplates created within the virtual cluster should get synced to the host cluster.
	ResourceClaimTemplates EnableSwitchWithPatches `json:"resourceClaimTemplates,omitempty"`
}

type EnableSwitchWithPatches struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`
}

type EnableSwitchWithResourcesMappings struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`

	// Mappings for Namespace and Object
	Mappings FromHostMappings `json:"mappings,omitempty"`
}

type FromHostMappings struct {
	// ByName is a map of host-object-namespace/host-object-name: virtual-object-namespace/virtual-object-name.
	// There are several wildcards supported:
	// 1. To match all objects in host namespace and sync them to different namespace in vCluster:
	// byName:
	//   "foo/*": "foo-in-virtual/*"
	// 2. To match specific object in the host namespace and sync it to the same namespace with the same name:
	// byName:
	//   "foo/my-object": "foo/my-object"
	// 3. To match specific object in the host namespace and sync it to the same namespace with different name:
	// byName:
	//   "foo/my-object": "foo/my-virtual-object"
	// 4. To match all objects in the vCluster host namespace and sync them to a different namespace in vCluster:
	// byName:
	//   "": "my-virtual-namespace/*"
	// 5. To match specific objects in the vCluster host namespace and sync them to a different namespace in vCluster:
	// byName:
	//   "/my-object": "my-virtual-namespace/my-object"
	ByName map[string]string `json:"byName,omitempty"`
}

type SyncFromHost struct {
	// Nodes defines if nodes should get synced from the host cluster to the virtual cluster, but not back.
	Nodes SyncNodes `json:"nodes,omitempty"`

	// Events defines if events should get synced from the host cluster to the virtual cluster, but not back.
	Events EnableSwitchWithPatches `json:"events,omitempty"`

	// IngressClasses defines if ingress classes should get synced from the host cluster to the virtual cluster, but not back.
	IngressClasses EnableSwitchWithPatchesAndSelector `json:"ingressClasses,omitempty"`

	// RuntimeClasses defines if runtime classes should get synced from the host cluster to the virtual cluster, but not back.
	RuntimeClasses EnableSwitchWithPatchesAndSelector `json:"runtimeClasses,omitempty"`

	// PriorityClasses defines if priority classes classes should get synced from the host cluster to the virtual cluster, but not back.
	PriorityClasses EnableSwitchWithPatchesAndSelector `json:"priorityClasses,omitempty"`

	// StorageClasses defines if storage classes should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	StorageClasses EnableAutoSwitchWithPatchesAndSelector `json:"storageClasses,omitempty"`

	// CSINodes defines if csi nodes should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSINodes EnableAutoSwitchWithPatches `json:"csiNodes,omitempty"`

	// CSIDrivers defines if csi drivers should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSIDrivers EnableAutoSwitchWithPatches `json:"csiDrivers,omitempty"`

	// CSIStorageCapacities defines if csi storage capacities should get synced from the host cluster to the virtual cluster, but not back. If auto, is automatically enabled when the virtual scheduler is enabled.
	CSIStorageCapacities EnableAutoSwitchWithPatches `json:"csiStorageCapacities,omitempty"`

	// CustomResources defines what custom resources should get synced read-only to the virtual cluster from the host cluster. vCluster will automatically add any required RBAC to the vCluster cluster role.
	CustomResources map[string]SyncFromHostCustomResource `json:"customResources,omitempty"`

	// VolumeSnapshotClasses defines if volume snapshot classes created within the virtual cluster should get synced to the host cluster.
	VolumeSnapshotClasses EnableSwitchWithPatches `json:"volumeSnapshotClasses,omitempty"`

	// ConfigMaps defines if config maps in the host should get synced to the virtual cluster.
	ConfigMaps EnableSwitchWithResourcesMappings `json:"configMaps,omitempty"`

	// Secrets defines if secrets in the host should get synced to the virtual cluster.
	Secrets EnableSwitchWithResourcesMappings `json:"secrets,omitempty"`

	// DeviceClasses defines if device classes in the host should get synced to the virtual cluster
	DeviceClasses EnableSwitchWithPatchesAndSelector `json:"deviceClasses,omitempty"`
}

type StandardLabelSelector v1.LabelSelector

func (s StandardLabelSelector) Empty() bool {
	selector, err := s.ToSelector()
	return err == nil && selector.Empty()
}

func (s StandardLabelSelector) Matches(obj client.Object) (bool, error) {
	selector, err := s.ToSelector()
	if err != nil {
		return false, fmt.Errorf("failed to convert label selector: %w", err)
	}
	return selector.Matches(labels.Set(obj.GetLabels())), nil
}

func (s StandardLabelSelector) ToSelector() (labels.Selector, error) {
	ls := v1.LabelSelector(s)
	return v1.LabelSelectorAsSelector(&ls)
}

type EnableSwitchWithPatchesAndSelector struct {
	EnableSwitchWithPatches

	// Selector defines the selector to use for the resource. If not set, all resources of that type will be synced.
	Selector StandardLabelSelector `json:"selector,omitempty"`
}

type EnableAutoSwitchWithPatchesAndSelector struct {
	EnableAutoSwitchWithPatches

	// Selector defines the selector to use for the resource. If not set, all resources of that type will be synced.
	Selector StandardLabelSelector `json:"selector,omitempty"`
}

// SyncToHostNamespaces defines how namespaces should be synced from the virtual cluster to the host cluster.
type SyncToHostNamespaces struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty" jsonschema:"required"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`

	// Mappings for Namespace and Object
	Mappings FromHostMappings `json:"mappings,omitempty"`

	// MappingsOnly defines if creation of namespaces not matched by mappings should be allowed.
	MappingsOnly bool `json:"mappingsOnly,omitempty"`

	// ExtraLabels are additional labels to add to the namespace in the host cluster.
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

type SyncToHostCustomResource struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty" jsonschema:"required"`

	// Scope defines the scope of the resource. If undefined, will use Namespaced. Currently only Namespaced is supported.
	Scope Scope `json:"scope,omitempty"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`
}

type Scope string

const (
	ScopeNamespaced Scope = "Namespaced"
	ScopeCluster    Scope = "Cluster"
)

type TranslatePatch struct {
	// Path is the path within the patch to target. If the path is not found within the patch, the patch is not applied.
	Path string `json:"path,omitempty" jsonschema:"required"`

	// Expression transforms the value according to the given JavaScript expression.
	Expression string `json:"expression,omitempty"`

	// ReverseExpression transforms the value according to the given JavaScript expression.
	ReverseExpression string `json:"reverseExpression,omitempty"`

	// Reference treats the path value as a reference to another object and will rewrite it based on the chosen mode
	// automatically. In single-namespace mode this will translate the name to "vxxxxxxxxx" to avoid conflicts with
	// other names, in multi-namespace mode this will not translate the name.
	Reference *TranslatePatchReference `json:"reference,omitempty"`

	// Labels treats the path value as a labels selector.
	Labels *TranslatePatchLabels `json:"labels,omitempty"`
}

type TranslatePatchLabels struct{}

type TranslatePatchReference struct {
	// APIVersion is the apiVersion of the referenced object.
	APIVersion string `json:"apiVersion,omitempty" jsonschema:"required"`

	// APIVersionPath is optional relative path to use to determine the kind. If APIVersionPath is not found, will fallback to apiVersion.
	APIVersionPath string `json:"apiVersionPath,omitempty"`

	// Kind is the kind of the referenced object.
	Kind string `json:"kind,omitempty" jsonschema:"required"`

	// KindPath is the optional relative path to use to determine the kind. If KindPath is not found, will fallback to kind.
	KindPath string `json:"kindPath,omitempty"`

	// NamePath is the optional relative path to the reference name within the object.
	NamePath string `json:"namePath,omitempty"`

	// NamespacePath is the optional relative path to the reference namespace within the object. If omitted or not found, namespacePath equals to the
	// metadata.namespace path of the object.
	NamespacePath string `json:"namespacePath,omitempty"`
}

type SyncFromHostCustomResource struct {
	// Enabled defines if this option should be enabled.
	Enabled bool `json:"enabled,omitempty" jsonschema:"required"`

	// Scope defines the scope of the resource
	Scope Scope `json:"scope,omitempty" jsonschema:"required"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`

	// Mappings for Namespace and Object
	Mappings FromHostMappings `json:"mappings,omitempty"`
}

type EnableAutoSwitch struct {
	// Enabled defines if this option should be enabled.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`
}

type EnableAutoSwitchWithPatches struct {
	// Enabled defines if this option should be enabled.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`
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

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`
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

	// RuntimeClassName is the runtime class to set for synced pods.
	RuntimeClassName string `json:"runtimeClassName,omitempty"`

	// PriorityClassName is the priority class to set for synced pods.
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// RewriteHosts is a special option needed to rewrite statefulset containers to allow the correct FQDN. virtual cluster will add
	// a small container to each stateful set pod that will initially rewrite the /etc/hosts file to match the FQDN expected by
	// the virtual cluster.
	RewriteHosts SyncRewriteHosts `json:"rewriteHosts,omitempty"`

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`

	// HybridScheduling is used to enable and configure hybrid scheduling for pods in the virtual cluster.
	HybridScheduling HybridScheduling `json:"hybridScheduling,omitempty"`
}

type SyncRewriteHosts struct {
	// Enabled specifies if rewriting stateful set pods should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// InitContainer holds extra options for the init container used by vCluster to rewrite the FQDN for stateful set pods.
	InitContainer SyncRewriteHostsInitContainer `json:"initContainer,omitempty"`
}

type SyncRewriteHostsInitContainer struct {
	// Image is the image virtual cluster should use to rewrite this FQDN.
	Image Image `json:"image,omitempty"`

	// Resources are the resources that should be assigned to the init container for each stateful set init container.
	Resources Resources `json:"resources,omitempty"`
}

type HybridScheduling struct {
	// Enabled specifies if hybrid scheduling is enabled.
	Enabled bool `json:"enabled,omitempty"`

	// HostSchedulers is a list of schedulers that are deployed on the host cluster.
	HostSchedulers []string `json:"hostSchedulers,omitempty"`
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

	// Patches patch the resource according to the provided specification.
	Patches []TranslatePatch `json:"patches,omitempty"`
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
	// ServiceCIDR holds the service cidr for the virtual cluster. This should only be set if privateNodes.enabled is true or vCluster cannot detect the host service cidr.
	ServiceCIDR string `json:"serviceCIDR,omitempty"`

	// PodCIDR holds the pod cidr for the virtual cluster. This should only be set if privateNodes.enabled is true.
	PodCIDR string `json:"podCIDR,omitempty"`

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
	// Endpoint is the endpoint of the virtual cluster. This is used to connect to the virtual cluster.
	Endpoint string `json:"endpoint,omitempty"`

	// Distro holds virtual cluster related distro options. A distro cannot be changed after vCluster is deployed.
	Distro Distro `json:"distro,omitempty"`

	// Standalone holds configuration for standalone mode. Standalone mode is set automatically when no container is detected and
	// also implies privateNodes.enabled.
	Standalone Standalone `json:"standalone,omitempty"`

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

type KubeVip struct {
	// Enabled defines if embedded kube-vip should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Interface is the network interface on which the VIP is announced.
	Interface string `json:"interface,omitempty"`

	// Gateway is the gateway address in CIDR notation (e.g., 10.100.0.1/24).
	// This is used to configure policy-based routing for the VIP and must include the subnet prefix.
	Gateway string `json:"gateway,omitempty"`
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
	// It defaults to the vCluster pro repository that includes the optional pro modules that are turned off by default.
	// If you still want to use the pure OSS build, set the repository to 'loft-sh/vcluster-oss'.
	Image Image `json:"image,omitempty"`

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

	// Set DNS policy for the pod.
	DNSPolicy DNSPolicy `json:"dnsPolicy,omitempty"`

	// Specifies the DNS parameters of a pod.
	DNSConfig *PodDNSConfig `json:"dnsConfig,omitempty"`

	// InitContainers are additional init containers for the statefulSet.
	InitContainers []interface{} `json:"initContainers,omitempty"`

	// SidecarContainers are additional sidecar containers for the statefulSet.
	SidecarContainers []interface{} `json:"sidecarContainers,omitempty"`

	// HostAliases allows you to add custom entries to the /etc/hosts file of each Pod created.
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`
}

type Distro struct {
	// K8S holds K8s relevant configuration.
	K8S DistroK8s `json:"k8s,omitempty"`
}

type DistroK8s struct {
	// Enabled specifies if the K8s distro should be enabled. Only one distro can be enabled at the same time.
	Enabled bool `json:"enabled,omitempty"`

	// Version is the Kubernetes version to use.
	Version string `json:"version,omitempty"`

	// APIServer holds configuration specific to starting the api server.
	APIServer DistroContainerEnabled `json:"apiServer,omitempty"`

	// ControllerManager holds configuration specific to starting the controller manager.
	ControllerManager DistroContainerEnabled `json:"controllerManager,omitempty"`

	// Scheduler holds configuration specific to starting the scheduler.
	Scheduler DistroContainerEnabled `json:"scheduler,omitempty"`

	DistroCommon `json:",inline"`
}

type DistroCommon struct {
	// Image is the distro image
	Image Image `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy for the distro image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Env are extra environment variables to use for the main container and NOT the init container.
	Env []map[string]interface{} `json:"env,omitempty"`

	// Resources for the distro init container
	Resources map[string]interface{} `json:"resources,omitempty"`

	// Security options can be used for the distro init container
	SecurityContext map[string]interface{} `json:"securityContext,omitempty"`
}

type DistroContainerEnabled struct {
	// Enabled signals this container should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// Command is the command to start the distro binary. This will override the existing command.
	Command []string `json:"command,omitempty"`

	// ExtraArgs are additional arguments to pass to the distro binary.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type Image struct {
	// Registry is the registry of the container image, e.g. my-registry.com or ghcr.io. This setting can be globally
	// overridden via the controlPlane.advanced.defaultImageRegistry option. Empty means docker hub.
	Registry string `json:"registry,omitempty"`

	// Repository is the repository of the container image, e.g. my-repo/my-image
	Repository string `json:"repository,omitempty"`

	// Tag is the tag of the container image, and is the default version.
	Tag string `json:"tag,omitempty"`
}

// UnmarshalJSON makes the schema change from string to Image backwards compatible
func (i *Image) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		ParseImageRef(str, i)
		return nil
	}

	type Alias Image
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*i = Image(aux)
	return nil
}

// UnmarshalYAML makes the schema change from string to Image backwards compatible
func (i *Image) UnmarshalYAML(node *yamlv3.Node) error {
	if node.Kind == yamlv3.ScalarNode {
		ParseImageRef(node.Value, i)
		return nil
	}

	type Alias Image
	var aux Alias
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*i = Image(aux)
	return nil
}

func (i *Image) String() (ref string) {
	if i == nil {
		return
	}

	if i.Registry != "" {
		ref = i.Registry + "/"
	}

	if i.Registry != "" && i.Repository != "" && !strings.ContainsRune(i.Repository, '/') {
		ref += "library/"
	}
	ref += i.Repository

	if i.Tag != "" {
		ref += ":" + i.Tag
	}

	return ref
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
	External ExternalDatabaseKine `json:"external,omitempty"`
}

type ExternalDatabaseKine struct {
	DatabaseKine

	// Connector specifies a secret located in a connected vCluster Platform that contains database server connection information
	// to be used by Platform to create a database and database user for the vCluster.
	// and non-privileged user. A kine endpoint should be created using the database and user on Platform registration.
	// This is optional.
	Connector string `json:"connector,omitempty"`
}

type DatabaseKine struct {
	// Enabled defines if the database should be used.
	Enabled bool `json:"enabled,omitempty"`

	// DataSource is the kine dataSource to use for the database. This depends on the database format.
	// This is optional for the external database. Examples:
	// * mysql: mysql://username:password@tcp(hostname:3306)/vcluster
	// * postgres: postgres://username:password@hostname:5432/vcluster
	DataSource string `json:"dataSource,omitempty"`

	// IdentityProvider is the kine identity provider to use when generating temporary authentication tokens for enhanced security.
	// This is optional for the external database. Examples:
	// * aws: RDS IAM Authentication
	IdentityProvider string `json:"identityProvider,omitempty"`

	// KeyFile is the key file to use for the database. This is optional.
	KeyFile string `json:"keyFile,omitempty"`

	// CertFile is the cert file to use for the database. This is optional.
	CertFile string `json:"certFile,omitempty"`

	// CaFile is the ca file to use for the database. This is optional.
	CaFile string `json:"caFile,omitempty"`

	// ExtraArgs are additional arguments to pass to Kine.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type Etcd struct {
	// Embedded defines to use embedded etcd as a storage backend for the virtual cluster
	Embedded EtcdEmbedded `json:"embedded,omitempty" product:"pro"`

	// Deploy defines to use an external etcd that is deployed by the helm chart
	Deploy EtcdDeploy `json:"deploy,omitempty"`

	// External defines to use a self-hosted external etcd that is not deployed by the helm chart
	External EtcdExternal `json:"external,omitempty"`
}

func (e Etcd) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type EtcdExternal struct {
	// Enabled defines if the external etcd should be used.
	Enabled bool `json:"enabled,omitempty"`

	// Endpoint holds the endpoint of the external etcd server, e.g. my-example-service:2379
	Endpoint string `json:"endpoint,omitempty"`

	// TLS defines the tls configuration for the external etcd server
	TLS EtcdExternalTLS `json:"tls,omitempty"`
}

// EtcdExternalTLS defines tls for external etcd server
type EtcdExternalTLS struct {
	// CaFile is the path to the ca file
	CaFile string `json:"caFile,omitempty"`

	// CertFile is the path to the cert file
	CertFile string `json:"certFile,omitempty"`

	// KeyFile is the path to the key file
	KeyFile string `json:"keyFile,omitempty"`
}

type EtcdEmbedded struct {
	// Enabled defines if the embedded etcd should be used.
	Enabled bool `json:"enabled,omitempty" product:"pro"`

	// MigrateFromDeployedEtcd signals that vCluster should migrate from the deployed external etcd to embedded etcd.
	MigrateFromDeployedEtcd bool `json:"migrateFromDeployedEtcd,omitempty"`

	// SnapshotCount defines the number of snapshots to keep for the embedded etcd. Defaults to 10000 if less than 1.
	SnapshotCount int `json:"snapshotCount,omitempty"`

	// ExtraArgs are additional arguments to pass to the embedded etcd.
	ExtraArgs []string `json:"extraArgs,omitempty"`
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

	// Security defines pod or container security context.
	Security ControlPlaneSecurity `json:"security,omitempty"`

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

	// Affinity is the affinity to apply to the pod.
	Affinity map[string]interface{} `json:"affinity,omitempty"`

	// Tolerations are the tolerations to apply to the pod.
	Tolerations []map[string]interface{} `json:"tolerations,omitempty"`

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
	// Deprecated: Use ControlPlane.Distro.K8S.Scheduler instead.
	VirtualScheduler EnableSwitch `json:"virtualScheduler,omitempty"`

	// ServiceAccount specifies options for the vCluster control plane service account.
	ServiceAccount ControlPlaneServiceAccount `json:"serviceAccount,omitempty"`

	// WorkloadServiceAccount specifies options for the service account that will be used for the workloads that run within the virtual cluster.
	WorkloadServiceAccount ControlPlaneWorkloadServiceAccount `json:"workloadServiceAccount,omitempty"`

	// HeadlessService specifies options for the headless service used for the vCluster StatefulSet.
	HeadlessService ControlPlaneHeadlessService `json:"headlessService,omitempty"`

	// Konnectivity holds dedicated konnectivity configuration. This is only available when privateNodes.enabled is true.
	Konnectivity Konnectivity `json:"konnectivity,omitempty"`

	// Registry allows enabling an embedded docker image registry in vCluster. This is useful for air-gapped environments or when you don't have a public registry available to distribute images.
	Registry Registry `json:"registry,omitempty"`

	// CloudControllerManager holds configuration for the embedded cloud controller manager. This is only available when private nodes are enabled.
	// The cloud controller manager is responsible for setting the node's ip addresses as well as the provider id for the node and other node metadata.
	CloudControllerManager CloudControllerManager `json:"cloudControllerManager,omitempty"`

	// GlobalMetadata is metadata that will be added to all resources deployed by Helm.
	GlobalMetadata ControlPlaneGlobalMetadata `json:"globalMetadata,omitempty"`

	// KubeVip holds configuration for embedded kube-vip that announces the virtual cluster endpoint IP on layer 2.
	KubeVip KubeVip `json:"kubeVip,omitempty"`

	// PodDisruptionBudget limits how many pods of an application can be voluntarily disrupted at once
	// to ensure availability during maintenance or scaling operations.
	PodDisruptionBudget PodDisruptionBudget `json:"podDisruptionBudget,omitempty"`
}

type Registry struct {
	// Enabled defines if the embedded registry should be enabled.
	Enabled bool `json:"enabled,omitempty"`

	// AnonymousPull allows enabling anonymous pull for the embedded registry. This allows anybody to pull images from the registry without authentication.
	AnonymousPull bool `json:"anonymousPull,omitempty"`

	// Config is the regular docker registry config. See https://distribution.github.io/distribution/about/configuration/ for more details.
	Config interface{} `json:"config,omitempty"`
}

type ControlPlaneHeadlessService struct {
	// Annotations are extra annotations for this resource.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are extra labels for this resource.
	Labels map[string]string `json:"labels,omitempty"`
}

type Proxy struct {
	// CustomResources is a map of resource keys (format: "kind.apiGroup/version") to proxy configuration
	CustomResources map[string]CustomResourceProxy `json:"customResources,omitempty"`
}

type AccessResourcesMode string

const (
	AccessResourcesModeOwned AccessResourcesMode = "owned"
	AccessResourcesModeAll   AccessResourcesMode = "all"
)

type CustomResourceProxy struct {
	// Enabled defines if this resource proxy should be enabled
	Enabled bool `json:"enabled,omitempty"`

	// TargetVirtualCluster is the target virtual cluster for the custom resource proxy
	TargetVirtualCluster VirtualClusterRef `json:"targetVirtualCluster"`

	// AccessResources defines which resources should be accessible in the proxy.
	AccessResources AccessResourcesMode `json:"accessResources,omitempty"`
}

// VirtualClusterRef is a reference to a virtual cluster within the platform.
type VirtualClusterRef struct {
	// Name is the name of the target virtual cluster.
	Name string `json:"name" jsonschema:"required,minLength=1,pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`

	// Project is the project of the target virtual cluster. If empty, defaults to the same project as the source vCluster.
	Project string `json:"project,omitempty" jsonschema:"pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"`
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
	Name string `protobuf:"bytes,1,opt,name=name" json:"name"`

	// Mounted read-only if true, read-write otherwise (false or unspecified).
	// Defaults to false.
	ReadOnly bool `protobuf:"varint,2,opt,name=readOnly" json:"readOnly,omitempty"`

	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	MountPath string `protobuf:"bytes,3,opt,name=mountPath" json:"mountPath"`

	// Path within the volume from which the container's volume should be mounted.
	// Defaults to "" (volume's root).
	SubPath string `protobuf:"bytes,4,opt,name=subPath" json:"subPath,omitempty"`

	// mountPropagation determines how mounts are propagated from the host
	// to container and the other way around.
	// When not set, MountPropagationNone is used.
	// This field is beta in 1.10.
	MountPropagation *string `protobuf:"bytes,5,opt,name=mountPropagation,casttype=MountPropagationMode" json:"mountPropagation,omitempty"`

	// Expanded path within the volume from which the container's volume should be mounted.
	// Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.
	// Defaults to "" (volume's root).
	// SubPathExpr and SubPath are mutually exclusive.
	SubPathExpr string `protobuf:"bytes,6,opt,name=subPathExpr" json:"subPathExpr,omitempty"`
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
	LivenessProbe LivenessProbe `json:"livenessProbe,omitempty"`

	// ReadinessProbe specifies if the readiness probe for the container should be enabled
	ReadinessProbe ReadinessProbe `json:"readinessProbe,omitempty"`

	// StartupProbe specifies if the startup probe for the container should be enabled
	StartupProbe StartupProbe `json:"startupProbe,omitempty"`
}

// LivenessProbe defines the configuration for the liveness probe.
// A liveness probe checks if the container is still running.
// If it fails, Kubernetes will restart the container.
type LivenessProbe struct {
	EnableSwitch

	// Number of consecutive failures for the probe to be considered failed
	FailureThreshold int `json:"failureThreshold,omitempty"`

	// Time (in seconds) to wait before starting the liveness probe
	InitialDelaySeconds int `json:"initialDelaySeconds,omitempty"`

	// Maximum duration (in seconds) that the probe will wait for a response.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Frequency (in seconds) to perform the probe
	PeriodSeconds int `json:"periodSeconds,omitempty"`
}

// ReadinessProbe defines the configuration for the readiness probe.
// A readiness probe checks if the container is ready to accept traffic.
// If it fails, Kubernetes removes the pod from the Service endpoints.
type ReadinessProbe struct {
	EnableSwitch

	// Number of consecutive failures for the probe to be considered failed
	FailureThreshold int `json:"failureThreshold,omitempty"`

	// Maximum duration (in seconds) that the probe will wait for a response.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Frequency (in seconds) to perform the probe
	PeriodSeconds int `json:"periodSeconds,omitempty"`
}

// StartupProbe defines the configuration for the startup probe.
// A startup probe checks if the application within the container has started.
// While the startup probe is failing, other probes are disabled.
type StartupProbe struct {
	EnableSwitch

	// Number of consecutive failures allowed before failing the pod
	FailureThreshold int `json:"failureThreshold,omitempty"`

	// Maximum duration (in seconds) that the probe will wait for a response.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// Frequency (in seconds) to perform the probe
	PeriodSeconds int `json:"periodSeconds,omitempty"`
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

type PodDisruptionBudget struct {
	// Enabled defines if the pod disruption budget should be enabled.
	Enabled bool `json:"enabled,omitempty"`
	// MinAvailable describes the minimal number or percentage of available pods.
	MinAvailable interface{} `json:"minAvailable,omitempty"`
	// MaxUnavailable describes the minimal number or percentage of unavailable pods.
	MaxUnavailable interface{} `json:"maxUnavailable,omitempty"`
	// UnhealthyPodEvictionPolicy defines the criteria when unhealthy pods should be considered for eviction.
	// Currently supported values are:
	// * IfHealthyBudget - pods that are in the Running phase but not yet healthy are considered disrupted
	//	and may be evicted even if the PodDisruptionBudget criteria are not met.
	// * AlwaysAllow - pods that are in the Running phase but not yet healthy are considered disrupted
	//	and can be evicted regardless of whether the criteria in a PDB is met.
	UnhealthyPodEvictionPolicy *policyv1.UnhealthyPodEvictionPolicyType `json:"unhealthyPodEvictionPolicy,omitempty"`
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

type LimitRange struct {
	// Enabled defines if the limit range should be deployed by vCluster. "auto" means that if resourceQuota is enabled,
	// the limitRange will be enabled as well.
	Enabled StrBool `json:"enabled,omitempty" jsonschema:"oneof_type=string;boolean"`

	// Default are the default limits for the limit range
	Default map[string]interface{} `json:"default,omitempty"`

	// DefaultRequest are the default request options for the limit range
	DefaultRequest map[string]interface{} `json:"defaultRequest,omitempty"`

	// Max are the max limits for the limit range
	Max map[string]interface{} `json:"max,omitempty"`

	// Min are the min limits for the limit range
	Min map[string]interface{} `json:"min,omitempty"`

	LabelsAndAnnotations `json:",inline"`
}

type NetworkPolicy struct {
	// Enabled defines if the network policy should be deployed by vCluster.
	Enabled bool `json:"enabled,omitempty"`

	LabelsAndAnnotations `json:",inline"`

	// FallbackDNS is the fallback DNS server to use if the virtual cluster does not have a DNS server.
	FallbackDNS string `json:"fallbackDns,omitempty"`

	// ControlPlane network policy rules
	ControlPlane NetworkPolicyControlPlane `json:"controlPlane,omitempty"`

	// Workload network policy rules
	Workload NetworkPolicyWorkload `json:"workload,omitempty"`
}

type NetworkPolicyControlPlane struct {
	// Ingress rules for the vCluster control plane.
	Ingress []NetworkPolicyIngressRule `json:"ingress,omitempty"`

	// Egress rules for the vCluster control plane.
	Egress []NetworkPolicyEgressRule `json:"egress,omitempty"`
}

type NetworkPolicyWorkload struct {
	// PublicEgress holds the public outgoing connections options for the vCluster workloads.
	PublicEgress NetworkPolicyWorkloadPublicEgress `json:"publicEgress,omitempty"`

	// Ingress rules for the vCluster workloads.
	Ingress []NetworkPolicyIngressRule `json:"ingress,omitempty"`

	// Egress rules for the vCluster workloads.
	Egress []NetworkPolicyEgressRule `json:"egress,omitempty"`
}

// NetworkPolicyIngressRule describes a particular set of traffic that is allowed to the pods
// matched by a NetworkPolicySpec's podSelector. The traffic must match both ports and from.
type NetworkPolicyIngressRule struct {
	// ports is a list of ports which should be made accessible on the pods selected for
	// this rule. Each item in this list is combined using a logical OR. If this field is
	// empty or missing, this rule matches all ports (traffic not restricted by port).
	// If this field is present and contains at least one item, then this rule allows
	// traffic only if the traffic matches at least one port in the list.
	// +optional
	// +listType=atomic
	Ports []NetworkPolicyPort `json:"ports,omitempty"`

	// from is a list of sources which should be able to access the pods selected for this rule.
	// Items in this list are combined using a logical OR operation. If this field is
	// empty or missing, this rule matches all sources (traffic not restricted by
	// source). If this field is present and contains at least one item, this rule
	// allows traffic only if the traffic matches at least one item in the from list.
	// +optional
	// +listType=atomic
	From []NetworkPolicyPeer `json:"from,omitempty"`
}

// NetworkPolicyEgressRule describes a particular set of traffic that is allowed out of pods
// matched by a NetworkPolicySpec's podSelector. The traffic must match both ports and to.
// This type is beta-level in 1.8
type NetworkPolicyEgressRule struct {
	// ports is a list of destination ports for outgoing traffic.
	// Each item in this list is combined using a logical OR. If this field is
	// empty or missing, this rule matches all ports (traffic not restricted by port).
	// If this field is present and contains at least one item, then this rule allows
	// traffic only if the traffic matches at least one port in the list.
	// +optional
	// +listType=atomic
	Ports []NetworkPolicyPort `json:"ports,omitempty"`

	// to is a list of destinations for outgoing traffic of pods selected for this rule.
	// Items in this list are combined using a logical OR operation. If this field is
	// empty or missing, this rule matches all destinations (traffic not restricted by
	// destination). If this field is present and contains at least one item, this rule
	// allows traffic only if the traffic matches at least one item in the to list.
	// +optional
	// +listType=atomic
	To []NetworkPolicyPeer `json:"to,omitempty"`
}

// NetworkPolicyPort describes a port to allow traffic on
type NetworkPolicyPort struct {
	// protocol represents the protocol (TCP, UDP, or SCTP) which traffic must match.
	// If not specified, this field defaults to TCP.
	// +optional
	Protocol *string `json:"protocol,omitempty"`

	// port represents the port on the given protocol. This can either be a numerical or named
	// port on a pod. If this field is not provided, this matches all port names and
	// numbers.
	// If present, only traffic on the specified protocol AND port will be matched.
	// +optional
	Port interface{} `json:"port,omitempty"`

	// endPort indicates that the range of ports from port to endPort if set, inclusive,
	// should be allowed by the policy. This field cannot be defined if the port field
	// is not defined or if the port field is defined as a named (string) port.
	// The endPort must be equal or greater than port.
	// +optional
	EndPort *int32 `json:"endPort,omitempty"`
}

// NetworkPolicyPeer describes a peer to allow traffic to/from. Only certain combinations of
// fields are allowed
type NetworkPolicyPeer struct {
	// podSelector is a label selector which selects pods. This field follows standard label
	// selector semantics; if present but empty, it selects all pods.
	//
	// If namespaceSelector is also set, then the NetworkPolicyPeer as a whole selects
	// the pods matching podSelector in the Namespaces selected by NamespaceSelector.
	// Otherwise it selects the pods matching podSelector in the policy's own namespace.
	// +optional
	PodSelector *StandardLabelSelector `json:"podSelector,omitempty"`

	// namespaceSelector selects namespaces using cluster-scoped labels. This field follows
	// standard label selector semantics; if present but empty, it selects all namespaces.
	//
	// If podSelector is also set, then the NetworkPolicyPeer as a whole selects
	// the pods matching podSelector in the namespaces selected by namespaceSelector.
	// Otherwise it selects all pods in the namespaces selected by namespaceSelector.
	// +optional
	NamespaceSelector *StandardLabelSelector `json:"namespaceSelector,omitempty"`

	// ipBlock defines policy on a particular IPBlock. If this field is set then
	// neither of the other fields can be.
	// +optional
	IPBlock *IPBlock `json:"ipBlock,omitempty"`
}

type NetworkPolicyWorkloadPublicEgress struct {
	// Enabled defines if the workload public egress should be enabled or disabled.
	Enabled bool `json:"enabled,omitempty"`

	// CIDR defines the allowed workload public egress destination.
	// Valid examples are "0.0.0.0/0", "192.168.1.0/24" or "2001:db8::/64"
	CIDR string `json:"cidr,omitempty"`

	// Except is a slice of CIDRs that should not be included. Items outside the cidr range will be rejected.
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64".
	// +optional
	Except []string `json:"except,omitempty"`
}

// OutgoingConnections to be removed via legacyconfig cleanup.
type OutgoingConnections struct {
	// IPBlock describes a particular CIDR (Ex. "192.168.1.0/24","2001:db8::/64") that is allowed
	// to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs
	// that should not be included within this rule.
	IPBlock IPBlock `json:"ipBlock,omitempty"`
}

// IPBlock describes a particular CIDR (Ex. "192.168.1.0/24","2001:db8::/64") that is allowed
// to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs
// that should not be included within this rule.
type IPBlock struct {
	// CIDR defines the allowed workload public egress destination.
	// Valid examples are "0.0.0.0/0", "192.168.1.0/24" or "2001:db8::/64"
	CIDR string `json:"cidr,omitempty"`

	// Except is a slice of CIDRs that should not be included. Items outside the cidr range will be rejected.
	// Valid examples are "192.168.1.0/24" or "2001:db8::/64".
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
	ReinvocationPolicy *string `protobuf:"bytes,10,opt,name=reinvocationPolicy,casttype=ReinvocationPolicyType" json:"reinvocationPolicy,omitempty"`

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

	// EnableVolumeSnapshotRules enables all required volume snapshot rules in the Role and
	// ClusterRole.
	EnableVolumeSnapshotRules EnableAutoSwitch `json:"enableVolumeSnapshotRules,omitempty"`
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
	// Deploy allows you to configure manifests and Helm charts to deploy within the host or virtual cluster.
	Deploy ExperimentalDeploy `json:"deploy,omitempty"`

	// SyncSettings are advanced settings for the syncer controller.
	SyncSettings ExperimentalSyncSettings `json:"syncSettings,omitempty"`

	// VirtualClusterKubeConfig allows you to override distro specifics and specify where vCluster will find the required certificates and vCluster config.
	// Deprecated: Removed in 0.29.0.
	VirtualClusterKubeConfig VirtualClusterKubeConfig `json:"virtualClusterKubeConfig,omitempty"`

	// DenyProxyRequests denies certain requests in the vCluster proxy.
	DenyProxyRequests []DenyRule `json:"denyProxyRequests,omitempty" product:"pro"`

	// Proxy enables vCluster-to-vCluster proxying of resources
	Proxy Proxy `json:"proxy,omitempty"`

	// Docker allows you to configure Docker related settings when deploying a vCluster using Docker.
	Docker ExperimentalDocker `json:"docker,omitempty"`
}

func (e Experimental) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type ExperimentalSyncSettings struct {
	// SetOwner specifies if vCluster should set an owner reference on the synced objects to the vCluster service. This allows for easy garbage collection.
	SetOwner bool `json:"setOwner,omitempty"`

	// HostMetricsBindAddress is the bind address for the local manager
	HostMetricsBindAddress string `json:"hostMetricsBindAddress,omitempty"`

	// VirtualMetricsBindAddress is the bind address for the virtual manager
	VirtualMetricsBindAddress string `json:"virtualMetricsBindAddress,omitempty"`
}

func (e ExperimentalSyncSettings) JSONSchemaExtend(base *jsonschema.Schema) {
	addProToJSONSchema(base, reflect.TypeOf(e))
}

type ExperimentalDocker struct {
	ExperimentalDockerContainer `json:",inline"`

	// Enabled defines if the vCluster was deployed using Docker. This is automatically set by vCluster and should not be set by the user.
	Enabled bool `json:"enabled,omitempty"`

	// Network defines the network to use for the vCluster. If not specified, the a network will be created for the vCluster.
	Network string `json:"network,omitempty"`

	// Nodes defines the nodes of the vCluster.
	Nodes []ExperimentalDockerNode `json:"nodes,omitempty"`

	// Defines if docker images should be pulled from the host docker daemon. This prevents pulling images again and allows to
	// use purely local images. Only works if containerd image storage is used. For more information, see https://docs.docker.com/engine/storage/containerd
	RegistryProxy EnableSwitch `json:"registryProxy,omitempty"`

	// Defines if vCluster should configure load balancer services inside the vCluster. This might require
	// sudo access on the host cluster for docker desktop or rancher desktop on macos.
	LoadBalancer ExperimentalDockerLoadBalancer `json:"loadBalancer,omitempty"`
}

type ExperimentalDockerLoadBalancer struct {
	EnableSwitch `json:",inline"`

	// ForwardPorts defines if the load balancer ips should be made available locally
	// via port forwarding. This will be only done if necessary for example on macos when using docker desktop.
	ForwardPorts bool `json:"forwardPorts,omitempty"`
}

type ExperimentalDockerNode struct {
	ExperimentalDockerContainer `json:",inline"`

	// Name defines the name of the node. If not specified, a random name will be generated.
	Name string `json:"name,omitempty"`
}

type ExperimentalDockerContainer struct {
	// Image defines the image to use for the container. Defaults to ghcr.io/loft-sh/vm-container.
	Image string `json:"image,omitempty"`

	// Ports defines extra port mappings to be added to the container.
	Ports []string `json:"ports,omitempty"`

	// Volumes defines extra volumes to be added to the container.
	Volumes []string `json:"volumes,omitempty"`

	// Env defines extra environment variables to be added to the container. Use key=value.
	Env []string `json:"env,omitempty"`

	// Args defines extra arguments to be added to the docker run command of the container.
	Args []string `json:"args,omitempty"`
}

type ExperimentalDeploy struct {
	// Host defines what manifests to deploy into the host cluster
	Host ExperimentalDeployHost `json:"host,omitempty"`

	// VCluster defines what manifests and charts to deploy into the vCluster
	VCluster ExperimentalDeployVCluster `json:"vcluster,omitempty"`
}

type ExperimentalDeployHost struct {
	// Manifests are raw Kubernetes manifests that should get applied within the host cluster.
	Manifests string `json:"manifests,omitempty"`

	// ManifestsTemplate is a Kubernetes manifest template that will be rendered with vCluster values before applying it within the host cluster.
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

// PlatformConfig is a type alias for the imported vclusterconfig.Platform type.
// This is kept for backwards compatibility with code that uses PlatformConfig.
// Deprecated: Use vclusterconfig.Platform directly.
type PlatformConfig = vclusterconfig.Platform

// PlatformAPIKey is a type alias for the imported vclusterconfig.PlatformAPIKey type.
// Deprecated: Use vclusterconfig.PlatformAPIKey directly.
type PlatformAPIKey = vclusterconfig.PlatformAPIKey

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
	APIGroups []string `protobuf:"bytes,1,rep,name=apiGroups" json:"apiGroups,omitempty"`

	// APIVersions is the API versions the resources belong to. '*' is all versions.
	APIVersions []string `protobuf:"bytes,2,rep,name=apiVersions" json:"apiVersions,omitempty"`

	// Resources is a list of resources this rule applies to.
	Resources []string `protobuf:"bytes,3,rep,name=resources" json:"resources,omitempty"`

	// Scope specifies the scope of this rule.
	Scope *string `protobuf:"bytes,4,rep,name=scope" json:"scope,omitempty"`

	// Verb is the kube verb associated with the request for API requests, not the http verb. This includes things like list and watch.
	// For non-resource requests, this is the lowercase http verb.
	// If '*' is present, the length of the slice must be one.
	Verbs []string `json:"operations,omitempty"`
}

// PodDNSConfig defines the DNS parameters of a pod in addition to
// those generated from DNSPolicy.
type PodDNSConfig struct {
	// A list of DNS name server IP addresses.
	// This will be appended to the base nameservers generated from DNSPolicy.
	// Duplicated nameservers will be removed.
	// +optional
	// +listType=atomic
	Nameservers []string `protobuf:"bytes,1,rep,name=nameservers" json:"nameservers,omitempty"`
	// A list of DNS search domains for host-name lookup.
	// This will be appended to the base search paths generated from DNSPolicy.
	// Duplicated search paths will be removed.
	// +optional
	// +listType=atomic
	Searches []string `protobuf:"bytes,2,rep,name=searches" json:"searches,omitempty"`
	// A list of DNS resolver options.
	// This will be merged with the base options generated from DNSPolicy.
	// Duplicated entries will be removed. Resolution options given in Options
	// will override those that appear in the base DNSPolicy.
	// +optional
	// +listType=atomic
	Options []PodDNSConfigOption `protobuf:"bytes,3,rep,name=options" json:"options,omitempty"`
}

// PodDNSConfigOption defines DNS resolver options of a pod.
type PodDNSConfigOption struct {
	// Required.
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// +optional
	Value *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

// DNSPolicy defines how a pod's DNS will be configured.
// +enum
type DNSPolicy string

const (
	// DNSClusterFirstWithHostNet indicates that the pod should use cluster DNS
	// first, if it is available, then fall back on the default
	// (as determined by kubelet) DNS settings.
	DNSClusterFirstWithHostNet DNSPolicy = "ClusterFirstWithHostNet"

	// DNSClusterFirst indicates that the pod should use cluster DNS
	// first unless hostNetwork is true, if it is available, then
	// fall back on the default (as determined by kubelet) DNS settings.
	DNSClusterFirst DNSPolicy = "ClusterFirst"

	// DNSDefault indicates that the pod should use the default (as
	// determined by kubelet) DNS settings.
	DNSDefault DNSPolicy = "Default"

	// DNSNone indicates that the pod should use empty DNS settings. DNS
	// parameters such as nameservers and search paths should be defined via
	// DNSConfig.
	DNSNone DNSPolicy = "None"
)

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

// SleepMode holds configuration for native/workload only sleep mode
type SleepMode struct {
	// Enabled toggles the sleep mode functionality, allowing for disabling sleep mode without removing other config
	Enabled bool `json:"enabled,omitempty"`
	// Timezone represents the timezone a sleep schedule should run against, defaulting to UTC if unset
	TimeZone string `json:"timeZone,omitempty"`
	// AutoSleep holds autoSleep details
	AutoSleep SleepModeAutoSleep `json:"autoSleep,omitempty"`
	// AutoWakeup holds configuration for waking the vCluster on a schedule rather than waiting for some activity.
	AutoWakeup AutoWakeup `json:"autoWakeup,omitempty"`
}

// SleepModeAutoSleep holds configuration for allowing a vCluster to sleep its workloads
// automatically
type SleepModeAutoSleep struct {
	// AfterInactivity represents how long a vCluster can be idle before workloads are automaticaly put to sleep
	AfterInactivity Duration `json:"afterInactivity,omitempty"`

	// Schedule represents a cron schedule for when to sleep workloads
	Schedule string `json:"schedule,omitempty"`

	// Exclude holds configuration for labels that, if present, will prevent a workload from going to sleep
	Exclude AutoSleepExclusion `json:"exclude,omitempty"`
}

// Duration allows for automatic Marshalling from strings like "1m" to a time.Duration
type Duration string

// MarshalJSON implements Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	dur, err := time.ParseDuration(string(d))
	if err != nil {
		return nil, err
	}
	return json.Marshal(dur.String())
}

// UnmarshalJSON implements Marshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	sval, ok := v.(string)
	if !ok {
		return errors.New("invalid duration")
	}

	_, err := time.ParseDuration(sval)
	if err != nil {
		return err
	}
	*d = Duration(sval)
	return nil
}

// AutoWakeup holds the cron schedule to wake workloads automatically
type AutoWakeup struct {
	Schedule string `json:"schedule,omitempty"`
}

// AutoSleepExclusion holds conifiguration for excluding workloads from sleeping by label(s)
type AutoSleepExclusion struct {
	Selector LabelSelector `json:"selector,omitempty"`
}

// Logging holds the log encoding details
type Logging struct {
	// Encoding specifies the format of vCluster logs, it can either be json or console.
	Encoding string `json:"encoding,omitempty"`
}
