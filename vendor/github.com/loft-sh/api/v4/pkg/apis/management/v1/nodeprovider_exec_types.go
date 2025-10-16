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
	NodeGroups []NodeProviderBCMNodeGroup         `json:"nodeGroups"`
}

type NodeProviderBCMNodeGroup struct {
	Name  string   `json:"name"`
	Nodes []string `json:"nodes"`
}

type NodeProviderBCMTestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type NodeProviderCalculateCostResult struct {
	Cost int64 `json:"cost"`
}

type NodeProviderTerraformValidateResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

type NamespacedNameArgs struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type NodeProviderExecResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
type NodeClaimData struct {
	// UserData that should be used to start the node.
	UserData string `json:"userData,omitempty"`

	// Terraform state of the node claim.
	State []byte `json:"state,omitempty"`

	// Operations that were applied to the node claim.
	Operations map[string]*Operation `json:"operations,omitempty"`
}

type NodeEnvironmentData struct {
	// Outputs of the node environment.
	Outputs []byte `json:"outputs,omitempty"`

	// Terraform state of the node environment.
	State []byte `json:"state,omitempty"`

	// Operations that were applied to the node environment.
	Operations map[string]*Operation `json:"operations,omitempty"`
}

const (
	OperationDrift   = "drift"
	OperationApply   = "apply"
	OperationDestroy = "destroy"
)

type OperationPhase string

const (
	OperationPhaseRunning OperationPhase = "Running"
	OperationPhaseSuccess OperationPhase = "Success"
	OperationPhaseFailed  OperationPhase = "Failed"
)

type Operation struct {
	// StartTimestamp of the operation.
	StartTimestamp metav1.Time `json:"startTimestamp,omitempty"`

	// EndTimestamp of the operation.
	EndTimestamp metav1.Time `json:"endTimestamp,omitempty"`

	// Phase of the operation.
	Phase OperationPhase `json:"phase,omitempty"`

	// Logs of the operation.
	Logs []byte `json:"logs,omitempty"`

	// Error of the operation.
	Error string `json:"error,omitempty"`
}
