package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.capacity.available"
// +kubebuilder:printcolumn:name="Total",type="integer",JSONPath=".status.capacity.total"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:resource:scope=Cluster

// NodeType holds the information of a node type.
// +k8s:openapi-gen=true
// +resource:path=nodetypes,rest=NodeTypeREST,statusRest=NodeTypeStatusREST
type NodeType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeTypeSpec   `json:"spec,omitempty"`
	Status NodeTypeStatus `json:"status,omitempty"`
}

type NodeTypeSpec struct {
	storagev1.NodeTypeSpec `json:",inline"`
}
type NodeTypeStatus struct {
	storagev1.NodeTypeStatus `json:",inline"`
}
