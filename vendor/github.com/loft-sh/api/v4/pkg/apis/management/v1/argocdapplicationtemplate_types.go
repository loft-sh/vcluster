package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplicationTemplate holds the ArgoCDApplicationTemplate information
// +k8s:openapi-gen=true
// +resource:path=argocdapplicationtemplates,rest=ArgoCDApplicationTemplateREST
type ArgoCDApplicationTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDApplicationTemplateSpec   `json:"spec,omitempty"`
	Status ArgoCDApplicationTemplateStatus `json:"status,omitempty"`
}

// BlueprintTemplateSpec holds the specification
type ArgoCDApplicationTemplateSpec struct {
	storagev1.ArgoCDApplicationTemplateSpec `json:",inline"`
}

// BlueprintTemplateStatus holds the status
type ArgoCDApplicationTemplateStatus struct {
	storagev1.ArgoCDApplicationTemplateStatus `json:",inline"`
}

func (a *ArgoCDApplicationTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *ArgoCDApplicationTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ArgoCDApplicationTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *ArgoCDApplicationTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
