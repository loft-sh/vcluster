package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterDebugShell creates (or returns) an ephemeral debug-shell container
// in a virtual cluster pod for the requesting user.
// +subresource-request
type VirtualClusterInstanceDebugShell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualClusterDebugShellSpec   `json:"spec,omitempty"`
	Status VirtualClusterDebugShellStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +subresource-request
type VirtualClusterInstanceDebugShellPods struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterDebugShellPodsStatus `json:"status,omitempty"`
}

// VirtualClusterDebugShellSpec defines the target pod for the debug shell.
type VirtualClusterDebugShellSpec struct {
	// PodName specifies which virtual cluster replica should get ephemeral container.
	// This is needed to virtual cluster deployed with HA (3+ replicas)
	PodName string `json:"podName,omitempty"`
}

// VirtualClusterDebugShellStatus returns the resolved pod/container for the shell.
type VirtualClusterDebugShellStatus struct {
	// ContainerName is the name of ephemeral container that was created
	ContainerName string `json:"containerName,omitempty"`

	// TargetName is the target name of ephemeral container
	TargetName string `json:"target,omitempty"`

	// PodName is the name of the virtual cluster pod
	PodName string `json:"podName,omitempty"`

	// PodNamespace is the namespace of the virtual cluster pod
	PodNamespace string `json:"podNamespace,omitempty"`
}

// VirtualClusterDebugShellPodStatus contains selectable vcluster pod metadata.
type VirtualClusterDebugShellPodStatus struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Ready     bool   `json:"ready,omitempty"`
}

type VirtualClusterDebugShellPodsStatus struct {
	// Pods are available virtual cluster pods that can be selected for debug shell.
	Pods []VirtualClusterDebugShellPodStatus `json:"pods,omitempty"`
}
