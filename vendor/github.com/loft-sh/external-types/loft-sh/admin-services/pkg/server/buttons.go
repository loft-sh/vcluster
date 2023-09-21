package server

// +genclient
// +genclient:nonNamespaced

// Button is an object that represents a button in the Loft UI that links to some external service
// for handling operations for licensing for example.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Button struct {
	// URL is the link at the other end of the button.
	URL string `json:"URL"`
	// DisplayText is the text to display on the button. If display text is unset the button will
	// never be shown in the loft UI.
	// +optional
	DisplayText string `json:"displayText,omitempty"`
	// PayloadType indicates the payload type to set in the request. Typically, this will be
	// "standard" -- meaning the standard payload that includes the instance token auth and a
	// return url, however in the future we may add additional types. An unset value is assumed to
	// be "standard".
	// +optional
	PayloadType string `json:"payloadType,omitempty"`
	// Direct indicates if the Loft front end should directly hit this endpoint. If false, it means
	// that the Loft front end will be hitting the license server first to generate a one time token
	// for the operation; this also means that there will be a redirect URL in the response to the
	// request for this and that link should be followed by the front end.
	// +optional
	Direct bool `json:"direct,omitempty"`
}

// Buttons is an object containing Button objects. These Button objects are rendered by the Loft UI
// to allow for extending Loft to include links to external services such as license updates.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type Buttons []*Button
