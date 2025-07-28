package v1

import (
	"strings"

	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeProviderPropertyKey = "vcluster.com/node-provider"
	NodeTypePropertyKey     = "vcluster.com/node-type"

	// NodeTypeConditionTypeSynced is the condition that indicates if the node type is synced with provider.
	NodeTypeConditionTypeSynced = "Synced"
)

var (
	NodeTypeConditions = []agentstoragev1.ConditionType{
		NodeTypeConditionTypeSynced,
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

	// Zone is the zone of the node type. If empty, will default to "global".
	// +optional
	Zone string `json:"zone,omitempty"`

	// Region is the region of the node type.
	// +optional
	Region string `json:"region,omitempty"`

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

func (a *NodeType) GetAllProperties() []corev1.NodeSelectorRequirement {
	// default properties
	zone := "global"
	if a.Spec.Zone != "" {
		zone = a.Spec.Zone
	}

	// build all properties
	allProperties := []corev1.NodeSelectorRequirement{
		{
			Key:      corev1.LabelInstanceTypeStable,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{a.Name},
		},
		{
			Key:      corev1.LabelOSStable,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{string(corev1.Linux)},
		},
		{
			Key:      corev1.LabelTopologyZone,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{zone},
		},
		{
			Key:      "karpenter.sh/capacity-type",
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{"on-demand"},
		},
		{
			Key:      NodeProviderPropertyKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{a.Spec.ProviderRef},
		},
		{
			Key:      NodeTypePropertyKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{a.Name},
		},
	}
	if a.Spec.Region != "" {
		allProperties = append(allProperties, corev1.NodeSelectorRequirement{
			Key:      corev1.LabelTopologyRegion,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{a.Spec.Region},
		})
	}

	// add custom properties
	for key, value := range a.Spec.Properties {
		allProperties = append(allProperties, corev1.NodeSelectorRequirement{
			Key:      key,
			Operator: corev1.NodeSelectorOpIn,
			Values:   strings.Split(value, ","),
		})
	}

	return allProperties
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

	// Capacity is the capacity of the node type.
	// +optional
	Capacity *NodeTypeCapacity `json:"capacity,omitempty"`

	// Conditions holds several conditions the node type might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

type NodeTypeCapacity struct {
	// Total is the total number of nodes of this type
	// +optional
	Total int `json:"total,omitempty"`

	// Available is the number of available nodes of this type
	// +optional
	Available int `json:"available,omitempty"`

	// Provisioned is the number of already provisioned nodes of this type
	// +optional
	Provisioned int `json:"provisioned,omitempty"`
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
