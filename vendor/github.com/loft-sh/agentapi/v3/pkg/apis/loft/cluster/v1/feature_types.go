package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Feature holds the feature information
// +k8s:openapi-gen=true
// +resource:path=features,rest=FeatureREST
type Feature struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FeatureSpec   `json:"spec,omitempty"`
	Status FeatureStatus `json:"status,omitempty"`
}

// FeatureSpec holds the specification
type FeatureSpec struct {
}

// FeatureStatus holds the status
type FeatureStatus struct {
	// Enabled signals if the feature is currently enabled or disabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}
