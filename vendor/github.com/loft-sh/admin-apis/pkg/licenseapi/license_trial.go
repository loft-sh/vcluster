package licenseapi

// Trial represents a trial
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Trial struct {
	// Name is the unique id of this trial
	Name string `json:"name,omitempty"`

	// DisplayName is a display name for the trial
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Start is the unix timestamp stating when the trial was started
	// +optional
	Start int64 `json:"start,omitempty"`

	// End is the unix timestamp stating when the trial will end or ended
	// +optional
	End int64 `json:"end,omitempty"`

	// Status is the status of this trial
	// +optional
	Status string `json:"status,omitempty"`
}
