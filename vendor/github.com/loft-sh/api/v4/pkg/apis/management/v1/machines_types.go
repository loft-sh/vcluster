package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// Machine is a read-only mirror of one physical machine in a bare metal
// provider's inventory. Managed exclusively by the platform's inventory sync.
// +k8s:openapi-gen=true
// +resource:path=machines,rest=MachineREST
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

// MachineSpec holds the mirror's identity.
type MachineSpec struct {
	storagev1.MachineSpec `json:",inline"`
}

// MachineStatus is the observed provider-side state of the machine.
type MachineStatus struct {
	storagev1.MachineStatus `json:",inline"`
}
