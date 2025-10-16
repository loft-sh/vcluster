package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterInstanceShell creates a pod for the requesting user
// +subresource-request
type VirtualClusterInstanceShell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterShellSpec   `json:"spec,omitempty"`
	Status VirtualClusterShellStatus `json:"status,omitempty"`
}

type VirtualClusterShellSpec struct {
}

type VirtualClusterShellStatus struct {
	// PodName is the name of the shell pod that was created
	PodName string `json:"podName,omitempty"`
	// PodNamespace is the namespace of the shell pod that was created
	PodNamespace string `json:"podNamespace,omitempty"`
}
