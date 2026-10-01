package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterSchema holds config request and response data for tenant clusters
// +k8s:openapi-gen=true
// +resource:path=virtualclusterschemas,rest=VirtualClusterSchemaREST
type VirtualClusterSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterSchemaSpec   `json:"spec,omitempty"`
	Status VirtualClusterSchemaStatus `json:"status,omitempty"`
}

// VirtualClusterSchemaSpec holds the specification
type VirtualClusterSchemaSpec struct {
	// Version is the version of the tenant cluster
	Version string `json:"version,omitempty"`
}

// VirtualClusterSchemaStatus holds the status
type VirtualClusterSchemaStatus struct {
	// Schema is the schema of the tenant cluster
	Schema string `json:"schema,omitempty"`

	// DefaultValues are the default values of the tenant cluster
	DefaultValues string `json:"defaultValues,omitempty"`
}
