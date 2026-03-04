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
}

// UsageDataDetails holds detailed information about the nodes and virtual cluster for an instance deployment of
// vCluster Platform
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type UsageDataDetails struct {
	// Nodes contains the details of the nodes
	Nodes []NodeInfo `json:"nodes"`

	// VClusters contains the details of the virtual clusters
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
}

// VirtualClusterInfo holds information about a single virtual cluster
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
