package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackTemplate holds the StackTemplate information
// +k8s:openapi-gen=true
// +resource:path=stacktemplates,rest=StackTemplateREST
type StackTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackTemplateSpec   `json:"spec,omitempty"`
	Status StackTemplateStatus `json:"status,omitempty"`
}

// StackTemplateSpec holds the specification
type StackTemplateSpec struct {
	storagev1.StackTemplateSpec `json:",inline"`
}

// StackTemplateStatus holds the status
type StackTemplateStatus struct {
	storagev1.StackTemplateStatus `json:",inline"`
}

func (a *StackTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *StackTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *StackTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *StackTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
