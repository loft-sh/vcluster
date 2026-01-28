package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterResourceUsage holds information about a virtual cluster's
// usage of node resources.
// +subresource-request
type VirtualClusterResourceUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterResourceUsageStatus `json:"status,omitempty"`
}

type VirtualClusterResourceUsageStatus struct {
	// ResourceUsage contains the aggregated result of the queries to the virtual cluster's nodes.
	ResourceUsage VirtualClusterResourceUsageMap `json:"resourceUsage,omitempty"`
}

type VirtualClusterResourceUsageMap struct {
	// Nodes is the total count of nodes attached to the virtual cluster.
	Nodes int `json:"nodes"`
	// Capacity is a map of resources to their total amounts across all attached nodes.
	Capacity map[string]int `json:"capacity,omitempty"`
}
