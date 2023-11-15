package licenseapi

// Limit defines a limit set in the license
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Limit struct {
	// Name is the name of the resource.
	// +optional
	Name string `json:"name,omitempty"`
	// DisplayName is for display purposes.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// AdjustButton is the button to be shown for adjusting the limit (e.g. buying more seats)
	// +optional
	AdjustButton *Button `json:"adjustButton,omitempty"`

	// Limit specifies the limit for this resource.
	// +optional
	Quantity *ResourceCount `json:"quantity,omitempty"`
}
