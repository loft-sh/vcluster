package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NodeClaim conditions
	NodeClaimConditionTypeProvisioned = "Provisioned"
	// NodeClaimConditionTypeJoined is the condition that indicates if the node claim is joined to the vCluster.
	NodeClaimConditionTypeJoined = "Joined"
	// ConditionTypeScheduled is the condition that indicates if the node claim is scheduled.
	NodeClaimConditionTypeScheduled = "Scheduled"
	// NodeClaimConditionTypeNotDrifted is the condition that indicates if the node claim is not drifted from the desired state.
	NodeClaimConditionTypeNotDrifted = "NotDrifted"
)

var (
	NodeClaimConditions = []agentstoragev1.ConditionType{
		NodeClaimConditionTypeScheduled,
		NodeClaimConditionTypeProvisioned,
		NodeClaimConditionTypeJoined,
	}
)

// NodeClaimPhase defines the phase of the NodeClaim
type NodeClaimPhase string

const (
	// NodeClaimPhasePending is the initial state of a NodeClaim.
	NodeClaimPhasePending NodeClaimPhase = "Pending"
	// NodeClaimPhaseAvailable means the underlying node has been successfully provisioned.
	NodeClaimPhaseAvailable NodeClaimPhase = "Available"
	// NodeClaimPhaseFailed means the provisioning process has failed.
	NodeClaimPhaseFailed NodeClaimPhase = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="NodeType",type="string",JSONPath=".spec.typeRef"
// +kubebuilder:subresource:status

// NodeClaim holds the node claim for vCluster.
// +k8s:openapi-gen=true
type NodeClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeClaimSpec   `json:"spec,omitempty"`
	Status NodeClaimStatus `json:"status,omitempty"`
}

func (a *NodeClaim) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodeClaim) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// NodeClaimSpec defines spec of node claim.
type NodeClaimSpec struct {
	// Taints will be applied to the NodeClaim's node.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`

	// StartupTaints are taints that are applied to nodes upon startup which are expected to be removed automatically
	// within a short period of time, typically by a DaemonSet that tolerates the taint. These are commonly used by
	// daemonsets to allow initialization and enforce startup ordering.  StartupTaints are ignored for provisioning
	// purposes in that pods are not required to tolerate a StartupTaint in order to have nodes provisioned for them.
	// +optional
	StartupTaints []corev1.Taint `json:"startupTaints,omitempty"`

	// KubeletArgs are additional arguments to pass to the kubelet.
	// +optional
	KubeletArgs map[string]string `json:"kubeletArgs,omitempty"`

	// DesiredCapacity specifies the resources requested by the NodeClaim.
	DesiredCapacity corev1.ResourceList `json:"desiredCapacity,omitempty"`

	// Requirements are the requirements for the NodeClaim.
	Requirements []corev1.NodeSelectorRequirement `json:"requirements,omitempty"`

	// Properties are extra properties for the NodeClaim.
	// +optional
	Properties map[string]string `json:"properties"`

	// ProviderRef is the name of the NodeProvider that this NodeClaim is based on.
	ProviderRef string `json:"providerRef"`

	// TypeRef is the full name of the NodeType that this NodeClaim is based on.
	// +optional
	TypeRef string `json:"typeRef,omitempty"`

	// VClusterRef references source vCluster. This is required.
	VClusterRef string `json:"vClusterRef"`

	// ControlPlane indicates if the node claim is for a control plane node. This is intentionally not omitempty as
	// we want to ensure that the control plane is always set for easier checking in for example terraform templates.
	// +optional
	ControlPlane bool `json:"controlPlane"`
}

type NodeClaimStatus struct {
	// Phase is the current lifecycle phase of the NodeClaim.
	// +optional
	Phase NodeClaimPhase `json:"phase,omitempty"`

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

// NodeClaimList contains a list of NodeClaim
type NodeClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeClaim{}, &NodeClaimList{})
}
