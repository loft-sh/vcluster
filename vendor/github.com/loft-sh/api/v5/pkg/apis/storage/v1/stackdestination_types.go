package v1

// StackDestination is where a stack's applications deploy: a tenant cluster or a control plane
// cluster.
type StackDestination struct {
	// VirtualCluster names the tenant cluster to deploy into. Mutually exclusive with Cluster.
	// +optional
	VirtualCluster *StackDestinationVirtualCluster `json:"virtualCluster,omitempty"`

	// Cluster names the connected cluster to deploy into. Mutually exclusive with VirtualCluster.
	// +optional
	Cluster *StackDestinationCluster `json:"cluster,omitempty"`
}

// StackDestinationVirtualCluster selects a tenant cluster to deploy into.
type StackDestinationVirtualCluster struct {
	// Name of the tenant cluster.
	// +optional
	Name string `json:"name,omitempty"`
}

// StackDestinationCluster selects a connected cluster to deploy into.
type StackDestinationCluster struct {
	// Name of the connected cluster.
	// +optional
	Name string `json:"name,omitempty"`
}
