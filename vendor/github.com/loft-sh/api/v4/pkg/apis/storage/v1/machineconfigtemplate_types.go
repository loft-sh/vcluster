package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// MachineConfigTemplate holds the machine config template for vCluster.
// +k8s:openapi-gen=true
type MachineConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineConfigTemplateSpec   `json:"spec,omitempty"`
	Status MachineConfigTemplateStatus `json:"status,omitempty"`
}

func (a *MachineConfigTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *MachineConfigTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *MachineConfigTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *MachineConfigTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

// MachineConfigTemplateSpec defines spec of machine config template.
type MachineConfigTemplateSpec struct {
	// DisplayName is the name of the NodeClaim that is displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// CloudInitTemplate is the cloud init template to use for the machine config.
	// +optional
	CloudInitTemplate string `json:"cloudInitTemplate,omitempty"`

	// NetworkDataTemplate is the network data template to use for the machine config.
	// +optional
	NetworkDataTemplate string `json:"networkDataTemplate,omitempty"`
}

type MachineConfigTemplateStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineConfigTemplateList contains a list of MachineConfigTemplate
type MachineConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineConfigTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachineConfigTemplate{}, &MachineConfigTemplateList{})
}
