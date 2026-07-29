package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RenderVirtualClusterTemplate holds template rendering request and response data for tenant clusters
// +k8s:openapi-gen=true
// +resource:path=rendervirtualclustertemplates,rest=RenderVirtualClusterTemplateREST
type RenderVirtualClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RenderVirtualClusterTemplateSpec   `json:"spec,omitempty"`
	Status RenderVirtualClusterTemplateStatus `json:"status,omitempty"`
}

// RenderVirtualClusterTemplateSpec holds the specification
type RenderVirtualClusterTemplateSpec struct {
	// Values is the YAML template string to render
	// +optional
	Values string `json:"values,omitempty"`

	// Parameters is the YAML parameters to apply during rendering
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// Loft contains metadata about the loft instance
	// +optional
	Loft RenderVirtualClusterTemplateLoft `json:"loft,omitempty"`
}

// RenderVirtualClusterTemplateLoft holds the loft metadata used during rendering
type RenderVirtualClusterTemplateLoft struct {
	// Name of the loft instance
	// +optional
	Name string `json:"name,omitempty"`

	// Project name
	// +optional
	Project string `json:"project,omitempty"`

	// Cluster name
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// User
	// +optional
	User string `json:"user,omitempty"`

	// Team
	// +optional
	Team string `json:"team,omitempty"`
}

// RenderVirtualClusterTemplateStatus holds the status
type RenderVirtualClusterTemplateStatus struct {
	// Values are the rendered template values
	// +optional
	Values string `json:"values,omitempty"`
}
