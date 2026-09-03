package v1

import (
	argoapplicationsv1alpha1 "github.com/loft-sh/external-types/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplicationTemplate holds the information of Argo CD application templates
// +k8s:openapi-gen=true
type ArgoCDApplicationTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDApplicationTemplateSpec   `json:"spec,omitempty"`
	Status ArgoCDApplicationTemplateStatus `json:"status,omitempty"`
}

func (a *ArgoCDApplicationTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *ArgoCDApplicationTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ArgoCDApplicationTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *ArgoCDApplicationTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

type ArgoCDApplicationTemplateSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an OS image
	// +optional
	Description string `json:"description,omitempty"`

	// Template is the blueprint template definition
	// +optional
	Template ArgoCDApplicationTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type ArgoCDApplicationTemplateDefinition struct {
	// Metadata is the metadata of the Argo CD application
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	// Spec is the spec of the Argo CD application
	// +optional
	Spec argoapplicationsv1alpha1.ApplicationSpec `json:"spec,omitempty"`
}

type ArgoCDApplicationTemplateStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplicationTemplateList contains a list of ArgoCDApplicationTemplates
type ArgoCDApplicationTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDApplicationTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDApplicationTemplate{}, &ArgoCDApplicationTemplateList{})
}
