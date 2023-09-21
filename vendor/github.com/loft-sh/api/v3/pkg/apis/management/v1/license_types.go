package v1

import (
	admintypes "github.com/loft-sh/external-types/loft-sh/admin-services/pkg/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=LicenseRequest,verb=create,subresource=request,input=github.com/loft-sh/api/v3/pkg/apis/management/v1.LicenseRequest,result=github.com/loft-sh/api/v3/pkg/apis/management/v1.LicenseRequest
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// License holds the license information
// +k8s:openapi-gen=true
// +resource:path=licenses,rest=LicenseREST
// +subresource:request=LicenseRequest,path=request,kind=LicenseRequest,rest=LicenseRequestREST
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec,omitempty"`
	Status LicenseStatus `json:"status,omitempty"`
}

type LicenseSpec struct {
}

type LicenseStatus struct {
	// Buttons is the selection of routes or endpoints in the license server that are used for license related
	// operations such as updating subscriptions.
	// +optional
	Buttons admintypes.Buttons `json:"buttons,omitempty"`
	// License is the license data received from the license server.
	// +optional
	License *admintypes.License `json:"info,omitempty"`
	// InstanceID is the instance ID for the Loft license/instance.
	// +optional
	InstanceID string `json:"instanceID,omitempty"`
}
