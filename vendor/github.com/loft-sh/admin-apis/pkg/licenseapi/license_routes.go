package licenseapi

// LicenseAPIRoutes contains all key routes of the license api
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type LicenseAPIRoutes struct {
	ChatAuth LicenseAPIRoute `json:"chatAuth,omitempty"`
}

// LicenseAPIRoute is a single route of the license api
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type LicenseAPIRoute struct {
	URL    string `json:"url,omitempty"`
	Method string `json:"method,omitempty"`
}
