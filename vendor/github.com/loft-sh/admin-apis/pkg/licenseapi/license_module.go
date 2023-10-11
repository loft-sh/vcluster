package licenseapi

// Module is a struct representing a module of the product
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Module struct {
	Name string `json:"name"`

	// +optional
	DisplayName string `json:"displayName,omitempty"`

	Limits   []*Limit   `json:"limits,omitempty"`
	Features []*Feature `json:"features,omitempty"`
}
