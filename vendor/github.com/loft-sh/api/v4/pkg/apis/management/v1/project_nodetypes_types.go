package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectNodeTypes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// NodeProviders holds all the allowed node providers for the project
	NodeProviders []storagev1.NodeProvider `json:"nodeProviders,omitempty"`

	// NodeTypes holds all the allowed node types for the project
	NodeTypes []storagev1.NodeType `json:"nodeTypes,omitempty"`
}
