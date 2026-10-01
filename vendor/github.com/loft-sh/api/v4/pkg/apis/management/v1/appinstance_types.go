package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppInstance holds the AppInstance information
// +k8s:openapi-gen=true
// +resource:path=appinstances,rest=AppInstanceREST
// +subresource:request=AppInstanceLog,path=log,kind=AppInstanceLog,rest=AppInstanceLogREST
type AppInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppInstanceSpec   `json:"spec,omitempty"`
	Status AppInstanceStatus `json:"status,omitempty"`
}

// AppInstanceSpec holds the specification
type AppInstanceSpec struct {
	storagev1.AppInstanceSpec `json:",inline"`
}

// AppInstanceStatus holds the status
type AppInstanceStatus struct {
	storagev1.AppInstanceStatus `json:",inline"`
}

func (a *AppInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *AppInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *AppInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *AppInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *AppInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *AppInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
