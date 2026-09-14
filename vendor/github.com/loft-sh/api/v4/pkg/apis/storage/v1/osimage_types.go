package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSImage holds the information of machine networks
// +k8s:openapi-gen=true
type OSImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSImageSpec   `json:"spec,omitempty"`
	Status OSImageStatus `json:"status,omitempty"`
}

func (a *OSImage) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *OSImage) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *OSImage) GetAccess() []Access {
	return a.Spec.Access
}

func (a *OSImage) SetAccess(access []Access) {
	a.Spec.Access = access
}

type OSImageSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an OS image
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// Properties is the configuration for the OS image
	// +optional
	Properties map[string]string `json:"properties,omitempty"`
}

type OSImageStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSImageList contains a list of OSImages
type OSImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OSImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OSImage{}, &OSImageList{})
}
