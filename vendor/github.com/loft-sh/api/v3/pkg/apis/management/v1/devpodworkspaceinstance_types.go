package v1

import (
	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	agentstoragev1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceInstance holds the DevPodWorkspaceInstance information
// +k8s:openapi-gen=true
// +resource:path=devpodworkspaceinstances,rest=DevPodWorkspaceInstanceREST
// +subresource:request=DevPodWorkspaceInstanceUp,path=up,kind=DevPodWorkspaceInstanceUp,rest=DevPodWorkspaceInstanceUpREST
// +subresource:request=DevPodWorkspaceInstanceDelete,path=delete,kind=DevPodWorkspaceInstanceDelete,rest=DevPodWorkspaceInstanceDeleteREST
// +subresource:request=DevPodWorkspaceInstanceSsh,path=ssh,kind=DevPodWorkspaceInstanceSsh,rest=DevPodWorkspaceInstanceSshREST
// +subresource:request=DevPodWorkspaceInstanceStop,path=stop,kind=DevPodWorkspaceInstanceStop,rest=DevPodWorkspaceInstanceStopREST
// +subresource:request=DevPodWorkspaceInstanceGetStatus,path=getstatus,kind=DevPodWorkspaceInstanceGetStatus,rest=DevPodWorkspaceInstanceGetStatusREST
type DevPodWorkspaceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceInstanceSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceInstanceStatus `json:"status,omitempty"`
}

// DevPodWorkspaceInstanceSpec holds the specification
type DevPodWorkspaceInstanceSpec struct {
	storagev1.DevPodWorkspaceInstanceSpec `json:",inline"`
}

// DevPodWorkspaceInstanceStatus holds the status
type DevPodWorkspaceInstanceStatus struct {
	storagev1.DevPodWorkspaceInstanceStatus `json:",inline"`

	// SleepModeConfig is the sleep mode config of the workspace. This will only be shown
	// in the front end.
	// +optional
	SleepModeConfig *clusterv1.SleepModeConfig `json:"sleepModeConfig,omitempty"`
}

func (a *DevPodWorkspaceInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *DevPodWorkspaceInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *DevPodWorkspaceInstance) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceInstance) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceInstance) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceInstance) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
