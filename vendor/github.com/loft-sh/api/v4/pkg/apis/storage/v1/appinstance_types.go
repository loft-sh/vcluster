package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	AppInstanceConditions = []agentstoragev1.ConditionType{
		AppInstanceSynced,
		AppInstanceDeployed,
	}
)

const (
	AppInstanceSynced   agentstoragev1.ConditionType = "Synced"
	AppInstanceDeployed agentstoragev1.ConditionType = "Deployed"

	// AppInstanceReasonAppNotFound is set on the Synced condition when the
	// app referenced by spec.templateRef does not exist.
	AppInstanceReasonAppNotFound = "AppNotFound"

	// AppInstanceReasonDestinationNotFound is set when the space, tenant cluster or
	// cluster the instance deploys into does not exist.
	AppInstanceReasonDestinationNotFound = "DestinationNotFound"

	// AppInstanceReasonDestinationNotReady is set when that destination exists but
	// cannot take a deploy yet.
	AppInstanceReasonDestinationNotReady = "DestinationNotReady"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppInstance holds the information of app instances
// +k8s:openapi-gen=true
type AppInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppInstanceSpec   `json:"spec,omitempty"`
	Status AppInstanceStatus `json:"status,omitempty"`
}

func (a *AppInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *AppInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *AppInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *AppInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

type AppInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the app instance
	// +optional
	Description string `json:"description,omitempty"`

	// TemplateRef holds the reference to the app to deploy
	// +optional
	TemplateRef *AppInstanceTemplateRef `json:"templateRef,omitempty"`

	// Template is the inline app configuration to deploy. Mutually exclusive with TemplateRef.
	// +optional
	Template *AppConfig `json:"template,omitempty"`

	// Parameters are values to pass to the app.
	// The values should be encoded as YAML string where each parameter is represented as a top-level field key.
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// ReleaseName is the name of the helm release that is created for this app instance.
	// If empty, defaults to the name of the AppInstance.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// Destination defines where the app is deployed to
	Destination AppInstanceDestination `json:"destination,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type AppInstanceDestination struct {
	// VirtualCluster deploys the app into a virtual cluster instance. Mutually exclusive with Space and Cluster.
	// +optional
	VirtualCluster *AppInstanceDestinationVirtualCluster `json:"virtualCluster,omitempty"`

	// Space deploys the app into the namespace of a space instance. Mutually exclusive with VirtualCluster and Cluster.
	// +optional
	Space *AppInstanceDestinationSpace `json:"space,omitempty"`

	// Cluster deploys the app into a connected cluster. Mutually exclusive with VirtualCluster and Space.
	// +optional
	Cluster *AppInstanceDestinationCluster `json:"cluster,omitempty"`
}

type AppInstanceDestinationVirtualCluster struct {
	// Name of the virtual cluster instance within the project
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace the helm release is deployed into. Only used when target is vCluster;
	// for the host target the release is always deployed into the virtual cluster's host namespace.
	// If empty, defaults to the app's default namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Target selects whether the app is deployed inside the virtual cluster or into its host namespace
	// +optional
	Target AppInstanceDestinationVirtualClusterTarget `json:"target,omitempty"`
}

type AppInstanceDestinationVirtualClusterTarget string

const (
	AppInstanceDestinationVirtualClusterTargetVirtualCluster AppInstanceDestinationVirtualClusterTarget = "vCluster"
	AppInstanceDestinationVirtualClusterTargetHost           AppInstanceDestinationVirtualClusterTarget = "host"
)

type AppInstanceDestinationSpace struct {
	// Name of the space instance within the project. The helm release is deployed
	// into the namespace of the space.
	// +optional
	Name string `json:"name,omitempty"`
}

type AppInstanceDestinationCluster struct {
	// Name of the connected cluster
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace in the cluster the helm release is deployed into.
	// If empty, defaults to the app's default namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type AppInstanceTemplateRef struct {
	// Name holds the name of the app to reference.
	// +optional
	Name string `json:"name,omitempty"`

	// Version of the app to deploy. If empty, the latest version is deployed.
	// +optional
	Version string `json:"version,omitempty"`
}

type AppInstanceStatus struct {
	// Phase describes the current phase the app instance is in
	// +optional
	Phase InstancePhase `json:"phase,omitempty"`

	// Reason describes the reason in machine-readable form why the instance is in the current phase
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form why the instance is in the current phase
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration is the last generation that was reconciled
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions holds several conditions the app instance might be in
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// App holds the resolved app configuration that was last deployed
	// +optional
	App *AppConfig `json:"app,omitempty"`

	// Version is the resolved app version that was last deployed
	// +optional
	Version string `json:"version,omitempty"`

	// ReleaseName is the effective name of the helm release created by the last
	// deployment. It is recorded so the release can be uninstalled even if the
	// referenced app or its default namespace can no longer be resolved.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// ReleaseNamespace is the effective namespace the helm release was deployed
	// into by the last deployment.
	// +optional
	ReleaseNamespace string `json:"releaseNamespace,omitempty"`

	// Revision is the revision of the helm release created by the last deployment
	// +optional
	Revision int `json:"revision,omitempty"`

	// DeployAttempts counts the consecutive failed deploy attempts for the current
	// spec generation and resolved app configuration. It backs the automatic retry
	// of failed deploys and is reset whenever the deploy input changes or a deploy
	// succeeds.
	// +optional
	DeployAttempts int `json:"deployAttempts,omitempty"`

	// LastDeployTime is when the last deploy attempt finished, successful or not.
	// Together with DeployAttempts it schedules the automatic retries of failed
	// deploys.
	// +optional
	LastDeployTime *metav1.Time `json:"lastDeployTime,omitempty"`
}

func (a *AppInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *AppInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppInstanceList contains a list of AppInstances
type AppInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppInstance{}, &AppInstanceList{})
}
