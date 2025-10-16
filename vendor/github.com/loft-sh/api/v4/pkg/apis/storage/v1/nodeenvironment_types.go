package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NodeEnvironment conditions
	NodeEnvironmentConditionTypeInfrastructureProvisioned = "Provisioned"
	NodeEnvironmentConditionTypeInfrastructureSynced      = "Synced"
	NodeEnvironmentConditionTypeKubernetesProvisioned     = "KubernetesProvisioned"
	NodeEnvironmentConditionTypeKubernetesSynced          = "KubernetesSynced"
)

var (
	NodeEnvironmentConditions = []agentstoragev1.ConditionType{
		NodeEnvironmentConditionTypeInfrastructureProvisioned,
		NodeEnvironmentConditionTypeInfrastructureSynced,
		NodeEnvironmentConditionTypeKubernetesProvisioned,
		NodeEnvironmentConditionTypeKubernetesSynced,
	}
)

// NodeEnvironmentPhase defines the phase of the NodeEnvironment
type NodeEnvironmentPhase string

const (
	// NodeEnvironmentPhasePending is the initial state of a NodeEnvironment.
	NodeEnvironmentPhasePending NodeEnvironmentPhase = "Pending"
	// NodeEnvironmentPhaseAvailable means the underlying node environment has been successfully provisioned.
	NodeEnvironmentPhaseAvailable NodeEnvironmentPhase = "Available"
	// NodeEnvironmentPhaseFailed means the provisioning process has failed.
	NodeEnvironmentPhaseFailed NodeEnvironmentPhase = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="NodeProvider",type="string",JSONPath=".spec.nodeProviderRef"
// +kubebuilder:subresource:status

// NodeEnvironment holds the node environment for vCluster.
// +k8s:openapi-gen=true
type NodeEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeEnvironmentSpec   `json:"spec,omitempty"`
	Status NodeEnvironmentStatus `json:"status,omitempty"`
}

func (a *NodeEnvironment) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodeEnvironment) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// NodeEnvironmentSpec defines spec of node environment.
type NodeEnvironmentSpec struct {
	// Properties are the properties for the NodeEnvironment.
	// +optional
	Properties map[string]string `json:"properties"`

	// ProviderRef is the name of the NodeProvider that this NodeEnvironment is based on.
	ProviderRef string `json:"providerRef"`

	// VClusterRef references source vCluster. This is required.
	VClusterRef string `json:"vClusterRef"`

	// ControlPlane indicates if the node environment is a control plane environment. This is intentionally not omitempty as
	// we want to ensure that the control plane is always set for easier checking in for example terraform templates.
	// +optional
	ControlPlane bool `json:"controlPlane"`
}

type NodeEnvironmentStatus struct {
	// Phase is the current lifecycle phase of the NodeEnvironment.
	// +optional
	Phase NodeEnvironmentPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions describe the current state of the platform NodeClaim.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeEnvironmentList contains a list of NodeEnvironment
type NodeEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeEnvironment{}, &NodeEnvironmentList{})
}
