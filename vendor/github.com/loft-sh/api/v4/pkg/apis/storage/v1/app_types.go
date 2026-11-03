package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// App holds the app information
// +k8s:openapi-gen=true
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

func (a *App) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *App) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *App) GetAccess() []Access {
	return a.Spec.Access
}

func (a *App) SetAccess(access []Access) {
	a.Spec.Access = access
}

// AppSpec holds the specification
type AppSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes an app
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Clusters are the clusters this app can be installed in.
	// +optional
	Clusters []string `json:"clusters,omitempty"`

	// RecommendedApp specifies where this app should show up as recommended app
	// +optional
	RecommendedApp []RecommendedApp `json:"recommendedApp,omitempty"`

	// AppConfig is the app configuration
	AppConfig `json:",inline"`

	// Versions are different app versions that can be referenced
	// +optional
	Versions []AppVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// DEPRECATED: Use config instead
	// manifest represents kubernetes resources that will be deployed into the target namespace
	// +optional
	Manifests string `json:"manifests,omitempty"`

	// DEPRECATED: Use config instead
	// helm defines the configuration for a helm deployment
	// +optional
	Helm *HelmConfiguration `json:"helm,omitempty"`
}

type AppVersion struct {
	// AppConfig is the app configuration
	AppConfig `json:",inline"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

type AppConfig struct {
	// DefaultNamespace is the default namespace this app should installed
	// in.
	// +optional
	DefaultNamespace string `json:"defaultNamespace,omitempty"`

	// Readme is a longer markdown string that describes the app.
	// +optional
	Readme string `json:"readme,omitempty"`

	// Icon holds an URL to the app icon
	// +optional
	Icon string `json:"icon,omitempty"`

	// Config is the helm config to use to deploy the helm release
	// +optional
	Config HelmReleaseConfig `json:"config,omitempty"`

	// Wait determines if Loft should wait during deploy for the app to become ready
	// +optional
	Wait bool `json:"wait,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs for hooks) (default 5m0s)
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Parameters define additional app parameters that will set helm values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// =======================
	// DEPRECATED FIELDS BELOW
	// =======================

	// DEPRECATED: Use config.bash instead
	// StreamContainer can be used to stream a containers logs instead of the helm output.
	// +optional
	// +internal
	StreamContainer *StreamContainer `json:"streamContainer,omitempty"`
}

type AppParameter struct {
	// Variable is the path of the variable. Can be foo or foo.bar for nested objects.
	// +optional
	Variable string `json:"variable,omitempty"`

	// Label is the label to show for this parameter
	// +optional
	Label string `json:"label,omitempty"`

	// Description is the description to show for this parameter
	// +optional
	Description string `json:"description,omitempty"`

	// Type of the parameter. Can be one of:
	// string, multiline, boolean, number and password
	// +optional
	Type string `json:"type,omitempty"`

	// Options is a slice of strings, where each string represents a mutually exclusive choice.
	// +optional
	Options []string `json:"options,omitempty"`

	// Min is the minimum number if type is number
	// +optional
	Min *int `json:"min,omitempty"`

	// Max is the maximum number if type is number
	// +optional
	Max *int `json:"max,omitempty"`

	// Required specifies if this parameter is required
	// +optional
	Required bool `json:"required,omitempty"`

	// DefaultValue is the default value if none is specified
	// +optional
	DefaultValue string `json:"defaultValue,omitempty"`

	// Placeholder shown in the UI
	// +optional
	Placeholder string `json:"placeholder,omitempty"`

	// Invalidation regex that if matched will reject the input
	// +optional
	Invalidation string `json:"invalidation,omitempty"`

	// Validation regex that if matched will allow the input
	// +optional
	Validation string `json:"validation,omitempty"`

	// Section where this app should be displayed. Apps with the same section name will be grouped together
	// +optional
	Section string `json:"section,omitempty"`
}

type UserOrTeam struct {
	// User specifies a Loft user.
	// +optional
	User string `json:"user,omitempty"`

	// Team specifies a Loft team.
	// +optional
	Team string `json:"team,omitempty"`
}

// HelmConfiguration holds the helm configuration
type HelmConfiguration struct {
	// Name of the chart to deploy
	Name string `json:"name"`

	// The additional helm values to use. Expected block string
	// +optional
	Values string `json:"values,omitempty"`

	// Version is the version of the chart to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// The repo url to use
	// +optional
	RepoURL string `json:"repoUrl,omitempty"`

	// The username to use for the selected repository
	// +optional
	Username string `json:"username,omitempty"`

	// The password to use for the selected repository
	// +optional
	Password string `json:"password,omitempty"`

	// Determines if the remote location uses an insecure
	// TLS certificate.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// AppStatus holds the status
type AppStatus struct {
}

// Target defines where an operation (e.g. an app deployment) should be executed.
type Target struct {
	// SpaceInstance defines a space instance as target
	// +optional
	SpaceInstance *TargetInstance `json:"spaceInstance,omitempty"`

	// VirtualClusterInstance defines a tenant cluster instance as target
	// +optional
	VirtualClusterInstance *TargetInstance `json:"virtualClusterInstance,omitempty"`

	// Cluster defines a connected cluster as target
	// +optional
	Cluster *TargetCluster `json:"cluster,omitempty"`
}

type TargetInstance struct {
	// Name is the name of the instance
	// +optional
	Name string `json:"name,omitempty"`

	// Project where the instance is in
	// +optional
	Project string `json:"project,omitempty"`
}

type TargetCluster struct {
	// Cluster is the cluster where the operation should get executed
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace where the operation should get executed
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type TargetVirtualCluster struct {
	// Cluster is the cluster where the tenant cluster lies
	// +optional
	Cluster string `json:"cluster,omitempty"`

	// Namespace is the namespace where the tenant cluster is located
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the tenant cluster
	// +optional
	Name string `json:"name,omitempty"`
}

type StreamContainer struct {
	// Label selector for pods. The newest matching pod will be used to stream logs from
	// +optional
	Selector metav1.LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`

	// Container is the container name to use
	// +optional
	Container string `json:"container,omitempty"`
}

type UserOrTeamEntity struct {
	// User describes an user
	// +optional
	User *EntityInfo `json:"user,omitempty"`

	// Team describes a team
	// +optional
	Team *EntityInfo `json:"team,omitempty"`
}

type EntityInfo struct {
	// Name is the kubernetes name of the object
	Name string `json:"name,omitempty"`

	// The display name shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Icon is the icon of the user / team
	// +optional
	Icon string `json:"icon,omitempty"`

	// The username that is used to login
	// +optional
	Username string `json:"username,omitempty"`

	// The users email address
	// +optional
	Email string `json:"email,omitempty"`

	// The user subject
	// +optional
	Subject string `json:"subject,omitempty"`
}

// RecommendedApp describes where an app can be displayed as recommended app
type RecommendedApp string

// Describe the status of a release
// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// RecommendedAppCluster indicates that an app should be displayed as recommended app in the cluster view
	RecommendedAppCluster RecommendedApp = "cluster"
	// RecommendedAppSpace indicates that an app should be displayed as recommended app in the space view
	RecommendedAppSpace RecommendedApp = "space"
	// RecommendedAppVirtualCluster indicates that an app should be displayed as recommended app in the tenant cluster view
	RecommendedAppVirtualCluster RecommendedApp = "virtualcluster"
)

func (x RecommendedApp) String() string { return string(x) }

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

func init() {
	SchemeBuilder.Register(&App{}, &AppList{})
}

type HelmReleaseConfig struct {
	// Chart holds information about a chart that should get deployed
	// +optional
	Chart Chart `json:"chart,omitempty"`

	// Manifests holds kube manifests that will be deployed as a chart
	// +optional
	Manifests string `json:"manifests,omitempty"`

	// Bash holds the bash script to execute in a container in the target
	// +optional
	Bash *Bash `json:"bash,omitempty"`

	// Values is the set of extra Values added to the chart.
	// These values merge with the default values inside of the chart.
	// You can use golang templating in here with values from parameters.
	// +optional
	Values string `json:"values,omitempty"`

	// Parameters are additional helm chart values that will get merged
	// with config and are then used to deploy the helm chart.
	// +optional
	Parameters string `json:"parameters,omitempty"`

	// Annotations are extra annotations for this helm release
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Bash struct {
	// Script is the script to execute.
	// +optional
	Script string `json:"script,omitempty"`

	// Image is the image to use for this app
	// +optional
	Image string `json:"image,omitempty"`

	// ClusterRole is the cluster role to use for this job
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`

	// PodSecurityContext for the bash pod.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// SecurityContext for the bash container.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

// Chart describes a chart
type Chart struct {
	// Name is the chart name in the repository
	Name string `json:"name,omitempty"`

	// Version is the chart version in the repository
	// +optional
	Version string `json:"version,omitempty"`

	// RepoURL is the repo url where the chart can be found
	// +optional
	RepoURL string `json:"repoURL,omitempty"`

	// The username that is required for this repository
	// +optional
	Username string `json:"username,omitempty"`

	// The username that is required for this repository
	// +optional
	UsernameRef *ChartSecretRef `json:"usernameRef,omitempty"`

	// The password that is required for this repository
	// +optional
	Password string `json:"password,omitempty"`

	// The password that is required for this repository
	// +optional
	PasswordRef *ChartSecretRef `json:"passwordRef,omitempty"`

	// If tls certificate checks for the chart download should be skipped
	// +optional
	InsecureSkipTlsVerify bool `json:"insecureSkipTlsVerify,omitempty"`
}

type ChartSecretRef struct {
	// ProjectSecretRef holds the reference to a project secret
	// +optional
	ProjectSecretRef *ProjectSecretRef `json:"projectSecretRef,omitempty"`
}

type ProjectSecretRef struct {
	// Project is the project name where the secret is located in.
	// +optional
	Project string `json:"project,omitempty"`

	// Name of the project secret to use.
	// +optional
	Name string `json:"name,omitempty"`

	// Key of the project secret to use.
	// +optional
	Key string `json:"key,omitempty"`
}

// Maintainer describes a Chart maintainer.
type Maintainer struct {
	// Name is a user name or organization name
	// +optional
	Name string `json:"name,omitempty"`
	// Email is an optional email address to contact the named maintainer
	// +optional
	Email string `json:"email,omitempty"`
	// URL is an optional URL to an address for the named maintainer
	// +optional
	URL string `json:"url,omitempty"`
}

// Metadata for a Chart file. This models the structure of a Chart.yaml file.
type Metadata struct {
	// The name of the chart
	// +optional
	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	// +optional
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	// +optional
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	// +optional
	Version string `json:"version,omitempty"`
	// A one-sentence description of the chart
	// +optional
	Description string `json:"description,omitempty"`
	// A list of string keywords
	// +optional
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	// +optional
	Maintainers []*Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	// +optional
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	// +optional
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	// +optional
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	// +optional
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	// +optional
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	// +optional
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Specifies the chart type: application or library
	// +optional
	Type string `json:"type,omitempty"`
	// Urls where to find the chart contents
	// +optional
	Urls []string `json:"urls,omitempty"`
}

