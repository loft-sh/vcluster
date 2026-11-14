package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceJoinScript holds join script request and response data for tenant cluster instances
// +subresource-request
type VirtualClusterInstanceJoinScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterInstanceJoinScriptSpec   `json:"spec,omitempty"`
	Status VirtualClusterInstanceJoinScriptStatus `json:"status,omitempty"`
}

// VirtualClusterInstanceJoinScriptSpec is the request body for the joinscript subresource.
// Profile configuration may be supplied via ProfileRef (catalog reference). If it is not
// set, the join script carries no profile-derived configuration.
type VirtualClusterInstanceJoinScriptSpec struct {
	// ProfileRef references a NodeProfile in the cluster-scoped catalog. The referenced
	// profile must exist and be permitted by the owning project's allowedNodeProfiles.
	// +optional
	ProfileRef string `json:"profileRef,omitempty"`
}

type VirtualClusterInstanceJoinScriptStatus struct {
	// JoinCommand holds the curl command that can be run on a node to join the vCluster
	// +optional
	JoinCommand string `json:"joinCommand,omitempty"`
}
