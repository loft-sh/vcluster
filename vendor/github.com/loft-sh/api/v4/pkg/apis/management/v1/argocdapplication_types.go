package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplication holds the ArgoCDApplication information
// +k8s:openapi-gen=true
// +resource:path=argocdapplications,rest=ArgoCDApplicationREST
type ArgoCDApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDApplicationSpec   `json:"spec,omitempty"`
	Status ArgoCDApplicationStatus `json:"status,omitempty"`
}

// ArgoCDApplicationSpec holds the specification
type ArgoCDApplicationSpec struct {
	storagev1.ArgoCDApplicationSpec `json:",inline"`
}

// ArgoCDApplicationStatus holds the status
type ArgoCDApplicationStatus struct {
	storagev1.ArgoCDApplicationStatus `json:",inline"`
}

func (a *ArgoCDApplication) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *ArgoCDApplication) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *ArgoCDApplication) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *ArgoCDApplication) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ArgoCDApplication) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *ArgoCDApplication) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
