package licenseapi

// UsageData holds information for an instance deployment of vCluster Platform
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type UsageData struct {
	// FeatureUsage contains the usage of features
	FeatureUsage map[string]FeatureUsage `json:"featureUsage"`

	// ResourceUsage contains the usage of resources
	ResourceUsage map[string]ResourceCount `json:"resourceUsage"`

	// Details contains the details of the usage data
	Details UsageDataDetails `json:"details"`

	// GPUUsage contains the instance-wide accelerator usage broken down by GPU type
	// (vendor and model). It is supplementary to the aggregate GPU count reported in
	// ResourceUsage and is intended for per-type insight.
	// +optional
	GPUUsage []GPUTypeUsage `json:"gpuUsage,omitempty"`
}

// UsageDataDetails holds detailed information about the nodes and tenant cluster for an instance deployment of
// vCluster Platform
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type UsageDataDetails struct {
	// Nodes contains the details of the nodes
	Nodes []NodeInfo `json:"nodes"`

	// VClusters contains the details of the tenant clusters
	VClusters []VirtualClusterInfo `json:"vClusters"`
}

// FeatureUsage holds information about whether a feature is used and its status
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type FeatureUsage struct {
	Used   bool   `json:"used"`
	Status string `json:"status"`
}

// NodeInfo holds information about a single node
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type NodeInfo struct {
	MachineID         string            `json:"machine_id"`
	CreationTimestamp string            `json:"creation_timestamp"`
	Capacity          map[string]string `json:"capacity"`

	// GPUs is the per-type accelerator breakdown for this node. It is derived from the
	// node's advertised resources (and, in later iterations, DRA ResourceSlices) and is
	// supplementary to Capacity, which is retained unchanged.
	// +optional
	GPUs []NodeGPUInfo `json:"gpus,omitempty"`
}

// NodeGPUInfo holds information about the accelerators of a single type (vendor and
// model) on a single node.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type NodeGPUInfo struct {
	// Vendor is a normalized accelerator vendor id, e.g. "nvidia", "amd", "intel",
	// "habana", "aws", "google".
	Vendor string `json:"vendor"`

	// Model is the hardware model when known (e.g. from the GPU vendor's node labels or
	// a DRA device attribute). Empty when the model cannot be determined.
	// +optional
	Model string `json:"model,omitempty"`

	// Source is how these accelerators were discovered: "device-plugin" or "dra".
	Source string `json:"source"`

	// Allocatable is the number of schedulable units advertised for this type on the node
	// (the device-plugin capacity, or the count of DRA devices).
	Allocatable int64 `json:"allocatable"`

	// Physical is the number of physical accelerators of this type on the node after
	// normalizing sharing schemes such as time-slicing and MIG. It equals Allocatable when
	// those schemes are not normalized. This is the value intended for metering.
	Physical int64 `json:"physical"`
}

// GPUTypeUsage is an instance-deployment-wide aggregate accelerator count for a single
// type (vendor and model), summed across all nodes.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type GPUTypeUsage struct {
	// Vendor is a normalized accelerator vendor id, e.g. "nvidia", "amd", "intel",
	// "habana", "aws", "google".
	Vendor string `json:"vendor"`

	// Model is the hardware model when known, empty otherwise.
	// +optional
	Model string `json:"model,omitempty"`

	// Allocatable is the total number of schedulable units of this type across all nodes.
	Allocatable int64 `json:"allocatable"`

	// Physical is the total number of physical accelerators of this type across all nodes,
	// after normalizing sharing schemes. This is the value intended for metering.
	Physical int64 `json:"physical"`
}

// VirtualClusterInfo holds information about a single tenant cluster
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type VirtualClusterInfo struct {
	UID               string   `json:"uid"`
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	CreationTimestamp string   `json:"creation_timestamp"`
	IsAvailable       bool     `json:"is_available"`
	NodeMachineIDs    []string `json:"node_machine_ids"`
}
