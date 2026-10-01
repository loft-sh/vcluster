package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster

// NodeProfile exposes reusable node runtime configuration to platform users.
// +k8s:openapi-gen=true
// +resource:path=nodeprofiles,rest=NodeProfileREST
type NodeProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeProfileSpec   `json:"spec,omitempty"`
	Status NodeProfileStatus `json:"status,omitempty"`
}

type NodeProfileSpec struct {
	storagev1.NodeProfileSpec `json:",inline"`
}

type NodeProfileStatus struct {
	storagev1.NodeProfileStatus `json:",inline"`
}
