package v1alpha1

import (
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Note: Application and ApplicationSet share the same field structure (TypeMeta, ObjectMeta, spec, status)
// for frontend abstraction (AbstractApplication), but spec and status have different types.
// Operation is Application-specific and not present in ApplicationSet.

// Application is a definition of Application resource.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=applications,shortName=app;apps
// +kubebuilder:printcolumn:name="Sync Status",type=string,JSONPath=`.status.sync.status`
// +kubebuilder:printcolumn:name="Health Status",type=string,JSONPath=`.status.health.status`
// +kubebuilder:printcolumn:name="Revision",type=string,JSONPath=`.status.sync.revision`,priority=10
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.project`,priority=10
type Application struct {
	metav1.TypeMeta   `json:",inline"`                                                                     // Common: shared with ApplicationSet
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`                               // Common: shared with ApplicationSet
	Spec              ApplicationSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`                     // Common: shared with ApplicationSet (different type)
	Status            ApplicationStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`       // Common: shared with ApplicationSet (different type)
	Operation         *Operation        `json:"operation,omitempty" protobuf:"bytes,4,opt,name=operation"` // Application-only: not in ApplicationSet
}

// ApplicationList is list of Application resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Application `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ApplicationSpec represents desired application state. Contains link to repository with application definition and additional parameters link definition revision.
type ApplicationSpec struct {
	// Source is a reference to the location of the application's manifests or chart
	Source *ApplicationSource `json:"source,omitempty" protobuf:"bytes,1,opt,name=source"`
	// Destination is a reference to the target Kubernetes server and namespace
	Destination ApplicationDestination `json:"destination" protobuf:"bytes,2,name=destination"`
	// Project is a reference to the project this application belongs to.
	// The empty string means that application belongs to the 'default' project.
	Project string `json:"project" protobuf:"bytes,3,name=project"`
	// SyncPolicy controls when and how a sync will be performed
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty" protobuf:"bytes,4,name=syncPolicy"`
	// IgnoreDifferences is a list of resources and their fields which should be ignored during comparison
	IgnoreDifferences IgnoreDifferences `json:"ignoreDifferences,omitempty" protobuf:"bytes,5,name=ignoreDifferences"`
	// Info contains a list of information (URLs, email addresses, and plain text) that relates to the application
	Info []Info `json:"info,omitempty" protobuf:"bytes,6,name=info"`
	// RevisionHistoryLimit limits the number of items kept in the application's revision history, which is used for informational purposes as well as for rollbacks to previous versions.
	// This should only be changed in exceptional circumstances.
	// Setting to zero will store no history. This will reduce storage used.
	// Increasing will increase the space used to store the history, so we do not recommend increasing it.
	// Default is 10.
	RevisionHistoryLimit *int64 `json:"revisionHistoryLimit,omitempty" protobuf:"bytes,7,name=revisionHistoryLimit"`

	// Sources is a reference to the location of the application's manifests or chart
	Sources ApplicationSources `json:"sources,omitempty" protobuf:"bytes,8,opt,name=sources"`

	// SourceHydrator provides a way to push hydrated manifests back to git before syncing them to the cluster.
	SourceHydrator *SourceHydrator `json:"sourceHydrator,omitempty" protobuf:"bytes,9,opt,name=sourceHydrator"`
}

// SyncPolicy controls when a sync will be performed in response to updates in git
type SyncPolicy struct {
	// Automated will keep an application synced to the target revision
	Automated *SyncPolicyAutomated `json:"automated,omitempty" protobuf:"bytes,1,opt,name=automated"`
	// Options allow you to specify whole app sync-options
	SyncOptions SyncOptions `json:"syncOptions,omitempty" protobuf:"bytes,2,opt,name=syncOptions"`
	// Retry controls failed sync retry behavior
	Retry *RetryStrategy `json:"retry,omitempty" protobuf:"bytes,3,opt,name=retry"`
	// ManagedNamespaceMetadata controls metadata in the given namespace (if CreateNamespace=true)
	ManagedNamespaceMetadata *ManagedNamespaceMetadata `json:"managedNamespaceMetadata,omitempty" protobuf:"bytes,4,opt,name=managedNamespaceMetadata"`
	// If you add a field here, be sure to update IsZero.
}

// SyncPolicyAutomated controls the behavior of an automated sync
type SyncPolicyAutomated struct {
	// Prune specifies whether to delete resources from the cluster that are not found in the sources anymore as part of automated sync (default: false)
	Prune bool `json:"prune,omitempty" protobuf:"bytes,1,opt,name=prune"`
	// SelfHeal specifies whether to revert resources back to their desired state upon modification in the cluster (default: false)
	SelfHeal bool `json:"selfHeal,omitempty" protobuf:"bytes,2,opt,name=selfHeal"`
	// AllowEmpty allows apps have zero live resources (default: false)
	AllowEmpty bool `json:"allowEmpty,omitempty" protobuf:"bytes,3,opt,name=allowEmpty"`
	// Enable allows apps to explicitly control automated sync
	Enabled *bool `json:"enabled,omitempty" protobuf:"bytes,4,opt,name=enabled"`
}

type Info struct {
	Name  string `json:"name" protobuf:"bytes,1,name=name"`
	Value string `json:"value" protobuf:"bytes,2,name=value"`
}

type SyncOptions []string

// AddOption adds a sync option to the list of sync options and returns the modified list.
// If option was already set, returns the unmodified list of sync options.
func (o SyncOptions) AddOption(option string) SyncOptions {
	if slices.Contains(o, option) {
		return o
	}
	return append(o, option)
}

// RemoveOption removes a sync option from the list of sync options and returns the modified list.
// If option has not been already set, returns the unmodified list of sync options.
func (o SyncOptions) RemoveOption(option string) SyncOptions {
	for i, j := range o {
		if j == option {
			return append(o[:i], o[i+1:]...)
		}
	}
	return o
}

// HasOption returns true if the list of sync options contains given option
func (o SyncOptions) HasOption(option string) bool {
	return slices.Contains(o, option)
}

// RetryStrategy contains information about the strategy to apply when a sync failed
type RetryStrategy struct {
	// Limit is the maximum number of attempts for retrying a failed sync. If set to 0, no retries will be performed.
	Limit int64 `json:"limit,omitempty" protobuf:"bytes,1,opt,name=limit"`
	// Backoff controls how to backoff on subsequent retries of failed syncs
	Backoff *Backoff `json:"backoff,omitempty" protobuf:"bytes,2,opt,name=backoff,casttype=Backoff"`
	// Refresh indicates if the latest revision should be used on retry instead of the initial one (default: false)
	Refresh bool `json:"refresh,omitempty" protobuf:"bytes,3,opt,name=refresh"`
}

// Backoff is the backoff strategy to use on subsequent retries for failing syncs
type Backoff struct {
	// Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")
	Duration string `json:"duration,omitempty" protobuf:"bytes,1,opt,name=duration"`
	// Factor is a factor to multiply the base duration after each failed retry
	Factor *int64 `json:"factor,omitempty" protobuf:"bytes,2,name=factor"`
	// MaxDuration is the maximum amount of time allowed for the backoff strategy
	MaxDuration string `json:"maxDuration,omitempty" protobuf:"bytes,3,opt,name=maxDuration"`
}

type ManagedNamespaceMetadata struct {
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,1,opt,name=labels"`
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,2,opt,name=annotations"`
}

// IgnoreDifferences is a list of resource filters that should be ignored during comparison
type IgnoreDifferences []ResourceIgnoreDifferences

// ResourceIgnoreDifferences contains resource filter and list of json paths which should be ignored during comparison with live state.
type ResourceIgnoreDifferences struct {
	Group             string   `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Kind              string   `json:"kind" protobuf:"bytes,2,opt,name=kind"`
	Name              string   `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	Namespace         string   `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	JSONPointers      []string `json:"jsonPointers,omitempty" protobuf:"bytes,5,opt,name=jsonPointers"`
	JQPathExpressions []string `json:"jqPathExpressions,omitempty" protobuf:"bytes,6,opt,name=jqPathExpressions"`
	// ManagedFieldsManagers is a list of trusted managers. Fields mutated by those managers will take precedence over the
	// desired state defined in the SCM and won't be displayed in diffs
	ManagedFieldsManagers []string `json:"managedFieldsManagers,omitempty" protobuf:"bytes,7,opt,name=managedFieldsManagers"`
}

// EnvEntry represents an entry in the application's environment
type EnvEntry struct {
	// Name is the name of the variable, usually expressed in uppercase
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value is the value of the variable
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

// Env is a list of environment variable entries
type Env []*EnvEntry

// ApplicationSource contains all required information about the source of an application
type ApplicationSource struct {
	// RepoURL is the URL to the repository (Git or Helm) that contains the application manifests
	RepoURL string `json:"repoURL" protobuf:"bytes,1,opt,name=repoURL"`
	// Path is a directory path within the Git repository, and is only valid for applications sourced from Git.
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
	// TargetRevision defines the revision of the source to sync the application to.
	// In case of Git, this can be commit, tag, or branch. If omitted, will equal to HEAD.
	// In case of Helm, this is a semver tag for the Chart's version.
	TargetRevision string `json:"targetRevision,omitempty" protobuf:"bytes,4,opt,name=targetRevision"`
	// Helm holds helm specific options
	Helm *ApplicationSourceHelm `json:"helm,omitempty" protobuf:"bytes,7,opt,name=helm"`
	// Kustomize holds kustomize specific options
	Kustomize *ApplicationSourceKustomize `json:"kustomize,omitempty" protobuf:"bytes,8,opt,name=kustomize"`
	// Directory holds path/directory specific options
	Directory *ApplicationSourceDirectory `json:"directory,omitempty" protobuf:"bytes,10,opt,name=directory"`
	// Plugin holds config management plugin specific options
	Plugin *ApplicationSourcePlugin `json:"plugin,omitempty" protobuf:"bytes,11,opt,name=plugin"`
	// Chart is a Helm chart name, and must be specified for applications sourced from a Helm repo.
	Chart string `json:"chart,omitempty" protobuf:"bytes,12,opt,name=chart"`
	// Ref is reference to another source within sources field. This field will not be used if used with a `source` tag.
	Ref string `json:"ref,omitempty" protobuf:"bytes,13,opt,name=ref"`
	// Name is used to refer to a source and is displayed in the UI. It is used in multi-source Applications.
	Name string `json:"name,omitempty" protobuf:"bytes,14,opt,name=name"`
}

// ApplicationSources contains list of required information about the sources of an application
type ApplicationSources []ApplicationSource

// ApplicationSourceType specifies the type of the application's source
type ApplicationSourceType string

const (
	ApplicationSourceTypeHelm      ApplicationSourceType = "Helm"
	ApplicationSourceTypeKustomize ApplicationSourceType = "Kustomize"
	ApplicationSourceTypeDirectory ApplicationSourceType = "Directory"
	ApplicationSourceTypePlugin    ApplicationSourceType = "Plugin"
)

// SourceHydrator specifies a dry "don't repeat yourself" source for manifests, a sync source from which to sync
// hydrated manifests, and an optional hydrateTo location to act as a "staging" area for hydrated manifests.
type SourceHydrator struct {
	// DrySource specifies where the dry "don't repeat yourself" manifest source lives.
	DrySource DrySource `json:"drySource" protobuf:"bytes,1,name=drySource"`
	// SyncSource specifies where to sync hydrated manifests from.
	SyncSource SyncSource `json:"syncSource" protobuf:"bytes,2,name=syncSource"`
	// HydrateTo specifies an optional "staging" location to push hydrated manifests to. An external system would then
	// have to move manifests to the SyncSource, e.g. by pull request.
	HydrateTo *HydrateTo `json:"hydrateTo,omitempty" protobuf:"bytes,3,opt,name=hydrateTo"`
}

// DrySource specifies a location for dry "don't repeat yourself" manifest source information.
type DrySource struct {
	// RepoURL is the URL to the git repository that contains the application manifests
	RepoURL string `json:"repoURL" protobuf:"bytes,1,name=repoURL"`
	// TargetRevision defines the revision of the source to hydrate
	TargetRevision string `json:"targetRevision" protobuf:"bytes,2,name=targetRevision"`
	// Path is a directory path within the Git repository where the manifests are located
	Path string `json:"path" protobuf:"bytes,3,name=path"`
	// Helm specifies helm specific options
	Helm *ApplicationSourceHelm `json:"helm,omitempty" protobuf:"bytes,4,opt,name=helm"`
	// Kustomize specifies kustomize specific options
	Kustomize *ApplicationSourceKustomize `json:"kustomize,omitempty" protobuf:"bytes,5,opt,name=kustomize"`
	// Directory specifies path/directory specific options
	Directory *ApplicationSourceDirectory `json:"directory,omitempty" protobuf:"bytes,6,opt,name=directory"`
	// Plugin specifies config management plugin specific options
	Plugin *ApplicationSourcePlugin `json:"plugin,omitempty" protobuf:"bytes,7,opt,name=plugin"`
}

// SyncSource specifies a location from which hydrated manifests may be synced.
type SyncSource struct {
	// TargetBranch is the branch from which hydrated manifests will be synced.
	// If HydrateTo is not set, this is also the branch to which hydrated manifests are committed.
	TargetBranch string `json:"targetBranch" protobuf:"bytes,1,name=targetBranch"`
	// Path is a directory path within the git repository where hydrated manifests should be committed to and synced from.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^.{2,}|[^./]$`
	Path string `json:"path" protobuf:"bytes,2,name=path"`
}

// HydrateTo specifies a location to which hydrated manifests should be pushed as a "staging area".
type HydrateTo struct {
	// TargetBranch is the branch to which hydrated manifests should be committed
	TargetBranch string `json:"targetBranch" protobuf:"bytes,1,name=targetBranch"`
}

// RefreshType specifies how to refresh the sources of a given application
type RefreshType string

const (
	RefreshTypeNormal RefreshType = "normal"
	RefreshTypeHard   RefreshType = "hard"
)

// HydrateType specifies the type of hydration
type HydrateType string

const (
	// HydrateTypeNormal is a normal hydration
	HydrateTypeNormal HydrateType = "normal"
)

// ApplicationSourceHelm holds helm specific options
type ApplicationSourceHelm struct {
	// ValuesFiles is a list of Helm value files to use when generating a template
	ValueFiles []string `json:"valueFiles,omitempty" protobuf:"bytes,1,opt,name=valueFiles"`
	// Parameters is a list of Helm parameters which are passed to the helm template command upon manifest generation
	Parameters []HelmParameter `json:"parameters,omitempty" protobuf:"bytes,2,opt,name=parameters"`
	// ReleaseName is the Helm release name to use. If omitted it will use the application name
	ReleaseName string `json:"releaseName,omitempty" protobuf:"bytes,3,opt,name=releaseName"`
	// Values specifies Helm values to be passed to helm template, typically defined as a block. ValuesObject takes precedence over Values, so use one or the other.
	// +patchStrategy=replace
	Values string `json:"values,omitempty" patchStrategy:"replace" protobuf:"bytes,4,opt,name=values"`
	// FileParameters are file parameters to the helm template
	FileParameters []HelmFileParameter `json:"fileParameters,omitempty" protobuf:"bytes,5,opt,name=fileParameters"`
	// Version is the Helm version to use for templating ("3")
	Version string `json:"version,omitempty" protobuf:"bytes,6,opt,name=version"`
	// PassCredentials pass credentials to all domains (Helm's --pass-credentials)
	PassCredentials bool `json:"passCredentials,omitempty" protobuf:"bytes,7,opt,name=passCredentials"`
	// IgnoreMissingValueFiles prevents helm template from failing when valueFiles do not exist locally by not appending them to helm template --values
	IgnoreMissingValueFiles bool `json:"ignoreMissingValueFiles,omitempty" protobuf:"bytes,8,opt,name=ignoreMissingValueFiles"`
	// SkipCrds skips custom resource definition installation step (Helm's --skip-crds)
	SkipCrds bool `json:"skipCrds,omitempty" protobuf:"bytes,9,opt,name=skipCrds"`
	// ValuesObject specifies Helm values to be passed to helm template, defined as a map. This takes precedence over Values.
	// +kubebuilder:pruning:PreserveUnknownFields
	ValuesObject *runtime.RawExtension `json:"valuesObject,omitempty" protobuf:"bytes,10,opt,name=valuesObject"`
	// Namespace is an optional namespace to template with. If left empty, defaults to the app's destination namespace.
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,11,opt,name=namespace"`
	// KubeVersion specifies the Kubernetes API version to pass to Helm when templating manifests. By default, Argo CD
	// uses the Kubernetes version of the target cluster.
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,12,opt,name=kubeVersion"`
	// APIVersions specifies the Kubernetes resource API versions to pass to Helm when templating manifests. By default,
	// Argo CD uses the API versions of the target cluster. The format is [group/]version/kind.
	APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,13,opt,name=apiVersions"`
	// SkipTests skips test manifest installation step (Helm's --skip-tests).
	SkipTests bool `json:"skipTests,omitempty" protobuf:"bytes,14,opt,name=skipTests"`
	// SkipSchemaValidation skips JSON schema validation (Helm's --skip-schema-validation)
	SkipSchemaValidation bool `json:"skipSchemaValidation,omitempty" protobuf:"bytes,15,opt,name=skipSchemaValidation"`
}

// HelmParameter is a parameter that's passed to helm template during manifest generation
type HelmParameter struct {
	// Name is the name of the Helm parameter
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Value is the value for the Helm parameter
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	// ForceString determines whether to tell Helm to interpret booleans and numbers as strings
	ForceString bool `json:"forceString,omitempty" protobuf:"bytes,3,opt,name=forceString"`
}

// HelmFileParameter is a file parameter that's passed to helm template during manifest generation
type HelmFileParameter struct {
	// Name is the name of the Helm parameter
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Path is the path to the file containing the values for the Helm parameter
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
}

// KustomizeImage represents a Kustomize image definition in the format [old_image_name=]<image_name>:<image_tag>
type KustomizeImage string

// KustomizeImages is a list of Kustomize images
type KustomizeImages []KustomizeImage

// ApplicationSourceKustomize holds options specific to an Application source specific to Kustomize
type ApplicationSourceKustomize struct {
	// NamePrefix overrides the namePrefix in the kustomization.yaml for Kustomize apps
	NamePrefix string `json:"namePrefix,omitempty" protobuf:"bytes,1,opt,name=namePrefix"`
	// NameSuffix overrides the nameSuffix in the kustomization.yaml for Kustomize apps
	NameSuffix string `json:"nameSuffix,omitempty" protobuf:"bytes,2,opt,name=nameSuffix"`
	// Images is a list of Kustomize image override specifications
	Images KustomizeImages `json:"images,omitempty" protobuf:"bytes,3,opt,name=images"`
	// CommonLabels is a list of additional labels to add to rendered manifests
	CommonLabels map[string]string `json:"commonLabels,omitempty" protobuf:"bytes,4,opt,name=commonLabels"`
	// Version controls which version of Kustomize to use for rendering manifests
	Version string `json:"version,omitempty" protobuf:"bytes,5,opt,name=version"`
	// CommonAnnotations is a list of additional annotations to add to rendered manifests
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty" protobuf:"bytes,6,opt,name=commonAnnotations"`
	// ForceCommonLabels specifies whether to force applying common labels to resources for Kustomize apps
	ForceCommonLabels bool `json:"forceCommonLabels,omitempty" protobuf:"bytes,7,opt,name=forceCommonLabels"`
	// ForceCommonAnnotations specifies whether to force applying common annotations to resources for Kustomize apps
	ForceCommonAnnotations bool `json:"forceCommonAnnotations,omitempty" protobuf:"bytes,8,opt,name=forceCommonAnnotations"`
	// Namespace sets the namespace that Kustomize adds to all resources
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,9,opt,name=namespace"`
	// CommonAnnotationsEnvsubst specifies whether to apply env variables substitution for annotation values
	CommonAnnotationsEnvsubst bool `json:"commonAnnotationsEnvsubst,omitempty" protobuf:"bytes,10,opt,name=commonAnnotationsEnvsubst"`
	// Replicas is a list of Kustomize Replicas override specifications
	Replicas KustomizeReplicas `json:"replicas,omitempty" protobuf:"bytes,11,opt,name=replicas"`
	// Patches is a list of Kustomize patches
	Patches KustomizePatches `json:"patches,omitempty" protobuf:"bytes,12,opt,name=patches"`
	// Components specifies a list of kustomize components to add to the kustomization before building
	Components []string `json:"components,omitempty" protobuf:"bytes,13,rep,name=components"`
	// IgnoreMissingComponents prevents kustomize from failing when components do not exist locally
	IgnoreMissingComponents bool `json:"ignoreMissingComponents,omitempty" protobuf:"bytes,17,opt,name=ignoreMissingComponents"`
	// LabelWithoutSelector specifies whether to apply common labels to resource selectors or not
	LabelWithoutSelector bool `json:"labelWithoutSelector,omitempty" protobuf:"bytes,14,opt,name=labelWithoutSelector"`
	// KubeVersion specifies the Kubernetes API version to pass to Helm when templating manifests.
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,15,opt,name=kubeVersion"`
	// APIVersions specifies the Kubernetes resource API versions to pass to Helm when templating manifests.
	APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,16,opt,name=apiVersions"`
	// LabelIncludeTemplates specifies whether to apply common labels to resource templates or not
	LabelIncludeTemplates bool `json:"labelIncludeTemplates,omitempty" protobuf:"bytes,18,opt,name=labelIncludeTemplates"`
}

// KustomizeReplica defines a Kustomize replica override
type KustomizeReplica struct {
	// Name of Deployment or StatefulSet
	Name string `json:"name" protobuf:"bytes,1,name=name"`
	// Number of replicas
	Count intstr.IntOrString `json:"count" protobuf:"bytes,2,name=count"`
}

// KustomizeReplicas is a list of KustomizeReplica overrides
type KustomizeReplicas []KustomizeReplica

// KustomizePatches is a list of Kustomize patches
type KustomizePatches []KustomizePatch

// KustomizePatch represents a Kustomize patch
type KustomizePatch struct {
	Path    string             `json:"path,omitempty" yaml:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	Patch   string             `json:"patch,omitempty" yaml:"patch,omitempty" protobuf:"bytes,2,opt,name=patch"`
	Target  *KustomizeSelector `json:"target,omitempty" yaml:"target,omitempty" protobuf:"bytes,3,opt,name=target"`
	Options map[string]bool    `json:"options,omitempty" yaml:"options,omitempty" protobuf:"bytes,4,opt,name=options"`
}

// KustomizeSelector is a selector for Kustomize patches
type KustomizeSelector struct {
	KustomizeResId     `json:",inline,omitempty" yaml:",inline,omitempty" protobuf:"bytes,1,opt,name=resId"`
	AnnotationSelector string `json:"annotationSelector,omitempty" yaml:"annotationSelector,omitempty" protobuf:"bytes,2,opt,name=annotationSelector"`
	LabelSelector      string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty" protobuf:"bytes,3,opt,name=labelSelector"`
}

// KustomizeResId is a resource identifier for Kustomize
type KustomizeResId struct {
	KustomizeGvk `json:",inline,omitempty" yaml:",inline,omitempty" protobuf:"bytes,1,opt,name=gvk"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
	Namespace    string `json:"namespace,omitempty" yaml:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
}

// KustomizeGvk is a group/version/kind for Kustomize
type KustomizeGvk struct {
	Group   string `json:"group,omitempty" yaml:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version,omitempty" yaml:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	Kind    string `json:"kind,omitempty" yaml:"kind,omitempty" protobuf:"bytes,3,opt,name=kind"`
}

// JsonnetVar represents a variable to be passed to jsonnet during manifest generation
type JsonnetVar struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
	Code  bool   `json:"code,omitempty" protobuf:"bytes,3,opt,name=code"`
}

// ApplicationSourceJsonnet holds options specific to applications of type Jsonnet
type ApplicationSourceJsonnet struct {
	// ExtVars is a list of Jsonnet External Variables
	ExtVars []JsonnetVar `json:"extVars,omitempty" protobuf:"bytes,1,opt,name=extVars"`
	// TLAS is a list of Jsonnet Top-level Arguments
	TLAs []JsonnetVar `json:"tlas,omitempty" protobuf:"bytes,2,opt,name=tlas"`
	// Additional library search dirs
	Libs []string `json:"libs,omitempty" protobuf:"bytes,3,opt,name=libs"`
}

// ApplicationSourceDirectory holds options for applications of type plain YAML or Jsonnet
type ApplicationSourceDirectory struct {
	// Recurse specifies whether to scan a directory recursively for manifests
	Recurse bool `json:"recurse,omitempty" protobuf:"bytes,1,opt,name=recurse"`
	// Jsonnet holds options specific to Jsonnet
	Jsonnet ApplicationSourceJsonnet `json:"jsonnet,omitempty,omitzero" protobuf:"bytes,2,opt,name=jsonnet"`
	// Exclude contains a glob pattern to match paths against that should be explicitly excluded from being used during manifest generation
	Exclude string `json:"exclude,omitempty" protobuf:"bytes,3,opt,name=exclude"`
	// Include contains a glob pattern to match paths against that should be explicitly included during manifest generation
	Include string `json:"include,omitempty" protobuf:"bytes,4,opt,name=include"`
}

// OptionalMap is an optional map parameter for plugins
type OptionalMap struct {
	// Map is the value of a map type parameter.
	// +optional
	Map map[string]string `json:"map" protobuf:"bytes,1,rep,name=map"`
}

// OptionalArray is an optional array parameter for plugins
type OptionalArray struct {
	// Array is the value of an array type parameter.
	// +optional
	Array []string `json:"array" protobuf:"bytes,1,rep,name=array"`
}

// ApplicationSourcePluginParameter is a parameter for a config management plugin
type ApplicationSourcePluginParameter struct {
	// Name is the name identifying a parameter.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// String_ is the value of a string type parameter.
	String_ *string `json:"string,omitempty" protobuf:"bytes,5,opt,name=string"` //nolint:revive //FIXME(var-naming)
	// Map is the value of a map type parameter.
	*OptionalMap `json:",omitempty" protobuf:"bytes,3,rep,name=map"`
	// Array is the value of an array type parameter.
	*OptionalArray `json:",omitempty" protobuf:"bytes,4,rep,name=array"`
}

// ApplicationSourcePluginParameters is a list of plugin parameters
type ApplicationSourcePluginParameters []ApplicationSourcePluginParameter

// ApplicationSourcePlugin holds options specific to config management plugins
type ApplicationSourcePlugin struct {
	Name       string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Env        `json:"env,omitempty" protobuf:"bytes,2,opt,name=env"`
	Parameters ApplicationSourcePluginParameters `json:"parameters,omitempty" protobuf:"bytes,3,opt,name=parameters"`
}

// ResourceHealthLocation indicates where the resource health status is stored
type ResourceHealthLocation string

var (
	ResourceHealthLocationInline  ResourceHealthLocation
	ResourceHealthLocationAppTree ResourceHealthLocation = "appTree"
)

// ApplicationStatus contains status information for the application
type ApplicationStatus struct {
	// Resources is a list of Kubernetes resources managed by this application
	Resources []ResourceStatus `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	// Sync contains information about the application's current sync status
	Sync SyncStatus `json:"sync,omitempty" protobuf:"bytes,2,opt,name=sync"`
	// Health contains information about the application's current health status
	Health AppHealthStatus `json:"health,omitempty" protobuf:"bytes,3,opt,name=health"`
	// History contains information about the application's sync history
	History RevisionHistories `json:"history,omitempty" protobuf:"bytes,4,opt,name=history"`
	// Conditions is a list of currently observed application conditions
	Conditions []ApplicationCondition `json:"conditions,omitempty" protobuf:"bytes,5,opt,name=conditions"`
	// ReconciledAt indicates when the application state was reconciled using the latest git version
	ReconciledAt *metav1.Time `json:"reconciledAt,omitempty" protobuf:"bytes,6,opt,name=reconciledAt"`
	// OperationState contains information about any ongoing operations, such as a sync
	OperationState *OperationState `json:"operationState,omitempty" protobuf:"bytes,7,opt,name=operationState"`
	// ObservedAt indicates when the application state was updated without querying latest git state
	//
	// Deprecated: controller no longer updates ObservedAt field
	ObservedAt *metav1.Time `json:"observedAt,omitempty" protobuf:"bytes,8,opt,name=observedAt"`
	// SourceType specifies the type of this application
	SourceType ApplicationSourceType `json:"sourceType,omitempty" protobuf:"bytes,9,opt,name=sourceType"`
	// Summary contains a list of URLs and container images used by this application
	Summary ApplicationSummary `json:"summary,omitempty" protobuf:"bytes,10,opt,name=summary"`
	// ResourceHealthSource indicates where the resource health status is stored: inline if not set or appTree
	ResourceHealthSource ResourceHealthLocation `json:"resourceHealthSource,omitempty" protobuf:"bytes,11,opt,name=resourceHealthSource"`
	// SourceTypes specifies the type of the sources included in the application
	SourceTypes []ApplicationSourceType `json:"sourceTypes,omitempty" protobuf:"bytes,12,opt,name=sourceTypes"`
	// ControllerNamespace indicates the namespace in which the application controller is located
	ControllerNamespace string `json:"controllerNamespace,omitempty" protobuf:"bytes,13,opt,name=controllerNamespace"`
	// SourceHydrator stores information about the current state of source hydration
	SourceHydrator SourceHydratorStatus `json:"sourceHydrator,omitempty" protobuf:"bytes,14,opt,name=sourceHydrator"`
}

// ApplicationSummary contains information about URLs and container images used by an application
type ApplicationSummary struct {
	// ExternalURLs holds all external URLs of application child resources.
	ExternalURLs []string `json:"externalURLs,omitempty" protobuf:"bytes,1,opt,name=externalURLs"`
	// Images holds all images of application child resources.
	Images []string `json:"images,omitempty" protobuf:"bytes,2,opt,name=images"`
}

// SourceHydratorStatus contains information about the current state of source hydration
type SourceHydratorStatus struct {
	// LastSuccessfulOperation holds info about the most recent successful hydration
	LastSuccessfulOperation *SuccessfulHydrateOperation `json:"lastSuccessfulOperation,omitempty" protobuf:"bytes,1,opt,name=lastSuccessfulOperation"`
	// CurrentOperation holds the status of the hydrate operation
	CurrentOperation *HydrateOperation `json:"currentOperation,omitempty" protobuf:"bytes,2,opt,name=currentOperation"`
}

// HydrateOperation contains information about the most recent hydrate operation
type HydrateOperation struct {
	// StartedAt indicates when the hydrate operation started
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,1,opt,name=startedAt"`
	// FinishedAt indicates when the hydrate operation finished
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,2,opt,name=finishedAt"`
	// Phase indicates the status of the hydrate operation
	Phase HydrateOperationPhase `json:"phase" protobuf:"bytes,3,opt,name=phase"`
	// Message contains a message describing the current status of the hydrate operation
	Message string `json:"message" protobuf:"bytes,4,opt,name=message"`
	// DrySHA holds the resolved revision (sha) of the dry source as of the most recent reconciliation
	DrySHA string `json:"drySHA,omitempty" protobuf:"bytes,5,opt,name=drySHA"`
	// HydratedSHA holds the resolved revision (sha) of the hydrated source as of the most recent reconciliation
	HydratedSHA string `json:"hydratedSHA,omitempty" protobuf:"bytes,6,opt,name=hydratedSHA"`
	// SourceHydrator holds the hydrator config used for the hydrate operation
	SourceHydrator SourceHydrator `json:"sourceHydrator,omitempty" protobuf:"bytes,7,opt,name=sourceHydrator"`
}

// SuccessfulHydrateOperation contains information about the most recent successful hydrate operation
type SuccessfulHydrateOperation struct {
	// DrySHA holds the resolved revision (sha) of the dry source as of the most recent reconciliation
	DrySHA string `json:"drySHA,omitempty" protobuf:"bytes,5,opt,name=drySHA"`
	// HydratedSHA holds the resolved revision (sha) of the hydrated source as of the most recent reconciliation
	HydratedSHA string `json:"hydratedSHA,omitempty" protobuf:"bytes,6,opt,name=hydratedSHA"`
	// SourceHydrator holds the hydrator config used for the hydrate operation
	SourceHydrator SourceHydrator `json:"sourceHydrator,omitempty" protobuf:"bytes,7,opt,name=sourceHydrator"`
}

// HydrateOperationPhase indicates the status of a hydrate operation
// +kubebuilder:validation:Enum=Hydrating;Failed;Hydrated
type HydrateOperationPhase string

const (
	HydrateOperationPhaseHydrating HydrateOperationPhase = "Hydrating"
	HydrateOperationPhaseFailed    HydrateOperationPhase = "Failed"
	HydrateOperationPhaseHydrated  HydrateOperationPhase = "Hydrated"
)

// OperationPhase is the phase of an operation
type OperationPhase string

// ResultCode is the result of a sync operation on a specific resource
type ResultCode string

// HookType is the type of a hook
type HookType string

// SyncPhase is the phase of a sync operation
type SyncPhase string

// HealthStatusCode is the health status of a resource
type HealthStatusCode string

// OperationInitiator contains information about the initiator of an operation
type OperationInitiator struct {
	// Username contains the name of a user who started operation
	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	// Automated is set to true if operation was initiated automatically by the application controller.
	Automated bool `json:"automated,omitempty" protobuf:"bytes,2,opt,name=automated"`
}

// Operation contains information about a requested or running operation
type Operation struct {
	// Sync contains parameters for the operation
	Sync *SyncOperation `json:"sync,omitempty" protobuf:"bytes,1,opt,name=sync"`
	// InitiatedBy contains information about who initiated the operations
	InitiatedBy OperationInitiator `json:"initiatedBy,omitempty" protobuf:"bytes,2,opt,name=initiatedBy"`
	// Info is a list of informational items for this operation
	Info []*Info `json:"info,omitempty" protobuf:"bytes,3,name=info"`
	// Retry controls the strategy to apply if a sync fails
	Retry RetryStrategy `json:"retry,omitempty" protobuf:"bytes,4,opt,name=retry"`
}

// SyncOperationResource contains resources to sync.
type SyncOperationResource struct {
	Group     string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Kind      string `json:"kind" protobuf:"bytes,2,opt,name=kind"`
	Name      string `json:"name" protobuf:"bytes,3,opt,name=name"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	Exclude   bool   `json:"-"`
}

// RevisionHistories is an array of history, oldest first and newest last
type RevisionHistories []RevisionHistory

// SyncOperation contains details about a sync operation.
type SyncOperation struct {
	// Revision is the revision (Git) or chart version (Helm) which to sync the application to.
	// If omitted, will use the revision specified in app spec.
	Revision string `json:"revision,omitempty" protobuf:"bytes,1,opt,name=revision"`
	// Prune specifies to delete resources from the cluster that are no longer tracked in git
	Prune bool `json:"prune,omitempty" protobuf:"bytes,2,opt,name=prune"`
	// DryRun specifies to perform a `kubectl apply --dry-run` without actually performing the sync
	DryRun bool `json:"dryRun,omitempty" protobuf:"bytes,3,opt,name=dryRun"`
	// SyncStrategy describes how to perform the sync
	SyncStrategy *SyncStrategy `json:"syncStrategy,omitempty" protobuf:"bytes,4,opt,name=syncStrategy"`
	// Resources describes which resources shall be part of the sync
	Resources []SyncOperationResource `json:"resources,omitempty" protobuf:"bytes,6,opt,name=resources"`
	// Source overrides the source definition set in the application.
	// This is typically set in a Rollback operation and is nil during a Sync operation
	Source *ApplicationSource `json:"source,omitempty" protobuf:"bytes,7,opt,name=source"`
	// Manifests is an optional field that overrides sync source with a local directory for development
	Manifests []string `json:"manifests,omitempty" protobuf:"bytes,8,opt,name=manifests"`
	// SyncOptions provide per-sync sync-options, e.g. Validate=false
	SyncOptions SyncOptions `json:"syncOptions,omitempty" protobuf:"bytes,9,opt,name=syncOptions"`
	// Sources overrides the source definition set in the application.
	// This is typically set in a Rollback operation and is nil during a Sync operation
	Sources ApplicationSources `json:"sources,omitempty" protobuf:"bytes,10,opt,name=sources"`
	// Revisions is the list of revision (Git) or chart version (Helm) which to sync each source in sources field for the application to.
	// If omitted, will use the revision specified in app spec.
	Revisions []string `json:"revisions,omitempty" protobuf:"bytes,11,opt,name=revisions"`
	// SelfHealAttemptsCount contains the number of auto-heal attempts
	SelfHealAttemptsCount int64 `json:"autoHealAttemptsCount,omitempty" protobuf:"bytes,12,opt,name=autoHealAttemptsCount"`
}

// OperationState contains information about state of a running operation
type OperationState struct {
	// Operation is the original requested operation
	Operation Operation `json:"operation" protobuf:"bytes,1,opt,name=operation"`
	// Phase is the current phase of the operation
	Phase OperationPhase `json:"phase" protobuf:"bytes,2,opt,name=phase"`
	// Message holds any pertinent messages when attempting to perform operation (typically errors).
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
	// SyncResult is the result of a Sync operation
	SyncResult *SyncOperationResult `json:"syncResult,omitempty" protobuf:"bytes,4,opt,name=syncResult"`
	// StartedAt contains time of operation start
	StartedAt metav1.Time `json:"startedAt" protobuf:"bytes,6,opt,name=startedAt"`
	// FinishedAt contains time of operation completion
	FinishedAt *metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,7,opt,name=finishedAt"`
	// RetryCount contains time of operation retries
	RetryCount int64 `json:"retryCount,omitempty" protobuf:"bytes,8,opt,name=retryCount"`
}

// SyncStrategy contains sync strategy options
type SyncStrategy struct {
	// Apply will perform a `kubectl apply` to perform the sync.
	Apply *SyncStrategyApply `json:"apply,omitempty" protobuf:"bytes,1,opt,name=apply"`
	// Hook will submit any referenced resources to perform the sync. This is the default strategy
	Hook *SyncStrategyHook `json:"hook,omitempty" protobuf:"bytes,2,opt,name=hook"`
}

// SyncStrategyApply uses `kubectl apply` to perform the apply
type SyncStrategyApply struct {
	// Force indicates whether or not to supply the --force flag to `kubectl apply`.
	// The --force flag deletes and re-create the resource, when PATCH encounters conflict and has
	// retried for 5 times.
	Force bool `json:"force,omitempty" protobuf:"bytes,1,opt,name=force"`
}

// SyncStrategyHook will perform a sync using hooks annotations.
// If no hook annotation is specified falls back to `kubectl apply`.
type SyncStrategyHook struct {
	// Embed SyncStrategyApply type to inherit any `apply` options
	// +optional
	SyncStrategyApply `json:",inline" protobuf:"bytes,1,opt,name=syncStrategyApply"`
}

// SyncOperationResult represent result of sync operation
type SyncOperationResult struct {
	// Resources contains a list of sync result items for each individual resource in a sync operation
	Resources ResourceResults `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	// Revision holds the revision this sync operation was performed to
	Revision string `json:"revision" protobuf:"bytes,2,opt,name=revision"`
	// Source records the application source information of the sync, used for comparing auto-sync
	Source ApplicationSource `json:"source,omitempty" protobuf:"bytes,3,opt,name=source"`
	// Sources records the application source information of the sync, used for comparing auto-sync
	Sources ApplicationSources `json:"sources,omitempty" protobuf:"bytes,4,opt,name=sources"`
	// Revisions holds the revision this sync operation was performed for respective indexed source in sources field
	Revisions []string `json:"revisions,omitempty" protobuf:"bytes,5,opt,name=revisions"`
	// ManagedNamespaceMetadata contains the current sync state of managed namespace metadata
	ManagedNamespaceMetadata *ManagedNamespaceMetadata `json:"managedNamespaceMetadata,omitempty" protobuf:"bytes,6,opt,name=managedNamespaceMetadata"`
}

// ResourceResult holds the operation result details of a specific resource
type ResourceResult struct {
	// Group specifies the API group of the resource
	Group string `json:"group" protobuf:"bytes,1,opt,name=group"`
	// Version specifies the API version of the resource
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	// Kind specifies the API kind of the resource
	Kind string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
	// Namespace specifies the target namespace of the resource
	Namespace string `json:"namespace" protobuf:"bytes,4,opt,name=namespace"`
	// Name specifies the name of the resource
	Name string `json:"name" protobuf:"bytes,5,opt,name=name"`
	// Status holds the final result of the sync. Will be empty if the resources is yet to be applied/pruned and is always zero-value for hooks
	Status ResultCode `json:"status,omitempty" protobuf:"bytes,6,opt,name=status"`
	// Message contains an informational or error message for the last sync OR operation
	Message string `json:"message,omitempty" protobuf:"bytes,7,opt,name=message"`
	// HookType specifies the type of the hook. Empty for non-hook resources
	HookType HookType `json:"hookType,omitempty" protobuf:"bytes,8,opt,name=hookType"`
	// HookPhase contains the state of any operation associated with this resource OR hook
	HookPhase OperationPhase `json:"hookPhase,omitempty" protobuf:"bytes,9,opt,name=hookPhase"`
	// SyncPhase indicates the particular phase of the sync that this result was acquired in
	SyncPhase SyncPhase `json:"syncPhase,omitempty" protobuf:"bytes,10,opt,name=syncPhase"`
	// Images contains the images related to the ResourceResult
	Images []string `json:"images,omitempty" protobuf:"bytes,11,opt,name=images"`
}

// ResourceResults defines a list of resource results for a given operation
type ResourceResults []*ResourceResult

// RevisionHistory contains history information about a previous sync
type RevisionHistory struct {
	// Revision holds the revision the sync was performed against
	Revision string `json:"revision,omitempty" protobuf:"bytes,2,opt,name=revision"`
	// DeployedAt holds the time the sync operation completed
	DeployedAt metav1.Time `json:"deployedAt" protobuf:"bytes,4,opt,name=deployedAt"`
	// ID is an auto incrementing identifier of the RevisionHistory
	ID int64 `json:"id" protobuf:"bytes,5,opt,name=id"`
	// Source is a reference to the application source used for the sync operation
	Source ApplicationSource `json:"source,omitempty" protobuf:"bytes,6,opt,name=source"`
	// DeployStartedAt holds the time the sync operation started
	DeployStartedAt *metav1.Time `json:"deployStartedAt,omitempty" protobuf:"bytes,7,opt,name=deployStartedAt"`
	// Sources is a reference to the application sources used for the sync operation
	Sources ApplicationSources `json:"sources,omitempty" protobuf:"bytes,8,opt,name=sources"`
	// Revisions holds the revision of each source in sources field the sync was performed against
	Revisions []string `json:"revisions,omitempty" protobuf:"bytes,9,opt,name=revisions"`
	// InitiatedBy contains information about who initiated the operations
	InitiatedBy OperationInitiator `json:"initiatedBy,omitempty" protobuf:"bytes,10,opt,name=initiatedBy"`
}

// SyncStatusCode is a type which represents possible comparison results
type SyncStatusCode string

// Possible comparison results
const (
	// SyncStatusCodeUnknown indicates that the status of a sync could not be reliably determined
	SyncStatusCodeUnknown SyncStatusCode = "Unknown"
	// SyncStatusCodeSynced indicates that desired and live states match
	SyncStatusCodeSynced SyncStatusCode = "Synced"
	// SyncStatusCodeOutOfSync indicates that there is a drift between desired and live states
	SyncStatusCodeOutOfSync SyncStatusCode = "OutOfSync"
)

// ApplicationConditionType represents type of application condition. Type name has following convention:
// prefix "Error" means error condition
// prefix "Warning" means warning condition
// prefix "Info" means informational condition
type ApplicationConditionType = string

const (
	// ApplicationConditionDeletionError indicates that controller failed to delete application
	ApplicationConditionDeletionError = "DeletionError"
	// ApplicationConditionInvalidSpecError indicates that application source is invalid
	ApplicationConditionInvalidSpecError = "InvalidSpecError"
	// ApplicationConditionComparisonError indicates controller failed to compare application state
	ApplicationConditionComparisonError = "ComparisonError"
	// ApplicationConditionSyncError indicates controller failed to automatically sync the application
	ApplicationConditionSyncError = "SyncError"
	// ApplicationConditionUnknownError indicates an unknown controller error
	ApplicationConditionUnknownError = "UnknownError"
	// ApplicationConditionSharedResourceWarning indicates that controller detected resources which belongs to more than one application
	ApplicationConditionSharedResourceWarning = "SharedResourceWarning"
	// ApplicationConditionRepeatedResourceWarning indicates that application source has resource with same Group, Kind, Name, Namespace multiple times
	ApplicationConditionRepeatedResourceWarning = "RepeatedResourceWarning"
	// ApplicationConditionExcludedResourceWarning indicates that application has resource which is configured to be excluded
	ApplicationConditionExcludedResourceWarning = "ExcludedResourceWarning"
	// ApplicationConditionOrphanedResourceWarning indicates that application has orphaned resources
	ApplicationConditionOrphanedResourceWarning = "OrphanedResourceWarning"
)

// ApplicationCondition contains details about an application condition, which is usually an error or warning
type ApplicationCondition struct {
	// Type is an application condition type
	Type ApplicationConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Message contains human-readable message indicating details about condition
	Message string `json:"message" protobuf:"bytes,2,opt,name=message"`
	// LastTransitionTime is the time the condition was last observed
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
}

// ComparedTo contains application source and target which was used for resources comparison
type ComparedTo struct {
	// Source is a reference to the application's source used for comparison
	Source ApplicationSource `json:"source,omitempty" protobuf:"bytes,1,opt,name=source"`
	// Destination is a reference to the application's destination used for comparison
	Destination ApplicationDestination `json:"destination" protobuf:"bytes,2,opt,name=destination"`
	// Sources is a reference to the application's multiple sources used for comparison
	Sources ApplicationSources `json:"sources,omitempty" protobuf:"bytes,3,opt,name=sources"`
	// IgnoreDifferences is a reference to the application's ignored differences used for comparison
	IgnoreDifferences IgnoreDifferences `json:"ignoreDifferences,omitempty" protobuf:"bytes,4,opt,name=ignoreDifferences"`
}

// SyncStatus contains information about the currently observed live and desired states of an application
type SyncStatus struct {
	// Status is the sync state of the comparison
	Status SyncStatusCode `json:"status" protobuf:"bytes,1,opt,name=status,casttype=SyncStatusCode"`
	// ComparedTo contains information about what has been compared
	ComparedTo ComparedTo `json:"comparedTo,omitempty" protobuf:"bytes,2,opt,name=comparedTo"`
	// Revision contains information about the revision the comparison has been performed to
	Revision string `json:"revision,omitempty" protobuf:"bytes,3,opt,name=revision"`
	// Revisions contains information about the revisions of multiple sources the comparison has been performed to
	Revisions []string `json:"revisions,omitempty" protobuf:"bytes,4,opt,name=revisions"`
}

// AppHealthStatus contains information about the currently observed health state of an application
type AppHealthStatus struct {
	// Status holds the status code of the application
	Status HealthStatusCode `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// Message is a human-readable informational message describing the health status
	//
	// Deprecated: this field is not used and will be removed in a future release.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// LastTransitionTime is the time the HealthStatus was set or updated
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
}

// HealthStatus contains information about the currently observed health state of a resource
type HealthStatus struct {
	// Status holds the status code of the resource
	Status HealthStatusCode `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// Message is a human-readable informational message describing the health status
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// LastTransitionTime is the time the HealthStatus was set or updated
	//
	// Deprecated: this field is not used and will be removed in a future release.
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
}

// ResourceStatus holds the current synchronization and health status of a Kubernetes resource.
type ResourceStatus struct {
	// Group represents the API group of the resource (e.g., "apps" for Deployments).
	Group string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	// Version indicates the API version of the resource (e.g., "v1", "v1beta1").
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	// Kind specifies the type of the resource (e.g., "Deployment", "Service").
	Kind string `json:"kind,omitempty" protobuf:"bytes,3,opt,name=kind"`
	// Namespace defines the Kubernetes namespace where the resource is located.
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	// Name is the unique name of the resource within the namespace.
	Name string `json:"name,omitempty" protobuf:"bytes,5,opt,name=name"`
	// Status represents the synchronization state of the resource (e.g., Synced, OutOfSync).
	Status SyncStatusCode `json:"status,omitempty" protobuf:"bytes,6,opt,name=status"`
	// Health indicates the health status of the resource (e.g., Healthy, Degraded, Progressing).
	Health *HealthStatus `json:"health,omitempty" protobuf:"bytes,7,opt,name=health"`
	// Hook is true if the resource is used as a lifecycle hook in an Argo CD application.
	Hook bool `json:"hook,omitempty" protobuf:"bytes,8,opt,name=hook"`
	// RequiresPruning is true if the resource needs to be pruned (deleted) as part of synchronization.
	RequiresPruning bool `json:"requiresPruning,omitempty" protobuf:"bytes,9,opt,name=requiresPruning"`
	// SyncWave determines the order in which resources are applied during a sync operation.
	SyncWave int64 `json:"syncWave,omitempty" protobuf:"bytes,10,opt,name=syncWave"`
	// RequiresDeletionConfirmation is true if the resource requires explicit user confirmation before deletion.
	RequiresDeletionConfirmation bool `json:"requiresDeletionConfirmation,omitempty" protobuf:"bytes,11,opt,name=requiresDeletionConfirmation"`
}
