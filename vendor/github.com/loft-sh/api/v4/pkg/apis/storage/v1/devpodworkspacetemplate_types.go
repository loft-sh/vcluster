package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceTemplate holds the DevPodWorkspaceTemplate information
// +k8s:openapi-gen=true
type DevPodWorkspaceTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevPodWorkspaceTemplateSpec   `json:"spec,omitempty"`
	Status DevPodWorkspaceTemplateStatus `json:"status,omitempty"`
}

func (a *DevPodWorkspaceTemplate) GetVersions() []VersionAccessor {
	var retVersions []VersionAccessor
	for _, v := range a.Spec.Versions {
		b := v
		retVersions = append(retVersions, &b)
	}

	return retVersions
}

func (a *DevPodWorkspaceTemplateVersion) GetVersion() string {
	return a.Version
}

func (a *DevPodWorkspaceTemplate) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *DevPodWorkspaceTemplate) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *DevPodWorkspaceTemplate) GetAccess() []Access {
	return a.Spec.Access
}

func (a *DevPodWorkspaceTemplate) SetAccess(access []Access) {
	a.Spec.Access = access
}

// DevPodWorkspaceTemplateSpec holds the specification
type DevPodWorkspaceTemplateSpec struct {
	// DisplayName is the name that is shown in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the virtual cluster template
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Parameters define additional app parameters that will set provider values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Template holds the DevPod workspace template
	Template DevPodWorkspaceTemplateDefinition `json:"template,omitempty"`

	// Versions are different versions of the template that can be referenced as well
	// +optional
	Versions []DevPodWorkspaceTemplateVersion `json:"versions,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

type DevPodWorkspaceTemplateDefinition struct {
	// Provider holds the DevPod provider configuration
	Provider DevPodWorkspaceProvider `json:"provider"`

	// SpaceTemplateRef is a reference to the space that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	SpaceTemplateRef *TemplateRef `json:"spaceTemplateRef,omitempty"`

	// SpaceTemplate is the inline template for a space that should get created for this DevPod.
	// If this is specified, the kubernetes provider will be selected automatically.
	// +optional
	SpaceTemplate *SpaceTemplateDefinition `json:"spaceTemplate,omitempty"`

	// WorkspaceEnv are environment variables that should be available within the created workspace.
	// +optional
	WorkspaceEnv map[string]DevPodProviderOption `json:"workspaceEnv,omitempty"`

	// UseProjectGitCredentials specifies if the project git credentials should be used instead of local ones for this workspace
	// +optional
	UseProjectGitCredentials bool `json:"useProjectGitCredentials,omitempty"`

	// GitCloneStrategy specifies how git based workspace are being cloned. Can be "" (full, default), treeless, blobless or shallow
	// +optional
	GitCloneStrategy GitCloneStrategy `json:"gitCloneStrategy,omitempty"`
}

// +enum
type GitCloneStrategy string

// WARN: Need to match https://github.com/loft-sh/devpod/pkg/git/clone.go
const (
	FullCloneStrategy     GitCloneStrategy = ""
	BloblessCloneStrategy GitCloneStrategy = "blobless"
	TreelessCloneStrategy GitCloneStrategy = "treeless"
	ShallowCloneStrategy  GitCloneStrategy = "shallow"
)

type DevPodWorkspaceProvider struct {
	// Name is the name of the provider. This can also be an url.
	Name string `json:"name"`

	// Options are the provider option values
	// +optional
	Options map[string]DevPodProviderOption `json:"options,omitempty"`

	// Env are environment options to set when using the provider.
	// +optional
	Env map[string]DevPodProviderOption `json:"env,omitempty"`
}

type DevPodProviderOption struct {
	// Value of this option.
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom specifies a secret where this value should be taken from.
	// +optional
	ValueFrom *DevPodProviderOptionFrom `json:"valueFrom,omitempty"`
}

type DevPodProviderOptionFrom struct {
	// ProjectSecretRef is the project secret to use for this value.
	// +optional
	ProjectSecretRef *corev1.SecretKeySelector `json:"projectSecretRef,omitempty"`

	// SharedSecretRef is the shared secret to use for this value.
	// +optional
	SharedSecretRef *corev1.SecretKeySelector `json:"sharedSecretRef,omitempty"`
}

type DevPodProviderSource struct {
	// Github source for the provider
	Github string `json:"github,omitempty"`

	// File source for the provider
	File string `json:"file,omitempty"`

	// URL where the provider was downloaded from
	URL string `json:"url,omitempty"`
}

type DevPodWorkspaceTemplateVersion struct {
	// Template holds the DevPod template
	// +optional
	Template DevPodWorkspaceTemplateDefinition `json:"template,omitempty"`

	// Parameters define additional app parameters that will set provider values
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Version is the version. Needs to be in X.X.X format.
	// +optional
	Version string `json:"version,omitempty"`
}

// DevPodWorkspaceTemplateStatus holds the status
type DevPodWorkspaceTemplateStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DevPodWorkspaceTemplateList contains a list of DevPodWorkspaceTemplate
type DevPodWorkspaceTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevPodWorkspaceTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevPodWorkspaceTemplate{}, &DevPodWorkspaceTemplateList{})
}
