package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// MachineConfigTemplate holds the machine config template for vCluster.
// +k8s:openapi-gen=true
// +resource:path=machineconfigtemplates,rest=MachineConfigTemplateREST
type MachineConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineConfigTemplateSpec   `json:"spec,omitempty"`
	Status MachineConfigTemplateStatus `json:"status,omitempty"`
}

// MachineConfigTemplateSpec defines spec of machine config template.
type MachineConfigTemplateSpec struct {
	storagev1.MachineConfigTemplateSpec `json:",inline"`
}

type MachineConfigTemplateStatus struct {
	storagev1.MachineConfigTemplateStatus `json:",inline"`
}

func (a *MachineConfigTemplate) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *MachineConfigTemplate) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *MachineConfigTemplate) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *MachineConfigTemplate) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
