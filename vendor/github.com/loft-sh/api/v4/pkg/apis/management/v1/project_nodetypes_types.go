package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type ProjectNodeTypes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// NodeTypes holds all the allowed node types for the project
	NodeTypes []ProjectNodeType `json:"nodeTypes,omitempty"`
}

type ProjectNodeType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   storagev1.NodeTypeSpec `json:"spec,omitempty"`
	Status ProjectNodeTypeStatus  `json:"status,omitempty"`
}

type ProjectNodeTypeStatus struct {
	storagev1.NodeTypeStatus `json:",inline"`

	// Requirements of the node type computed from the node type properties
	Requirements []corev1.NodeSelectorRequirement `json:"requirements,omitempty"`
}
