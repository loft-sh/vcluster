package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackInstanceOutputs holds the published outputs of a StackInstance. The values live in
// the instance's managed outputs Secret, never on its status, and reading them needs the
// get verb on the stackinstances/outputs subresource.
// +subresource-request
type StackInstanceOutputs struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Outputs are the stack's published outputs, in the order publishedOutputs declares
	// them. Task outputs the stack does not publish are not listed here.
	// +optional
	Outputs []StackInstanceOutput `json:"outputs,omitempty"`
}

// StackInstanceOutput is one published output of a StackInstance.
type StackInstanceOutput struct {
	// Name is the public name the stack publishes the value under.
	Name string `json:"name"`

	// Task is the task that produced the value.
	Task string `json:"task"`

	// Sensitive reflects the output's declared source type, not its content: a value read via
	// fromSecret is sensitive, one read via fromResource is not. To have a value treated as
	// sensitive, read it from a Secret via fromSecret. Always serialized.
	Sensitive bool `json:"sensitive"`

	// State reports whether the value can be read yet.
	State StackOutputState `json:"state"`

	// Reason is the cause of the state in one word. An Available output carries one only when its
	// task failed after the value was captured, meaning the value served is the last good one.
	// +optional
	Reason StackOutputReason `json:"reason,omitempty"`

	// Message explains the reason in words. It names the task, the output or the failure behind it.
	// +optional
	Message string `json:"message,omitempty"`

	// Value is the captured value. It is set when State is Available, including when the
	// captured value is an empty string.
	// +optional
	Value *string `json:"value,omitempty"`
}

// StackOutputState reports whether a published output can be read.
type StackOutputState string

const (
	// StackOutputStateAvailable means the value is captured and returned.
	StackOutputStateAvailable StackOutputState = "Available"
	// StackOutputStatePending means no value can be served yet. Reason says whether one was never
	// captured or the stored one went stale.
	StackOutputStatePending StackOutputState = "Pending"
	// StackOutputStateFailed means the value cannot be produced: the task failed, the reference
	// names an output the stack does not declare, or the output has no source. Reason says which.
	StackOutputStateFailed StackOutputState = "Failed"
)

// StackOutputReason is the cause behind the state. It tells a stack that has to be edited from one
// that may still come good on its own.
type StackOutputReason string

const (
	// StackOutputReasonOutputNotDeclared means publishedOutputs names an unknown output. Fix the stack.
	StackOutputReasonOutputNotDeclared StackOutputReason = "OutputNotDeclared"
	// StackOutputReasonNoSource means the declared output names no source to read from. Fix the stack.
	StackOutputReasonNoSource StackOutputReason = "NoSource"
	// StackOutputReasonTaskFailed means the task producing the value failed. A value served with it
	// is the one captured before the failure.
	StackOutputReasonTaskFailed StackOutputReason = "TaskFailed"
	// StackOutputReasonNotCaptured means the value has not been read from its source yet.
	StackOutputReasonNotCaptured StackOutputReason = "NotCaptured"
	// StackOutputReasonSourceChanged means the stored value came from a source the stack no longer declares.
	StackOutputReasonSourceChanged StackOutputReason = "SourceChanged"
)
