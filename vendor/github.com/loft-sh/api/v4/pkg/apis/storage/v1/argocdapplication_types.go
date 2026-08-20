package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	argoapplicationsv1alpha1 "github.com/loft-sh/external-types/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	ArgoCDApplicationConditions = []agentstoragev1.ConditionType{
		ArgoCDApplicationSynced,
	}
)

const (
	ArgoCDApplicationSynced agentstoragev1.ConditionType = "Synced"

	// ArgoCDApplicationReasonTemplateNotFound is set on the Synced condition when the
	// ArgoCDApplicationTemplate referenced by spec.templateRef does not exist.
	ArgoCDApplicationReasonTemplateNotFound = "TemplateNotFound"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplication holds the information of blueprint instances
// +k8s:openapi-gen=true
type ArgoCDApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDApplicationSpec   `json:"spec,omitempty"`
	Status ArgoCDApplicationStatus `json:"status,omitempty"`
}

func (a *ArgoCDApplication) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *ArgoCDApplication) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *ArgoCDApplication) GetAccess() []Access {
	return a.Spec.Access
}

func (a *ArgoCDApplication) SetAccess(access []Access) {
	a.Spec.Access = access
}

type ArgoCDApplicationSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an OS image
	// +optional
	Description string `json:"description,omitempty"`

	// TemplateRef holds the Argo CD application template reference
	TemplateRef *ArgoCDApplicationTemplateRef `json:"templateRef,omitempty"`

	// Template is the Argo CD application template definition
	// +optional
	Template *ArgoCDApplicationTemplateDefinition `json:"template,omitempty"`

	// Parameters are values to pass to the template.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// Destination holds the Argo CD target reference
	Destination ArgoCDDestination `json:"destination,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type ArgoCDDestination struct {
	// VirtualCluster name. Mutually exclusive with Cluster.
	// +optional
	VirtualCluster *ArgoCDDestinationVirtualCluster `json:"virtualCluster,omitempty"`

	// Cluster name. Mutually exclusive with VirtualCluster.
	// +optional
	Cluster *ArgoCDDestinationCluster `json:"cluster,omitempty"`
}

type ArgoCDDestinationVirtualCluster struct {
	// Name of the tenant cluster
	// +optional
	Name string `json:"name,omitempty"`

	// Target of the tenant cluster
	// +optional
	Target ArgoCDDestinationVirtualClusterTarget `json:"target,omitempty"`
}

type ArgoCDDestinationVirtualClusterTarget string

const (
	ArgoCDDestinationVirtualClusterTargetVirtualCluster ArgoCDDestinationVirtualClusterTarget = "vCluster"
	ArgoCDDestinationVirtualClusterTargetHost           ArgoCDDestinationVirtualClusterTarget = "host"
)

type ArgoCDDestinationCluster struct {
	// Name of the cluster
	// +optional
	Name string `json:"name,omitempty"`
}

type ArgoCDApplicationTemplateRef struct {
	// Name holds the name of the blueprint template to reference.
	// +optional
	Name string `json:"name,omitempty"`
}

type ArgoCDApplicationStatus struct {
	// Conditions holds several conditions the tenant cluster might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Host of the Argo CD server
	// +optional
	Host string `json:"host,omitempty"`

	// Application holds the status of the Argo CD application
	// +optional
	Application *argoapplicationsv1alpha1.ApplicationStatus `json:"application,omitempty"`
}

func (a *ArgoCDApplication) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *ArgoCDApplication) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ArgoCDApplicationList contains a list of ArgoCDApplications
type ArgoCDApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDApplication{}, &ArgoCDApplicationList{})
}
