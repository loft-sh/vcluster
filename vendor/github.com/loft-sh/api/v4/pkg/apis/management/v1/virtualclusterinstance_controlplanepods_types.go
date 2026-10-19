package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualClusterControlPlanePods holds control plane pod information for a tenant cluster instance.
// +subresource-request
type VirtualClusterControlPlanePods struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterControlPlanePodsStatus `json:"status,omitempty"`
}

type VirtualClusterControlPlanePodsStatus struct {
	// Pods contains a reduced view of control plane pods (status and identifying metadata only).
	Pods []VirtualClusterControlPlanePod `json:"pods,omitempty"`
}

// VirtualClusterControlPlanePod is a minimal subset of core Pod fields for control plane status display.
type VirtualClusterControlPlanePod struct {
	Metadata VirtualClusterControlPlanePodObjectMeta `json:"metadata,omitempty"`
	Status   VirtualClusterControlPlanePodStatus     `json:"status,omitempty"`
}

// VirtualClusterControlPlanePodObjectMeta holds only metadata needed for list keys and terminating state.
type VirtualClusterControlPlanePodObjectMeta struct {
	Name              string       `json:"name,omitempty"`
	UID               types.UID    `json:"uid,omitempty"`
	DeletionTimestamp *metav1.Time `json:"deletionTimestamp,omitempty"`
}

// VirtualClusterControlPlanePodStatus mirrors the Pod status fields used for readiness and issue reporting.
type VirtualClusterControlPlanePodStatus struct {
	Phase                 string                                      `json:"phase,omitempty"`
	Ready                 bool                                        `json:"ready,omitempty"`
	Reason                string                                      `json:"reason,omitempty"`
	ContainerStatuses     []VirtualClusterControlPlaneContainerStatus `json:"containerStatuses,omitempty"`
	InitContainerStatuses []VirtualClusterControlPlaneContainerStatus `json:"initContainerStatuses,omitempty"`
}

// VirtualClusterControlPlaneContainerStatus is a reduced ContainerStatus (no image, hash, probe details, etc.).
type VirtualClusterControlPlaneContainerStatus struct {
	Name    string                                   `json:"name,omitempty"`
	Ready   bool                                     `json:"ready,omitempty"`
	Started *bool                                    `json:"started,omitempty"`
	State   VirtualClusterControlPlaneContainerState `json:"state,omitempty"`
}

// VirtualClusterControlPlaneContainerState only carries waiting and terminated substates used by the UI.
type VirtualClusterControlPlaneContainerState struct {
	Waiting    *VirtualClusterControlPlaneContainerStateWaiting    `json:"waiting,omitempty"`
	Terminated *VirtualClusterControlPlaneContainerStateTerminated `json:"terminated,omitempty"`
}

type VirtualClusterControlPlaneContainerStateWaiting struct {
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type VirtualClusterControlPlaneContainerStateTerminated struct {
	Reason   string `json:"reason,omitempty"`
	Message  string `json:"message,omitempty"`
	ExitCode int32  `json:"exitCode"`
}
