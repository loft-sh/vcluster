package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceJoinScript holds join script request and response data for tenant cluster instances
// +subresource-request
type VirtualClusterInstanceJoinScript struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterInstanceJoinScriptStatus `json:"status,omitempty"`
}

type VirtualClusterInstanceJoinScriptStatus struct {
	// JoinCommand holds the curl command that can be run on a node to join the vCluster
	// +optional
	JoinCommand string `json:"joinCommand,omitempty"`
}
