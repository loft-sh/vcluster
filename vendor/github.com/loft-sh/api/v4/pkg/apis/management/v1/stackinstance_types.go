package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +genclient:method=GetOutputs,verb=get,subresource=outputs,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.StackInstanceOutputs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackInstance holds the StackInstance information
// +k8s:openapi-gen=true
// +resource:path=stackinstances,rest=StackInstanceREST
// +subresource:request=StackInstanceOutputs,path=outputs,kind=StackInstanceOutputs,rest=StackInstanceOutputsREST
type StackInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackInstanceSpec   `json:"spec,omitempty"`
	Status StackInstanceStatus `json:"status,omitempty"`
}

// StackInstanceSpec holds the specification
type StackInstanceSpec struct {
	storagev1.StackInstanceSpec `json:",inline"`
}

// StackInstanceStatus holds the status
type StackInstanceStatus struct {
	storagev1.StackInstanceStatus `json:",inline"`
}

func (a *StackInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *StackInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *StackInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *StackInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *StackInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *StackInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
