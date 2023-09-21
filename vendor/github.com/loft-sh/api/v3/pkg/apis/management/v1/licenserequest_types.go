package v1

import (
	"github.com/loft-sh/external-types/loft-sh/admin-services/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LicenseRequest holds license request information
// +subresource-request
type LicenseRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the admin request spec (the input for the request).
	Spec LicenseRequestSpec `json:"spec,omitempty"`

	// Status is the admin request output (the output or result of the request).
	Status LicenseRequestStatus `json:"status,omitempty"`
}

type LicenseRequestSpec struct {
	// Route is the route to make the request to on the license server.
	Route string `json:"url"`
	// Input is the input payload to send to the url.
	Input server.StandardRequestInputFrontEnd `json:"input,omitempty"`
}

type LicenseRequestStatus struct {
	// OK indicates if the license request operation was successful or not. If OK is true, the front end should follow
	// the link in the output.
	// +optional
	OK bool `json:"ok,omitempty"`
	// Output is where the request output is stored.
	// +optional
	Output server.StandardRequestOutput `json:"output,omitempty"`
}
