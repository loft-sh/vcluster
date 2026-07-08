package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SlurmInstanceVirtualClusterResolved indicates the target tenant cluster has
	// been resolved (either an existing reference, or a VirtualClusterInstance
	// created from a template).
	SlurmInstanceVirtualClusterResolved agentstoragev1.ConditionType = "VirtualClusterResolved"

	// SlurmInstanceVirtualClusterReady indicates the target tenant cluster is up and reachable.
	SlurmInstanceVirtualClusterReady agentstoragev1.ConditionType = "VirtualClusterReady"

	// SlurmInstanceSSHKeysApplied indicates the referenced SSH keys have been applied.
	SlurmInstanceSSHKeysApplied agentstoragev1.ConditionType = "SSHKeysApplied"

	// SlurmInstanceSlurmTailnetReady indicates the login proxy access key has been
	// minted and the tailnet credentials (url/token/hostname) have been injected
	// into the tenant cluster template parameters.
	SlurmInstanceSlurmTailnetReady agentstoragev1.ConditionType = "SlurmTailnetReady"
)

var SlurmInstanceConditions = []agentstoragev1.ConditionType{
	SlurmInstanceVirtualClusterResolved,
	SlurmInstanceVirtualClusterReady,
	SlurmInstanceSSHKeysApplied,
	SlurmInstanceSlurmTailnetReady,
}

// SlurmInstancePhase describes the lifecycle phase of a SlurmInstance.
type SlurmInstancePhase string

const (
	// SlurmInstancePhasePending is the initial state of a SlurmInstance.
	SlurmInstancePhasePending SlurmInstancePhase = "Pending"
	// SlurmInstancePhaseDeploying means reconciliation is in progress.
	SlurmInstancePhaseDeploying SlurmInstancePhase = "Deploying"
	// SlurmInstancePhaseReady means the SlurmInstance is fully reconciled.
	SlurmInstancePhaseReady SlurmInstancePhase = "Ready"
	// SlurmInstancePhaseFailed means reconciliation failed.
	SlurmInstancePhaseFailed SlurmInstancePhase = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlurmInstance represents a Slurm cluster running inside a tenant cluster. The
// actual Slurm deployment (dependencies, Slinky operator and Slurm chart) is
// performed by the referenced tenant cluster template at provisioning time (or
// injected into an existing tenant cluster). The SlurmInstance object manages
// access (RBAC) and the SSH keys that are authorized on the login node.
// +k8s:openapi-gen=true
type SlurmInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SlurmInstanceSpec   `json:"spec,omitempty"`
	Status SlurmInstanceStatus `json:"status,omitempty"`
}

func (a *SlurmInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *SlurmInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *SlurmInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *SlurmInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *SlurmInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *SlurmInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

// SlurmInstanceSpec defines the desired state of a SlurmInstance.
type SlurmInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the SlurmInstance.
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object.
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams.
	// +optional
	Access []Access `json:"access,omitempty"`

	// VirtualCluster selects the tenant cluster this Slurm instance runs in.
	// Exactly one of Name (existing) or Template (create) must be set.
	VirtualCluster SlurmVirtualCluster `json:"virtualCluster"`

	// RootAuthorizedKeys references existing SSHKey resources whose public keys
	// are authorized as root on the Slurm login node.
	// +optional
	RootAuthorizedKeys []SlurmSSHKeyRef `json:"rootAuthorizedKeys,omitempty"`
}

// SlurmVirtualCluster selects the tenant cluster the Slurm instance runs in.
// The tenant cluster is always provisioned from a VirtualClusterTemplate into
// the SlurmInstance's own project (namespace).
type SlurmVirtualCluster struct {
	// Template provisions the tenant cluster from a VirtualClusterTemplate.
	Template SlurmVirtualClusterTemplate `json:"template"`
}

// SlurmVirtualClusterTemplate references a VirtualClusterTemplate used to
// provision a new tenant cluster.
type SlurmVirtualClusterTemplate struct {
	// Name is the name of the VirtualClusterTemplate to reference.
	Name string `json:"name"`

	// Version is the template version to use. Defaults to the latest version.
	// +optional
	Version string `json:"version,omitempty"`

	// InstanceName is the name of the tenant cluster instance to create.
	// Defaults to the SlurmInstance name.
	// +optional
	InstanceName string `json:"instanceName,omitempty"`

	// Parameters are values (YAML) passed to the VirtualClusterTemplate.
	// +optional
	Parameters string `json:"parameters,omitempty"`
}

// SlurmSSHKeyRef references an existing SSHKey resource.
type SlurmSSHKeyRef struct {
	// Name of the SSHKey resource.
	Name string `json:"name"`
}

// SlurmInstanceStatus defines the observed state of a SlurmInstance.
type SlurmInstanceStatus struct {
	// Phase is the current lifecycle phase of the SlurmInstance.
	// +optional
	Phase SlurmInstancePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable message indicating details about the current state.
	// +optional
	Message string `json:"message,omitempty"`

	// ResolvedCluster is the connected cluster the tenant cluster runs in.
	// +optional
	ResolvedCluster string `json:"resolvedCluster,omitempty"`

	// VirtualClusterInstance references the tenant cluster instance created when
	// provisioning from a template.
	// +optional
	VirtualClusterInstance *SlurmVirtualClusterInstanceRef `json:"virtualClusterInstance,omitempty"`

	// Conditions describe the current state of the SlurmInstance.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// SlurmVirtualClusterInstanceRef references a tenant cluster instance.
type SlurmVirtualClusterInstanceRef struct {
	// Namespace of the tenant cluster instance.
	Namespace string `json:"namespace,omitempty"`
	// Name of the tenant cluster instance.
	Name string `json:"name,omitempty"`
	// Cluster the tenant cluster runs in.
	Cluster string `json:"cluster,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlurmInstanceList contains a list of SlurmInstance.
type SlurmInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SlurmInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SlurmInstance{}, &SlurmInstanceList{})
}
