package v1

import (
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
