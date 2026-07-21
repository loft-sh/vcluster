package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NodeProfileReferenceLabel records how the joined node was configured. Its value
	// is the catalog NodeProfile name (when the join request used profileRef). Nodes
	// joined without any profile carry no such label.
	NodeProfileReferenceLabel = "vcluster.com/node-profile"
	// NodeProfileConfigSecretKey is the bootstrap-token Secret data key under which the
	// effective NodeProfileSpec is JSON-encoded when a profile is associated with a
	// joinscript request. The vCluster-side join script renderer reads from this key.
	// NOTE: Secret data keys must match [-._a-zA-Z0-9]+ (no "/"), so this is not a
	// domain-prefixed label-style key.
	NodeProfileConfigSecretKey = "vcluster.com_profile-config"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster

// NodeProfile holds reusable node runtime configuration that can be referenced by
// manual joins, auto nodes, and platform NodeClaims.
// +k8s:openapi-gen=true
type NodeProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeProfileSpec   `json:"spec,omitempty"`
	Status NodeProfileStatus `json:"status,omitempty"`
}

// NodeProfileSpec defines reusable node settings.
type NodeProfileSpec struct {
	// DisplayName is the name that should be displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes what the profile is intended for.
	// +optional
	Description string `json:"description,omitempty"`

	// Taints are taints that should be applied to the joined node.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`

	// StartupTaints are temporary taints applied while the node initializes.
	// +optional
	StartupTaints []corev1.Taint `json:"startupTaints,omitempty"`

	// NodeLabels are labels that should be applied to the joined node.
	// +optional
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`

	// NodeAnnotations are annotations that should be applied to the joined node.
	// +optional
	NodeAnnotations map[string]string `json:"nodeAnnotations,omitempty"`
}

type NodeProfileStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeProfileList contains a list of NodeProfile.
type NodeProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeProfile{}, &NodeProfileList{})
}
