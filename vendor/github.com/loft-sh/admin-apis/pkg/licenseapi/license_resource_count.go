package licenseapi

// ResourceCount stores the number of existing, active and total number of resources created.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type ResourceCount struct {
	// Active specifies the number of currently active resource (non-sleeping).
	// +optional
	Active *int64 `json:"active,omitempty"`
	// Total specifies the number of currently existing resources.
	// +optional
	Total *int64 `json:"total,omitempty"`
	// TotalCreated is a continuous counter of the amount of resources ever created.
	// +optional
	TotalCreated *int64 `json:"created,omitempty"`
	// Committed specifies the amount of resource consumption customers have committed to for a given billing period.
	// It can be exceeded and will then be charged with overage fees.
	// +optional
	Committed *int64 `json:"committed,omitempty"`
}
