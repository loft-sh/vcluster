package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackTemplate is the reusable, parameterized blueprint for a StackInstance. It is passive
// (no controller) and resolved by the StackInstance controller at reconcile time.
// +k8s:openapi-gen=true
type StackTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackTemplateSpec   `json:"spec,omitempty"`
	Status StackTemplateStatus `json:"status,omitempty"`
}

func (a *StackTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *StackTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *StackTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *StackTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

type StackTemplateSpec struct {
	// DisplayName is the catalog/UI name.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the stack template.
	// +optional
	Description string `json:"description,omitempty"`

	// Icon is a logo/png reference for the catalog.
	// +optional
	Icon string `json:"icon,omitempty"`

	// StackTemplateDefinition is the reusable payload (declared inputs + task DAG),
	// embedded inline so the template's YAML shape is unchanged. The same struct is a
	// StackInstance's inline spec.template.
	StackTemplateDefinition `json:",inline"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type StackTemplateStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackTemplateList contains a list of StackTemplates
type StackTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StackTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StackTemplate{}, &StackTemplateList{})
}
