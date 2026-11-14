package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v5/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v5/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Network",type="string",JSONPath=".status.networkEnvironment"
// +kubebuilder:subresource:status

// NodePool keeps a set of NodeClaims for a cluster that are all built from one template.
// +k8s:openapi-gen=true
// +resource:path=nodepools,rest=NodePoolREST,statusRest=NodePoolStatusREST
type NodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodePoolSpec   `json:"spec,omitempty"`
	Status NodePoolStatus `json:"status,omitempty"`
}

// NodePoolSpec defines spec of node pool.
type NodePoolSpec struct {
	storagev1.NodePoolSpec `json:",inline"`
}

type NodePoolStatus struct {
	storagev1.NodePoolStatus `json:",inline"`
}

func (a *NodePool) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *NodePool) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *NodePool) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *NodePool) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

func (a *NodePool) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NodePool) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}
