package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="NodeProvider",type="string",JSONPath=".spec.nodeProviderRef"
// +kubebuilder:subresource:status

// NodeEnvironment holds the node environment for vCluster.
// +k8s:openapi-gen=true
// +resource:path=nodeenvironments,rest=NodeEnvironmentREST,statusRest=NodeEnvironmentStatusREST
type NodeEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeEnvironmentSpec   `json:"spec,omitempty"`
	Status NodeEnvironmentStatus `json:"status,omitempty"`
}

// NodeEnvironmentSpec defines spec of node environment.
type NodeEnvironmentSpec struct {
	storagev1.NodeEnvironmentSpec `json:",inline"`
}

type NodeEnvironmentStatus struct {
	storagev1.NodeEnvironmentStatus `json:",inline"`
}

func (a *NodeEnvironment) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *NodeEnvironment) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *NodeEnvironment) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *NodeEnvironment) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

func (a *NodeEnvironment) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodeEnvironment) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}
