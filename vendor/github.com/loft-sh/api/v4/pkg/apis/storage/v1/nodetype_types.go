package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeTypePropertyKey = "vcluster.com/node-type"

	// NodeTypeConditionTypeSynced is the condition that indicates if the node type is synced with provider.
	NodeTypeConditionTypeSynced  = "Synced"
	NodeTypeConditionHasCapacity = "HasCapacity"
)

var (
	NodeTypeConditions = []agentstoragev1.ConditionType{
		NodeTypeConditionTypeSynced,
		NodeTypeConditionHasCapacity,
	}
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.capacity.available"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.capacity.total"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:resource:scope=Cluster

// NodeType holds the information of a node type.
// +k8s:openapi-gen=true
type NodeType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NodeTypeSpec   `json:"spec,omitempty"`
	Status NodeTypeStatus `json:"status,omitempty"`
}

func (a *NodeType) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodeType) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

type NodeTypeSpec struct {
	// ProviderRef is the node provider to use for this node type.
	// +optional
	ProviderRef string `json:"providerRef,omitempty"`

	// Properties returns a flexible set of properties that may be selected for scheduling.
	Properties map[string]string `json:"properties,omitempty"`

	// Resources lists the full resources for a single node.
	Resources corev1.ResourceList `json:"resources,omitempty"`

	// Overhead defines the resource overhead for this node type.
	// +optional
	Overhead *NodeTypeOverhead `json:"overhead,omitempty"`

	// Cost is the instance cost. The higher the cost, the less likely it is to be selected. If empty, cost is automatically calculated
	// from the resources specified.
	// +optional
	Cost int64 `json:"cost,omitempty"`

	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

// NodeTypeOverhead defines the resource overhead for a node type.
type NodeTypeOverhead struct {
	// KubeReserved is the resource overhead for kubelet and other Kubernetes system daemons.
	// +optional
	KubeReserved corev1.ResourceList `json:"kubeReserved,omitempty"`
}

// NodeTypePhase defines the phase of the NodeType
type NodeTypePhase string

const (
	// PhasePending is the initial state of a NodeType.
	NodeTypePhasePending NodeTypePhase = "Pending"
	// NodeTypePhaseAvailable means the provisioning process has failed.
	NodeTypePhaseAvailable NodeTypePhase = "Available"
	// NodeTypePhaseFailed means the provisioning process has failed.
	NodeTypePhaseFailed NodeTypePhase = "Failed"
)

// NodeTypeStatus holds the status of a node type
type NodeTypeStatus struct {
	// Phase is the current lifecycle phase of the NodeType.
	// +optional
	Phase NodeTypePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// Cost is the calculated instance cost from the resources specified or the price specified from spec. The higher the cost, the less likely it is to be selected.
	// +optional
	Cost int64 `json:"cost,omitempty"`

	// Capacity is the capacity of the node type.
	// +optional
	Capacity NodeTypeCapacity `json:"capacity,omitempty"`

	// Requirements is the calculated requirements based of the properties for the node type.
	// +optional
	Requirements []corev1.NodeSelectorRequirement `json:"requirements,omitempty"`

	// Conditions holds several conditions the node type might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// IMPORTANT: DO NOT use omitempty for values in NodeTypeCapacity.
// The values are used in NodePool calculations and for UI.
type NodeTypeCapacity struct {
	// Total is the total number of nodes of this type
	Total int `json:"total"`

	// Claimed is the number of already claimed nodes of this type
	Claimed int `json:"claimed"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeTypeList contains a list of NodeType
type NodeTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeType `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeType{}, &NodeTypeList{})
}
