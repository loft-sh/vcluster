package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NetworkEnvironment conditions
	NetworkEnvironmentConditionTypeInfrastructureProvisioned = "Provisioned"
	NetworkEnvironmentConditionTypeInfrastructureSynced      = "Synced"
)

var (
	NetworkEnvironmentConditions = []agentstoragev1.ConditionType{
		NetworkEnvironmentConditionTypeInfrastructureProvisioned,
		NetworkEnvironmentConditionTypeInfrastructureSynced,
	}
)

// NetworkEnvironmentPhase defines the phase of the NetworkEnvironment
type NetworkEnvironmentPhase string

const (
	// NetworkEnvironmentPhasePending is the initial state of a NetworkEnvironment.
	NetworkEnvironmentPhasePending NetworkEnvironmentPhase = "Pending"
	// NetworkEnvironmentPhaseAvailable means the underlying network environment has been successfully provisioned.
	NetworkEnvironmentPhaseAvailable NetworkEnvironmentPhase = "Available"
	// NetworkEnvironmentPhaseFailed means the provisioning process has failed.
	NetworkEnvironmentPhaseFailed NetworkEnvironmentPhase = "Failed"
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
type NetworkEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkEnvironmentSpec   `json:"spec,omitempty"`
	Status NetworkEnvironmentStatus `json:"status,omitempty"`
}

func (a *NetworkEnvironment) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *NetworkEnvironment) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *NetworkEnvironment) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *NetworkEnvironment) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *NetworkEnvironment) GetAccess() []Access {
	return a.Spec.Access
}

func (a *NetworkEnvironment) SetAccess(access []Access) {
	a.Spec.Access = access
}

// NetworkEnvironmentSpec defines spec of network environment.
type NetworkEnvironmentSpec struct {
	// DisplayName is the name of the NodeClaim that is displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// Properties are the properties for the NetworkEnvironment.
	// +optional
	Properties map[string]string `json:"properties"`

	// ProviderRef is the name of the NodeProvider that this NetworkEnvironment is based on.
	// It may be left empty, in which case the NetworkEnvironment is a pure property container:
	// it provisions no infrastructure, becomes available immediately, carries no cleanup
	// finalizer, and can never be a provider's default environment - NodeClaims must reference
	// it explicitly. A NodeProvider that requires a network environment cannot be served by one.
	// Once set, it cannot be changed.
	// +optional
	ProviderRef string `json:"providerRef,omitempty"`
}

type NetworkEnvironmentStatus struct {
	// Phase is the current lifecycle phase of the NetworkEnvironment.
	// +optional
	Phase NetworkEnvironmentPhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions describe the current state of the platform NodeClaim.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkEnvironmentList contains a list of NetworkEnvironment
type NetworkEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkEnvironment{}, &NetworkEnvironmentList{})
}
