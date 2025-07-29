package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeProviderTypeBCM      string = "bcm"
	NodeProviderTypePods     string = "pods"
	NodeProviderTypeKubeVirt string = "kubeVirt"

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
	NodeProvidedManagedTypeIndicatorLabel = "autoscaling.loft.sh/managed-by"

	// BCM specific annotations
	NodeTypeNodesAnnotation      = "bcm.loft.sh/nodes"
	NodeTypeNodeGroupsAnnotation = "bcm.loft.sh/node-groups"
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
	// Pods configures a node provider based on Kubernetes Pods.
	// This can be used for simpler, pod-based node provisioning for testing or demos.
	// +optional
	Pods *NodeProviderPods `json:"pods,omitempty"`

	// BCM configures a node provider for BCM Bare Metal Cloud environments.
	// +optional
	BCM *NodeProviderBCM `json:"bcm,omitempty"`

	// Kubevirt configures a node provider using KubeVirt, enabling virtual machines
	// to be provisioned as nodes within a vCluster.
	// +optional
	KubeVirt *NodeProviderKubeVirt `json:"kubeVirt,omitempty"`

	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

// NodeProviderPodsSpec defines the configuration for a pod-based node provider.
type NodeProviderPods struct {
	// Image is the container image to use for the pod-based node provider.
	Image string `json:"image,omitempty"`
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

type BCMNodeTypeSpec struct {
	NodeTypeSpec `json:",inline"`

	// Name is the name of this node type.
	Name string `json:"name"`

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

// NodeProviderKubeVirt defines the configuration for a KubeVirt node provider.
type NodeProviderKubeVirt struct {
	// If KubeVirt is deployed inside one of the vClusters, reference this vCluster here.
	// Otherwise it will be assumed to run in the same host cluster as platform.
	// +optional
	VirtualClusterRef *VirtualClusterClusterRef `json:"virtualClusterRef,omitempty"`

	// TargetNamespace is the namespace within the KubeVirt cluster where the
	// VirtualMachines and their secrets will be created.
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`
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
