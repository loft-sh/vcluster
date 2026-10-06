package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="NodeProvider",type="string",JSONPath=".spec.providerRef"
// +kubebuilder:subresource:status

// NetworkEnvironment holds the network environment for vCluster.
// +k8s:openapi-gen=true
// +resource:path=networkenvironments,rest=NetworkEnvironmentREST,statusRest=NetworkEnvironmentStatusREST
type NetworkEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkEnvironmentSpec   `json:"spec,omitempty"`
	Status NetworkEnvironmentStatus `json:"status,omitempty"`
}

// NetworkEnvironmentSpec defines spec of network environment.
type NetworkEnvironmentSpec struct {
	storagev1.NetworkEnvironmentSpec `json:",inline"`
}

type NetworkEnvironmentStatus struct {
	storagev1.NetworkEnvironmentStatus `json:",inline"`
}

func (a *NetworkEnvironment) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *NetworkEnvironment) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *NetworkEnvironment) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *NetworkEnvironment) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}

func (a *NetworkEnvironment) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NetworkEnvironment) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}
