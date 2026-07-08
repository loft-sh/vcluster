package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlurmInstance represents a Slurm cluster running inside a tenant cluster.
// +k8s:openapi-gen=true
// +resource:path=slurminstances,rest=SlurmInstanceREST,statusRest=SlurmInstanceStatusREST
type SlurmInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SlurmInstanceSpec   `json:"spec,omitempty"`
	Status SlurmInstanceStatus `json:"status,omitempty"`
}

// SlurmInstanceSpec defines the desired state of a SlurmInstance.
type SlurmInstanceSpec struct {
	storagev1.SlurmInstanceSpec `json:",inline"`
}

// SlurmInstanceStatus defines the observed state of a SlurmInstance.
type SlurmInstanceStatus struct {
	storagev1.SlurmInstanceStatus `json:",inline"`

	// CanUse specifies if the requester can use the instance
	// +optional
	CanUse bool `json:"canUse,omitempty"`

	// CanUpdate specifies if the requester can update the instance
	// +optional
	CanUpdate bool `json:"canUpdate,omitempty"`
}
