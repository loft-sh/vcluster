package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	NodeProviderTypeBCM        string = "bcm"
	NodeProviderTypeKubeVirt   string = "kubeVirt"
	NodeProviderTypeTerraform  string = "terraform"
	NodeProviderTypeClusterAPI string = "clusterAPI"
	NodeProviderTypeMetal3     string = "metal3"
	NodeProviderTypeNICo       string = "nico"

	// NodeProviderConditionTypeInitialized is the condition that indicates if the node provider is initialized.
	NodeProviderConditionTypeInitialized = "Initialized"

	// NodeProviderConditionTypeDeployed is the condition that indicates if infrastructure components are deployed.
	NodeProviderConditionTypeDeployed = "Deployed"

	// NodeProviderReasonFeatureNotAllowed is set on the Initialized condition when the license
	// does not cover the provider type this NodeProvider configures. The provider is left
	// running as-is; nothing new is initialized until the license changes.
	NodeProviderReasonFeatureNotAllowed = "FeatureNotAllowed"
)

var (
	NodeProviderConditions = []agentstoragev1.ConditionType{
		NodeProviderConditionTypeInitialized,
		NodeProviderConditionTypeDeployed,
	}
)

// NodeProviderPhase defines the phase of the NodeProvider
type NodeProviderPhase string

const (
	// NodeProviderPhasePending is the initial state of a NodeProvider.
	NodeProviderPhasePending NodeProviderPhase = "Pending"
	// NodeProviderPhaseAvailable means the underlying node has been successfully provisioned.
	NodeProviderPhaseAvailable NodeProviderPhase = "Available"
	// NodeProviderPhaseFailed means the provisioning process has failed.
	NodeProviderPhaseFailed NodeProviderPhase = "Failed"
	// NodeProvider specific label
	NodeProvidedManagedTypeIndicatorLabel     = "autoscaling.loft.sh/managed-by"
	NodeProviderManagedTypeMetadataAnnotation = "autoscaling.loft.sh/managed-metadata"

	// NodeTypeMaxCapacityAnnotation is the annotation used to store the maximum capacity of a NodeType
	NodeTypeMaxCapacityAnnotation = "autoscaling.loft.sh/max-capacity"

	// BCM specific annotations
	NodeTypeNodesAnnotation      = "bcm.loft.sh/nodes"
	NodeTypeNodeGroupsAnnotation = "bcm.loft.sh/node-groups"

	// KubeVirt specific annotations
	NodeTypeVMTemplateAnnotation = "kubevirt.vcluster.com/vm-template"

	// ClusterAPI specific annotations
	NodeTypeClusterAPIInfrastructureMachineTemplateAnnotation = "clusterapi.loft.sh/infrastructure-machine-template"
	NodeTypeClusterAPIBootstrapConfigTemplateAnnotation       = "clusterapi.loft.sh/bootstrap-config-template"

	// Properties
	NodeProviderCCMEnabledProperty = "vcluster.com/ccm-enabled"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeProvider holds the information of a node provider config.
// This resource defines various ways a node can be provisioned or configured.
// +k8s:openapi-gen=true
type NodeProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeProviderSpec   `json:"spec,omitempty"`
	Status NodeProviderStatus `json:"status,omitempty"`
}

func (a *NodeProvider) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodeProvider) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// NodeProviderSpec defines the desired state of NodeProvider.
// Only one of the provider types (Pods, BCM, Kubevirt) should be specified at a time.
type NodeProviderSpec struct {
	// Properties are global properties that are applied to all node claims and environments managed by this provider.
	// +optional
	Properties map[string]string `json:"properties,omitempty"`

	// BCM configures a node provider for BCM Bare Metal Cloud environments.
	// +optional
	BCM *NodeProviderBCM `json:"bcm,omitempty"`

	// Kubevirt configures a node provider using KubeVirt, enabling virtual machines
	// to be provisioned as nodes within a vCluster.
	// +optional
	KubeVirt *NodeProviderKubeVirt `json:"kubeVirt,omitempty"`

	// Terraform configures a node provider using Terraform, enabling nodes to be provisioned using Terraform.
	// +optional
	Terraform *NodeProviderTerraform `json:"terraform,omitempty"`

	// ClusterAPI configures a node provider using Cluster API, enabling nodes to be provisioned using Cluster API.
	// This requires the vCluster to be deployed with Cluster API as well.
	// +optional
	ClusterAPI *NodeProviderClusterAPI `json:"clusterAPI,omitempty"`

	// Metal3 configures a node provider using metal3.io BareMetalHost resources.
	// +optional
	Metal3 *NodeProviderMetal3 `json:"metal3,omitempty"`

	// NICo configures a node provider backed by the NVIDIA Infra Controller
	// (NICo) REST API.
	// +optional
	NICo *NodeProviderNICo `json:"nico,omitempty"`

	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

type NodeProviderClusterAPI struct {
	ClusterAPIObjects `json:",inline"`

	// ClusterRef is a reference to connected control plane cluster in which KubeVirt operator is running
	ClusterRef NodeProviderClusterRef `json:"clusterRef,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []ClusterAPINodeTypeSpec `json:"nodeTypes,omitempty"`
}

// NodeProviderBCMSpec defines the configuration for a BCM node provider.
type NodeProviderBCM struct {
	// SecretRef is a reference to secret with keys for BCM auth.
	SecretRef *NamespacedRef `json:"secretRef"`

	// Endpoint is an address for head node.
	Endpoint string `json:"endpoint"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []BCMNodeTypeSpec `json:"nodeTypes,omitempty"`
}

// NodeProviderNICo configures a node provider backed by the NVIDIA Infra
// Controller (NICo) REST API. Platform workloads use the provider org.
// Tenant workloads use the org from their Tenant's nico.vcluster.com/org annotation.
type NodeProviderNICo struct {
	// Endpoint is the base URL (scheme + host + optional port) of the NICo REST
	// API. The /v2/org/{org}/nico path is appended by the client.
	Endpoint string `json:"endpoint"`

	// Org is the NICo organization used for provider-scoped API calls (the
	// /v2/org/{org}/nico path) and stamped as the provider token's organization
	// claim. Optional; defaults to "vcluster-autonodes".
	// +optional
	Org string `json:"org,omitempty"`

	// SiteID is the NICo site UUID this provider operates against.
	// +optional
	SiteID string `json:"siteId,omitempty"`

	// SiteIPBlockID adopts an existing NICo site-level parent IPBlock. Mutually
	// exclusive with SiteIPBlockCIDR.
	// +optional
	SiteIPBlockID string `json:"siteIPBlockID,omitempty"`

	// SiteIPBlockCIDR creates the NICo site-level parent IPBlock with this CIDR.
	// Mutually exclusive with SiteIPBlockID.
	// +optional
	SiteIPBlockCIDR string `json:"siteIPBlockCIDR,omitempty"`

	// InstanceTypeIDs is an allow-list of NICo InstanceType IDs to surface as
	// NodeTypes. Empty surfaces all.
	// +optional
	InstanceTypeIDs []string `json:"instanceTypeIds,omitempty"`

	// Identity selects how the platform authenticates to the NICo REST API.
	// Platform-issued JWTs are currently the only supported source, so
	// identity.platformIssued.enabled must be true. The block is structured so
	// additional identity sources can be added later.
	// +optional
	Identity *NICoIdentity `json:"identity,omitempty"`

	// InsecureSkipTLSVerify disables TLS certificate verification against the
	// NICo endpoint. Intended for development and test only.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`

	// NetworkMode selects how tenant networking is provisioned at this site.
	// Defaults to fnn.
	// +optional
	// +kubebuilder:validation:Enum=fnn;flat
	NetworkMode NICoNetworkMode `json:"networkMode,omitempty"`
}

// NICoNetworkMode selects the NICo network virtualization a site provisions.
// +enum
type NICoNetworkMode string

const (
	// NICoNetworkModeFNN configures FNN VPCs with explicit interfaces and VPC prefixes.
	NICoNetworkModeFNN NICoNetworkMode = "fnn"

	// NICoNetworkModeFlat configures FLAT VPCs with automatic interface assignment.
	NICoNetworkModeFlat NICoNetworkMode = "flat"
)

// NICoIdentity selects the source of the tokens the platform uses to
// authenticate to the NICo REST API. Exactly one source is configured; today
// only platform-issued tokens are supported.
type NICoIdentity struct {
	// PlatformIssued authenticates with short-TTL RS256 JWTs that the platform
	// signs with its own OIDC key and issuer and NICo verifies against the
	// platform's OIDC JWKS.
	PlatformIssued NICoPlatformIssued `json:"platformIssued"`
}

// NICoPlatformIssued configures platform-issued JWT authentication to NICo.
type NICoPlatformIssued struct {
	// Enabled turns on platform-issued JWT authentication. It must be true, since
	// platform-issued tokens are currently the only supported identity source.
	Enabled bool `json:"enabled"`
}

type NodeProviderTerraform struct {
	// NodeTemplate is the template to use for this node provider.
	NodeTemplate *TerraformTemplate `json:"nodeTemplate,omitempty"`

	// NetworkEnvironmentTemplate is the template to use for this network environment.
	NetworkEnvironmentTemplate *TerraformNetworkEnvironmentTemplate `json:"networkEnvironmentTemplate,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []TerraformNodeTypeSpec `json:"nodeTypes,omitempty"`
}

type NamedNodeTypeSpec struct {
	NodeTypeSpec `json:",inline"`

	// Name is the name of this node type.
	Name string `json:"name"`

	// Metadata holds metadata to add to this managed NodeType.
	Metadata ManagedNodeTypeObjectMeta `json:"metadata,omitempty"`
}

type ManagedNodeTypeObjectMeta struct {
	// Labels holds labels to add to this managed NodeType.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations holds annotations to add to this managed NodeType.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TerraformNetworkEnvironmentTemplate struct {
	// Deprecated: Use Infrastructure instead.
	TerraformTemplate `json:",inline"`

	// Infrastructure is the infrastructure template to use for this network environment.
	Infrastructure *TerraformTemplate `json:"infrastructure,omitempty"`
}

type TerraformTemplate struct {
	// Inline is the inline template to use for this node type.
	Inline string `json:"inline,omitempty"`

	// Git is the git repository to use for this node type.
	Git *TerraformTemplateSourceGit `json:"git,omitempty"`

	// Timeout is the timeout to use for the terraform operations. Defaults to 60m.
	Timeout string `json:"timeout,omitempty"`
}

type TerraformTemplateSourceGit struct {
	// Repository is the repository to clone
	Repository string `json:"repository,omitempty"`

	// Branch is the branch to use
	Branch string `json:"branch,omitempty"`

	// Commit is the commit SHA to checkout
	Commit string `json:"commit,omitempty"`

	// Tag is the tag reference to checkout
	Tag string `json:"tag,omitempty"`

	// SubPath is the subpath in the repo to use
	SubPath string `json:"subPath,omitempty"`

	// Credentials is the reference to a secret containing the username and password for the git repository.
	Credentials *SecretRef `json:"credentials,omitempty"`

	// FetchInterval is the interval to use for refetching the git repository. Defaults to 5m. Refetching only checks for remote changes but does not do a complete repull.
	FetchInterval string `json:"fetchInterval,omitempty"`

	// ExtraEnv is the extra environment variables to use for the clone
	ExtraEnv []string `json:"extraEnv,omitempty"`
}

type TerraformNodeTypeSpec struct {
	NamedNodeTypeSpec `json:",inline"`

	// NodeTemplate is the template to use for this node type.
	NodeTemplate *TerraformTemplate `json:"nodeTemplate,omitempty"`

	// MaxCapacity is the maximum number of nodes that can be created for this NodeType.
	MaxCapacity int `json:"maxCapacity,omitempty"`
}

type BCMNodeTypeSpec struct {
	NamedNodeTypeSpec `json:",inline"`

	// Nodes specifies nodes.
	Nodes []string `json:"nodes,omitempty"`

	// NodeGroups is the name of the node groups to use for this provider.
	NodeGroups []string `json:"nodeGroups,omitempty"`
}

type NamespacedRef struct {
	// Name is the name of this resource
	Name string `json:"name"`
	// Namespace is the namespace of this resource
	Namespace string `json:"namespace"`
}

type ClusterAPINodeTypeSpec struct {
	NamedNodeTypeSpec `json:",inline"`
	ClusterAPIObjects `json:",inline"`

	// MergeInfrastructureMachineTemplate will be merged into base InfrastructureMachine template for this NodeProvider.
	// This allows overwriting of specific fields from top level template by individual NodeTypes
	// This is mutually exclusive with InfrastructureMachineTemplate
	MergeInfrastructureMachineTemplate *runtime.RawExtension `json:"mergeInfrastructureMachineTemplate,omitempty"`

	// MergeBootstrapConfigTemplate will be merged into base BootstrapConfig template for this NodeProvider.
	// This allows overwriting of specific fields from top level template by individual NodeTypes
	// This is mutually exclusive with BootstrapConfigTemplate
	MergeBootstrapConfigTemplate *runtime.RawExtension `json:"mergeBootstrapConfigTemplate,omitempty"`

	// MaxCapacity is the maximum number of nodes that can be created for this NodeType.
	MaxCapacity int `json:"maxCapacity,omitempty"`
}

type ClusterAPIObjects struct {
	// InfrastructureMachineTemplate is a template for the infrastructure machine, e.g. AWSMachine
	InfrastructureMachineTemplate *runtime.RawExtension `json:"infrastructureMachineTemplate,omitempty"`

	// BootstrapConfigTemplate is a template for the bootstrap config. Currently only KubeadmConfig is supported.
	BootstrapConfigTemplate *runtime.RawExtension `json:"bootstrapConfigTemplate,omitempty"`
}

// KubeVirtNodeTypeSpec defines single NodeType spec for KubeVirt provider type.
type KubeVirtNodeTypeSpec struct {
	NamedNodeTypeSpec `json:",inline"`

	// VirtualMachineTemplate is a full KubeVirt VirtualMachine template to use for this NodeType.
	// This is mutually exclusive with MergeVirtualMachineTemplate
	VirtualMachineTemplate *runtime.RawExtension `json:"virtualMachineTemplate,omitempty"`

	// MergeVirtualMachineTemplate will be merged into base VirtualMachine template for this NodeProvider.
	// This allows overwriting of specific fields from top level template by individual NodeTypes
	// This is mutually exclusive with VirtualMachineTemplate
	MergeVirtualMachineTemplate *runtime.RawExtension `json:"mergeVirtualMachineTemplate,omitempty"`

	// MaxCapacity is the maximum number of nodes that can be created for this NodeType.
	MaxCapacity int `json:"maxCapacity,omitempty"`
}

// KubeVirtNamespaceStrategy determines in which namespace of the connected cluster
// the VirtualMachines for a NodeClaim are created.
type KubeVirtNamespaceStrategy string

const (
	// KubeVirtNamespaceStrategyProvider creates all VirtualMachines in the namespace
	// referenced by the node provider's clusterRef. This is the default.
	KubeVirtNamespaceStrategyProvider KubeVirtNamespaceStrategy = "Provider"

	// KubeVirtNamespaceStrategyVirtualCluster creates the VirtualMachines in the
	// namespace of the tenant cluster the NodeClaim belongs to.
	KubeVirtNamespaceStrategyVirtualCluster KubeVirtNamespaceStrategy = "VirtualCluster"
)

// NodeProviderKubeVirt defines the configuration for a KubeVirt node provider.
type NodeProviderKubeVirt struct {
	// ClusterRef is a reference to connected control plane cluster in which KubeVirt operator is running
	ClusterRef NodeProviderClusterRef `json:"clusterRef,omitempty"`

	// NamespaceStrategy determines in which namespace of the connected cluster the
	// VirtualMachines are created.
	// "Provider" (default) creates all VirtualMachines in clusterRef.namespace.
	// "VirtualCluster" creates the VirtualMachines in the namespace of the tenant
	// cluster the NodeClaim belongs to. If the NodeClaim cannot be traced back to a
	// tenant cluster namespace within clusterRef.cluster, clusterRef.namespace is
	// used instead.
	// +kubebuilder:validation:Enum=Provider;VirtualCluster
	// +optional
	NamespaceStrategy KubeVirtNamespaceStrategy `json:"namespaceStrategy,omitempty"`

	// Deploy configures components deployed into the connected control plane cluster.
	// +optional
	Deploy KubeVirtProviderDeployment `json:"deploy,omitempty"`

	// VirtualMachineTemplate is a KubeVirt VirtualMachine template to use by NodeTypes managed by this NodeProvider
	VirtualMachineTemplate *runtime.RawExtension `json:"virtualMachineTemplate,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []KubeVirtNodeTypeSpec `json:"nodeTypes"`
}

type KubeVirtProviderDeployment struct {
	// KubeVirt configures the KubeVirt operator deployment.
	// +optional
	KubeVirt KubeVirtDeployment `json:"kubevirt,omitempty"`

	// VClusterDeviceOperator configures the vCluster device operator deployment.
	// +optional
	VClusterDeviceOperator *VClusterDeviceOperatorDeployment `json:"vClusterDeviceOperator,omitempty"`
}

type VClusterDeviceOperatorDeployment struct {
	// Enabled controls whether the vCluster device operator is deployed into the cluster.
	Enabled bool `json:"enabled"`

	// ChartRepo overrides the Helm chart repository used to install the operator.
	// +optional
	ChartRepo string `json:"chartRepo,omitempty"`

	// Chart overrides the Helm chart name used to install the operator.
	// +optional
	Chart string `json:"chart,omitempty"`

	// Version overrides the Helm chart version used to install the operator.
	// +optional
	Version string `json:"version,omitempty"`

	// HelmValues is raw YAML that will be passed as values to the Helm chart.
	// +optional
	HelmValues string `json:"helmValues,omitempty"`
}

type KubeVirtDeployment struct {
	// Enabled controls whether the KubeVirt operator is deployed into the cluster.
	Enabled bool `json:"enabled"`

	// ChartRepo overrides the Helm chart repository used to install the KubeVirt operator.
	// +optional
	ChartRepo string `json:"chartRepo,omitempty"`

	// Chart overrides the Helm chart name used to install the KubeVirt operator.
	// +optional
	Chart string `json:"chart,omitempty"`

	// Version overrides the Helm chart version used to install the KubeVirt operator.
	// +optional
	Version string `json:"version,omitempty"`

	// HelmValues is raw YAML that will be passed as values to the KubeVirt Helm chart.
	// +optional
	HelmValues string `json:"helmValues,omitempty"`
}

type NodeProviderClusterRef struct {
	// Cluster is the connected cluster the VMs will be created in
	Cluster string `json:"cluster"`

	// Namespace is the namespace inside the connected cluster holding VMs
	Namespace string `json:"namespace,omitempty"`
}

type MultusDeployment struct {
	// Enabled controls whether Multus CNI is deployed into the cluster.
	Enabled bool `json:"enabled"`

	// HelmValues is raw YAML that will be passed as values to the Multus Helm chart.
	// +optional
	HelmValues string `json:"helmValues,omitempty"`
}

type DHCPDeployment struct {
	// Enabled controls whether the DHCP server is deployed into the cluster.
	Enabled bool `json:"enabled"`

	// ChartRepo overrides the Helm chart repository used to install the DHCP server.
	// +optional
	ChartRepo string `json:"chartRepo,omitempty"`

	// Chart overrides the Helm chart name used to install the DHCP server.
	// +optional
	Chart string `json:"chart,omitempty"`

	// Version overrides the Helm chart version used to install the DHCP server.
	// +optional
	Version string `json:"version,omitempty"`

	// HelmValues is raw YAML that will be passed as values to the DHCP Helm chart.
	// +optional
	HelmValues string `json:"helmValues,omitempty"`
}

type Metal3Deployment struct {
	// Enabled controls whether Metal3 and Ironic are deployed into the cluster.
	Enabled bool `json:"enabled"`

	// ChartRepo overrides the Helm chart repository used to install Metal3.
	// +optional
	ChartRepo string `json:"chartRepo,omitempty"`

	// Chart overrides the Helm chart name used to install Metal3.
	// +optional
	Chart string `json:"chart,omitempty"`

	// Version overrides the Helm chart version used to install Metal3.
	// +optional
	Version string `json:"version,omitempty"`

	// HelmValues is raw YAML that will be passed as values to the Metal3 Helm chart.
	// +optional
	HelmValues string `json:"helmValues,omitempty"`
}

type Metal3ProviderDeployment struct {
	// Multus configures the Multus CNI deployment.
	// +optional
	Multus MultusDeployment `json:"multus,omitempty"`

	// DHCP configures the DHCP server deployment.
	// +optional
	DHCP DHCPDeployment `json:"dhcp,omitempty"`

	// Metal3 configures the Metal3/Ironic deployment.
	// +optional
	Metal3 Metal3Deployment `json:"metal3,omitempty"`
}

type NodeProviderMetal3 struct {
	// ClusterRef is a reference to connected control plane cluster in which KubeVirt operator is running
	ClusterRef NodeProviderClusterRef `json:"clusterRef,omitempty"`

	Deploy Metal3ProviderDeployment `json:"deploy,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []Metal3NodeTypeSpec `json:"nodeTypes,omitempty"`

	// NeutronEnabled turns on the neutron network shim for this provider: BareMetalHost network
	// attachments are allocated by the platform and reconciled through ConfigMaps instead of
	// being written directly as DHCP annotations.
	// +optional
	NeutronEnabled bool `json:"neutronEnabled,omitempty"`

	// Netris attaches BareMetalHosts to a Netris server cluster on provisioning.
	// +optional
	Netris *NodeProviderMetal3Netris `json:"netris,omitempty"`

	// NetBox imports machines from a NetBox inventory: every device carrying the
	// configured tag becomes a Machine of this provider and a BareMetalHost in
	// the provider's cluster. Several providers may import from the same NetBox
	// with different tags, or from different NetBox instances.
	// +optional
	NetBox *NodeProviderMetal3NetBox `json:"netBox,omitempty"`
}

type NodeProviderMetal3Netris struct {
	// SecretRef references a Secret with keys url, username and password for the Netris API.
	SecretRef *NamespacedRef `json:"secretRef"`
}

// NodeProviderMetal3NetBox configures the NetBox -> Machine -> BareMetalHost
// import of a metal3 provider. The connector holds how to reach NetBox; this
// holds what to take from it. Each machine's BMC address and login are read
// from the device itself (its management IP and the bmc_username and
// bmc_password custom fields); the protocol Ironic speaks to it is the
// provider's to set, since NetBox has no field for it.
type NodeProviderMetal3NetBox struct {
	// SecretRef references the NetBox connector Secret (labelled
	// loft.sh/connector-type=netbox) to read devices from.
	SecretRef *NamespacedRef `json:"secretRef"`

	// Tag is the slug of the NetBox tag that marks a device for import. A device
	// carrying it is imported automatically and kept in sync; removing the tag
	// stops the sync but never deprovisions a machine that is in use. Defaults
	// to "vcluster-sync".
	// +optional
	Tag string `json:"tag,omitempty"`

	// CustomFields names the device custom fields the BMC login is read from,
	// for deployments that already keep it under other names. Unset names keep
	// their defaults.
	// +optional
	CustomFields *NodeProviderMetal3NetBoxCustomFields `json:"customFields,omitempty"`

	// AddressTemplate is a Go template rendering the BareMetalHost bmc.address
	// from the device: the protocol and path Ironic dials, which NetBox records
	// no field for. {{ .Address }} is the management IP as a URL host
	// (IPv6 bracketed), {{ .IP }} the bare IP, {{ .Device }} the NetBox device
	// name, {{ .Manufacturer }} and {{ .DeviceType }} their slugs, {{ .Serial }}
	// the serial. Defaults to
	// "redfish://{{ .Address }}/redfish/v1/Systems/1", which is
	// what Lenovo, HPE and Supermicro BMCs answer; Dell iDRACs need
	// "idrac-redfish://{{ .Address }}/redfish/v1/Systems/System.Embedded.1".
	// +optional
	AddressTemplate string `json:"addressTemplate,omitempty"`

	// BareMetalHostTemplate is merged into every BareMetalHost the import
	// creates. Its labels and annotations are kept current on existing hosts;
	// its spec is applied when a host is created and wins over what the import
	// generates, so it can set rootDeviceHints, disable BMC certificate
	// verification, or point bmc.credentialsName at a Secret you manage. A
	// template that sets bmc.credentialsName makes the BMC login custom fields
	// optional, and one that sets bootMACAddress makes the NetBox boot MAC
	// optional.
	// +optional
	BareMetalHostTemplate *NodeProviderMetal3NetBoxBareMetalHostTemplate `json:"bareMetalHostTemplate,omitempty"`
}

// NodeProviderMetal3NetBoxBareMetalHostTemplate is the part of an imported
// BareMetalHost the provider dictates rather than NetBox.
type NodeProviderMetal3NetBoxBareMetalHostTemplate struct {
	// Metadata holds labels and annotations set on every imported host, on top
	// of the ones the import derives from NetBox.
	// +optional
	Metadata TemplateMetadata `json:"metadata,omitempty"`

	// Spec is merged into the generated BareMetalHost spec, with the
	// template's values taking precedence field by field.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Spec *runtime.RawExtension `json:"spec,omitempty"`
}

// NodeProviderMetal3NetBoxCustomFields names the NetBox device custom fields
// holding the BMC login. NetBox has no schema for it, so deployments keep it in
// custom fields; these are the names to read.
type NodeProviderMetal3NetBoxCustomFields struct {
	// BMCUsername holds the BMC login name. Defaults to "bmc_username".
	// +optional
	BMCUsername string `json:"bmcUsername,omitempty"`

	// BMCPassword holds the BMC password. The platform never serves this
	// field's value. Defaults to "bmc_password".
	// +optional
	BMCPassword string `json:"bmcPassword,omitempty"`
}

type Metal3NodeTypeSpec struct {
	NamedNodeTypeSpec `json:",inline"`

	// BareMetalHosts is a list of BareMetalHosts to use for this NodeType.
	// +optional
	BareMetalHosts Metal3BareMetalHosts `json:"bareMetalHosts,omitempty"`
}

type Metal3BareMetalHosts struct {
	// Selector is a label selector to select the BareMetalHosts to use for this NodeType.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// NodeProviderStatus defines the observed state of NodeProvider.
type NodeProviderStatus struct {
	// Conditions describe the current state of the platform NodeProvider.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Phase is the current lifecycle phase of the NodeProvider.
	// +optional
	Phase NodeProviderPhase `json:"phase,omitempty"`

	// Message is a human-readable message indicating details about why the NodeProvider is in its current state.
	// +optional
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeProviderList contains a list of NodeProvider
type NodeProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeProvider{}, &NodeProviderList{})
}
