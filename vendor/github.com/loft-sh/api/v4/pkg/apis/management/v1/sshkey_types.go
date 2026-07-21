package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// SSHKey holds the OS image.
// +k8s:openapi-gen=true
// +resource:path=sshkeys,rest=SSHKeyREST
type SSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHKeySpec   `json:"spec,omitempty"`
	Status SSHKeyStatus `json:"status,omitempty"`
}

type SSHKeySpec struct {
	storagev1.SSHKeySpec `json:",inline"`
}

type SSHKeyStatus struct {
	storagev1.SSHKeyStatus `json:",inline"`
}

func (a *SSHKey) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *SSHKey) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SSHKey) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *SSHKey) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
