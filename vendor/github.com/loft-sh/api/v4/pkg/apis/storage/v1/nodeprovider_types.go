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

	// NodeProviderConditionTypeInitialized is the condition that indicates if the node provider is initialized.
	NodeProviderConditionTypeInitialized = "Initialized"
)

var (
	NodeProviderConditions = []agentstoragev1.ConditionType{
		NodeProviderConditionTypeInitialized,
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

	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

type NodeProviderClusterAPI struct {
	ClusterAPIObjects `json:",inline"`

	// ClusterRef is a reference to connected host cluster in which KubeVirt operator is running
	ClusterRef *NodeProviderClusterRef `json:"clusterRef,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []ClusterAPINodeTypeSpec `json:"nodeTypes,omitempty"`
}

// NodeProviderBCMSpec defines the configuration for a BCM node provider.
type NodeProviderBCM struct {
	// SecretRef is a reference to secret with keys for BCM auth.
	SecretRef *NamespacedRef `json:"secretRef"`

	// Endpoint is a address for head node.
	Endpoint string `json:"endpoint"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []BCMNodeTypeSpec `json:"nodeTypes,omitempty"`
}

type NodeProviderTerraform struct {
	// NodeTemplate is the template to use for this node provider.
	NodeTemplate *TerraformTemplate `json:"nodeTemplate,omitempty"`

	// NodeEnvironmentTemplate is the template to use for this node environment.
	NodeEnvironmentTemplate *TerraformNodeEnvironmentTemplate `json:"nodeEnvironmentTemplate,omitempty"`

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

type TerraformNodeEnvironmentTemplate struct {
	// Deprecated: Use Infrastructure and Kubernetes instead.
	TerraformTemplate `json:",inline"`

	// Infrastructure is the infrastructure template to use for this node environment.
	Infrastructure *TerraformTemplate `json:"infrastructure,omitempty"`

	// Kubernetes is the kubernetes template to use for this node environment.
	Kubernetes *TerraformTemplate `json:"kubernetes,omitempty"`
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

// NodeProviderKubeVirt defines the configuration for a KubeVirt node provider.
type NodeProviderKubeVirt struct {
	// ClusterRef is a reference to connected host cluster in which KubeVirt operator is running
	ClusterRef *NodeProviderClusterRef `json:"clusterRef,omitempty"`

	// VirtualMachineTemplate is a KubeVirt VirtualMachine template to use by NodeTypes managed by this NodeProvider
	VirtualMachineTemplate *runtime.RawExtension `json:"virtualMachineTemplate,omitempty"`

	// NodeTypes define NodeTypes that should be automatically created for this provider.
	NodeTypes []KubeVirtNodeTypeSpec `json:"nodeTypes"`
}

type NodeProviderClusterRef struct {
	// Cluster is the connected cluster the VMs will be created in
	Cluster string `json:"cluster"`

	// Namespace is the namespace inside the connected cluster holding VMs
	Namespace string `json:"namespace,omitempty"`
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
