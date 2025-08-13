package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type NodeProviderExec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NodeProviderExecSpec   `json:"spec"`
	Status            NodeProviderExecStatus `json:"status,omitempty"`
}

type NodeProviderExecSpec struct {
	// Command is the action to perform.
	// +kubebuilder:validation:Enum=bcmTestConnection;bcmGetResources
	Command string               `json:"command"`
	Args    runtime.RawExtension `json:"args,omitempty"`
}

type NodeProviderExecStatus struct {
	// Result is the output of the executed command.
	Result runtime.RawExtension `json:"result,omitempty"`
}

type NodeProviderBCMNodeWithResources struct {
	Name      string               `json:"name"`
	Resources *corev1.ResourceList `json:"resources,omitempty"`
}

type NodeProviderBCMGetResourcesResult struct {
	Nodes      []NodeProviderBCMNodeWithResources `json:"nodes"`
	NodeGroups []string                           `json:"nodeGroups"`
}

type NodeProviderBCMTestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
