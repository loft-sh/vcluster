package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterResourceUsage holds information about a tenant cluster's
// usage of node resources.
// +subresource-request
type VirtualClusterResourceUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterResourceUsageStatus `json:"status,omitempty"`
}

type VirtualClusterResourceUsageStatus struct {
	// ResourceUsage contains the aggregated result of the queries to the tenant cluster's nodes.
	ResourceUsage VirtualClusterResourceUsageMap `json:"resourceUsage,omitempty"`
}

type VirtualClusterResourceUsageMap struct {
	// Nodes is the total count of nodes attached to the tenant cluster.
	Nodes int `json:"nodes"`
	// Capacity is a map of raw node resource names to their total amounts across all attached
	// nodes (e.g. "cpu", "nvidia.com/gpu"). It is per-resource-name, not per-vendor: for GPUs it
	// only includes "nvidia.com/gpu" and not other accelerator vendors, so use GPUs (not
	// Capacity) for a cross-vendor GPU total.
	Capacity map[string]int `json:"capacity,omitempty"`
	// GPUs is the accelerator usage across all attached nodes, broken down by GPU type
	// (vendor and model).
	GPUs []GPUTypeUsage `json:"gpus,omitempty"`
}

// GPUTypeUsage is an aggregate accelerator count for a single type (vendor and model)
// across all nodes attached to the tenant cluster.
type GPUTypeUsage struct {
	// Vendor is a normalized accelerator vendor id, e.g. "nvidia", "amd", "intel".
	Vendor string `json:"vendor"`
	// Model is the hardware model when known, empty otherwise.
	Model string `json:"model,omitempty"`
	// Allocatable is the total number of schedulable units of this type across all nodes.
	Allocatable int64 `json:"allocatable"`
	// Physical is the total number of physical accelerators of this type across all nodes.
	Physical int64 `json:"physical"`
}
