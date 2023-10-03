package server

// +genclient
// +genclient:nonNamespaced

// StandardRequestInputFrontEnd is the standard input request payload for the front end -- this is
// the same (it is embedded) in the StandardRequestInput just without the token auth as the Loft
// backend will handle adding that for communication to the admin server.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type StandardRequestInputFrontEnd struct {
	// ReturnURL is the url that operations should ultimately return to after their operation is
	// complete. For example, once the license activate process is done, the Loft portal should
	// redirect to this URL.
	// +optional
	ReturnURL string `json:"returnURL,omitempty"`
}

// StandardRequestInput is a standard payload object used between the Loft front end and the Loft
// admin server. It accepts auth information and a URL that is used by the admin server when
// crafting the redirect link that is returned from "start" type operations (ex. trial activate
// start that creates a one-time token and returns a url to go to for the front end).
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type StandardRequestInput struct {
	*InstanceTokenAuth
	StandardRequestInputFrontEnd
}

// StandardRequestOutput is a standard payload object returned by the Loft admin server, it
// contains a RedirectURL that encodes a one-time token (if applicable) and a return URL (if
// applicable) that the front end can then follow to continue the license operation.
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type StandardRequestOutput struct {
	// RedirectURL is the URL to redirect to for continuing the license operation (typically stripe
	// or the loft portal).
	// +optional
	RedirectURL string `json:"redirectURL,omitempty"`
}
