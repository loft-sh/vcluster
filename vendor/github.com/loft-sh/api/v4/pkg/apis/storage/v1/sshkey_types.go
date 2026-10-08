package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSHKey holds the information of SSH keys
// +k8s:openapi-gen=true
type SSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHKeySpec   `json:"spec,omitempty"`
	Status SSHKeyStatus `json:"status,omitempty"`
}

func (a *SSHKey) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SSHKey) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SSHKey) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SSHKey) SetAccess(access []Access) {
	a.Spec.Access = access
}

type SSHKeySpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an SSH key
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// PublicKey is the public SSH key
	PublicKey string `json:"publicKey,omitempty"`
}

type SSHKeyStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SSHKeyList contains a list of SSHKeys
type SSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSHKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SSHKey{}, &SSHKeyList{})
}
