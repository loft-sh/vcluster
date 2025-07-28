package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VCluster",type="string",JSONPath=".spec.vClusterRef"
// +kubebuilder:printcolumn:name="NodeType",type="string",JSONPath=".spec.nodeTypeRef"
// +kubebuilder:subresource:status

// NodeClaim holds the node claim for vCluster.
// +k8s:openapi-gen=true
// +resource:path=nodeclaims,rest=NodeClaimREST,statusRest=NodeClaimStatusREST
type NodeClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeClaimSpec   `json:"spec,omitempty"`
	Status NodeClaimStatus `json:"status,omitempty"`
}

// NodeClaimSpec defines spec of node claim.
type NodeClaimSpec struct {
	storagev1.NodeClaimSpec `json:",inline"`
}

type NodeClaimStatus struct {
	storagev1.NodeClaimStatus `json:",inline"`
}
