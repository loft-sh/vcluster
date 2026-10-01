package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// OSImage holds the OS image.
// +k8s:openapi-gen=true
// +resource:path=osimages,rest=OSImageREST
type OSImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OSImageSpec   `json:"spec,omitempty"`
	Status OSImageStatus `json:"status,omitempty"`
}

type OSImageSpec struct {
	storagev1.OSImageSpec `json:",inline"`
}

type OSImageStatus struct {
	storagev1.OSImageStatus `json:",inline"`
}

func (a *OSImage) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *OSImage) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *OSImage) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *OSImage) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
