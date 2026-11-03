package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// StackInstanceConditions are the conditions the StackInstance controller owns on a StackInstance.
	StackInstanceConditions = []agentstoragev1.ConditionType{
		StackInstanceReady,
	}
)

const (
	// StackInstanceReady is the top-level readiness condition of a StackInstance.
	StackInstanceReady agentstoragev1.ConditionType = "Ready"

	// StackInstanceReasonInvalid is set on the Ready condition when the spec itself is malformed,
	// so no task can run until a person edits it.
	StackInstanceReasonInvalid = "Invalid"
	// StackInstanceReasonTemplateNotFound is set on the Ready condition when the StackTemplate
	// referenced by spec.templateRef does not exist.
	StackInstanceReasonTemplateNotFound = "TemplateNotFound"
	// StackInstanceReasonTemplateInvalid is set on the Ready condition when the referenced
	// StackTemplate exists but cannot be rendered (missing parameter, failed validation, or a
	// substitution error). This is a permanent user error, so the controller reports it and
	// waits rather than retrying fast.
	StackInstanceReasonTemplateInvalid = "TemplateInvalid"
	// StackInstanceReasonProgressing is set on the Ready condition while tasks are still rolling out.
	StackInstanceReasonProgressing = "Progressing"
	// StackInstanceReasonDegraded is set on the Ready condition when at least one task failed.
	StackInstanceReasonDegraded = "Degraded"
	// StackInstanceReasonNoTasks is set on the Ready condition when the StackInstance resolved to
	// no tasks at all, so there is nothing to run. A StackInstance that is Pending because a task
	// is blocked reports that task's own reason instead.
	StackInstanceReasonNoTasks = "NoTasks"
	// StackInstanceReasonFeatureNotAllowed is set on the Ready condition when the license does not
	// cover a feature the StackInstance's tasks need. Nothing runs until the license changes.
	StackInstanceReasonFeatureNotAllowed = "FeatureNotAllowed"
	// StackInstanceReasonOutputsConflict is set on the Ready condition when a Secret the
	// StackInstance does not own holds the name its task outputs are stored under. Nothing runs
	// until that Secret is gone, since retrying cannot free a name somebody else holds.
	StackInstanceReasonOutputsConflict = "OutputsConflict"
	// StackInstanceReasonOutputsRejected is set on the Ready condition when the apiserver refuses
	// the Secret the StackInstance stores its task outputs in.
	StackInstanceReasonOutputsRejected = "OutputsRejected"
	// StackInstanceReasonDeleting is set on the Ready condition while the StackInstance is being
	// torn down, so the condition matches the Deleting phase.
	StackInstanceReasonDeleting = "Deleting"
	// StackInstanceReasonDestinationNotFound is set on the Ready condition when spec.destination
	// names a cluster or tenant cluster that does not exist. Destinations are immutable, so this
	// clears only once the destination is created.
	StackInstanceReasonDestinationNotFound = "DestinationNotFound"
	// StackInstanceReasonDestinationDeleting is set on the Ready condition when spec.destination
	// is being torn down. Applications are not applied while it is, because a recreated
	// application would hold up the destination's own deletion.
	StackInstanceReasonDestinationDeleting = "DestinationDeleting"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackInstance orchestrates a dependency-ordered bundle of applications for a single destination.
// +k8s:openapi-gen=true
type StackInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackInstanceSpec   `json:"spec,omitempty"`
	Status StackInstanceStatus `json:"status,omitempty"`
}

func (a *StackInstance) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *StackInstance) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *StackInstance) GetAccess() []Access {
	return a.Spec.Access
}

func (a *StackInstance) SetAccess(access []Access) {
	a.Spec.Access = access
}

func (a *StackInstance) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *StackInstance) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

type StackInstanceSpec struct {
	// DisplayName is the name that should be displayed in the UI
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the stack instance
	// +optional
	Description string `json:"description,omitempty"`

	// Destination is where every materialized application is deployed: a tenant cluster or a
	// control plane cluster. Immutable after creation.
	Destination StackDestination `json:"destination,omitempty"`

	// Template defines the stack inline. It has the same shape as a StackTemplate's
	// parameters and tasks, so a template's payload can be copy-pasted here and back.
	// Exactly one of Template or TemplateRef must be set (validated at admission).
	// +optional
	Template *StackTemplateDefinition `json:"template,omitempty"`

	// TemplateRef references a cluster-scoped StackTemplate, resolved live each reconcile.
	// Exactly one of Template or TemplateRef must be set (validated at admission).
	// +optional
	TemplateRef *StackTemplateRef `json:"templateRef,omitempty"`

	// Parameters are the values the template's tasks reference as .Values.<name>, the same way
	// an app reads its parameters. Declarations provide defaults and validation but are not a
	// whitelist: values without a matching declaration are passed through and may be
	// referenced too.
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// Defaults are applied to all tasks unless overridden per-task.
	// +optional
	Defaults *StackDefaults `json:"defaults,omitempty"`

	// PrunePolicy controls what happens to an owned application whose task is removed
	// from the resolved task set. Leaving it out is treated as Retain.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == 'Retain' || self == 'Prune'",message="prunePolicy must be Retain or Prune"
	PrunePolicy StackPrunePolicy `json:"prunePolicy,omitempty"`

	// Owner holds the owner of this object
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams
	// +optional
	Access []Access `json:"access,omitempty"`
}

// StackPrunePolicy controls orphaned-application handling on template drift.
type StackPrunePolicy string

const (
	// StackPrunePolicyRetain keeps an orphaned application (default, also the zero value).
	StackPrunePolicyRetain StackPrunePolicy = "Retain"
	// StackPrunePolicyPrune deletes an orphaned application in reverse-topological order.
	StackPrunePolicyPrune StackPrunePolicy = "Prune"
)

type StackDefaults struct {
	// TaskTimeout is the default per-task timeout. The empty value is treated as 10m.
	// +optional
	TaskTimeout *metav1.Duration `json:"taskTimeout,omitempty"`
}

type StackTask struct {
	// Name is the stable identifier of the task. DNS-label-safe, unique within the stack.
	// A task that declares outputs may use letters and digits only: its outputs are
	// referenced as {{ .Outputs.task.name }}, and the template syntax cannot
	// address a name containing "-".
	Name string `json:"name"`

	// DependsOn lists task names that must reach readiness before this task starts.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// ArgoCDApplication defines this task as an Argo CD application. Exactly one of
	// ArgoCDApplication or App must be set per task (validated at admission).
	// +optional
	ArgoCDApplication *StackArgoCDApplicationTask `json:"argoCDApplication,omitempty"`

	// App defines this task as a platform App. Exactly one of ArgoCDApplication or App
	// must be set per task (validated at admission).
	// +optional
	App *StackAppTask `json:"app,omitempty"`

	// Timeout overrides Defaults.TaskTimeout for this task.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Outputs declares named values this task publishes once it is Healthy. Later tasks
	// consume them in their specs as {{ .Outputs.task.name }} and must list
	// this task in dependsOn (validated at admission).
	// +optional
	Outputs []StackTaskOutput `json:"outputs,omitempty"`
}

// StackTaskOutput declares one captured value and where the controller reads it from.
type StackTaskOutput struct {
	// Name identifies the output. Letters and digits only, unique within the task: the
	// output is referenced as {{ .Outputs.task.name }}, and the template syntax
	// cannot address a name containing "-".
	Name string `json:"name"`

	// FromSecret reads the value from a Secret key on the destination cluster. Exactly
	// one of fromSecret or fromResource must be set per output.
	// +optional
	FromSecret *StackSecretOutputSource `json:"fromSecret,omitempty"`

	// FromResource reads a single field from a resource on the destination cluster.
	// Exactly one of fromSecret or fromResource must be set per output.
	// +optional
	FromResource *StackResourceOutputSource `json:"fromResource,omitempty"`
}

// StackSecretOutputSource names one Secret key on the destination cluster.
type StackSecretOutputSource struct {
	// Namespace of the Secret. Must be a namespace this stack deploys into.
	Namespace string `json:"namespace"`

	// Name of the Secret.
	Name string `json:"name"`

	// Key inside the Secret's data.
	Key string `json:"key"`
}

// StackResourceOutputSource names one field of one resource on the destination cluster.
type StackResourceOutputSource struct {
	// APIVersion of the resource, e.g. "v1" or "apps/v1".
	APIVersion string `json:"apiVersion"`

	// Kind of the resource, e.g. "Service".
	Kind string `json:"kind"`

	// Namespace of the resource. Must be a namespace this stack deploys into;
	// cluster-scoped resources cannot be read.
	Namespace string `json:"namespace"`

	// Name of the resource.
	Name string `json:"name"`

	// JSONPath selects the field, in the kubectl template syntax, e.g.
	// "{.spec.clusterIP}". It must select exactly one scalar value.
	JSONPath string `json:"jsonPath"`
}

// StackArgoCDApplicationTask is the Argo CD payload of a StackTask.
type StackArgoCDApplicationTask struct {
	// Template is an inline ArgoCD application template definition. Exactly one of
	// Template or TemplateRef must be set per task (validated at admission).
	// +optional
	Template *ArgoCDApplicationTemplateDefinition `json:"template,omitempty"`

	// TemplateRef references an ArgoCDApplicationTemplate (per-application blueprint).
	// Exactly one of Template or TemplateRef must be set per task (validated at admission).
	// +optional
	TemplateRef *ArgoCDApplicationTemplateRef `json:"templateRef,omitempty"`

	// Parameters are rendered as part of the task, then passed to the referenced
	// ArgoCDApplicationTemplate. Only valid with TemplateRef; admission rejects the field
	// on a task with an inline Template, whose values belong in the definition itself.
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// StackAppTask is the AppInstance payload of a StackTask.
type StackAppTask struct {
	// Template is an inline app definition. Exactly one of Template or TemplateRef
	// must be set per task (validated at admission).
	// +optional
	Template *StackAppTemplate `json:"template,omitempty"`

	// TemplateRef references a named app by name and version. Exactly one of Template or
	// TemplateRef must be set per task (validated at admission).
	// +optional
	TemplateRef *AppInstanceTemplateRef `json:"templateRef,omitempty"`

	// Parameters are passed to the referenced app. Only valid with TemplateRef; admission
	// rejects the field on a task with an inline Template, whose values belong in the
	// definition itself.
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// StackAppTemplate is the inline definition of the app an app task deploys. Like the other task
// templates it is a metadata/spec pair, but its spec is a small StackAppSpec: a chart plus a few
// instance fields. Where the app runs, as whom, and who can see it come from the stack instead.
type StackAppTemplate struct {
	// Metadata holds the labels and annotations for the child app instance.
	// +optional
	TemplateMetadata `json:"metadata,omitempty"`

	// Spec is the app definition of the child app instance.
	// +optional
	Spec StackAppSpec `json:"spec,omitempty"`
}

// StackAppSpec is the part of an app instance an app task owns: a few instance fields plus an
// inlined AppConfig (chart, values, deploy options). Destination, owner, and access come from the
// stack, and the referenced-app fields live on the AppTask templateRef arm, so none appear here.
type StackAppSpec struct {
	// DisplayName is the name shown for the app instance in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the app instance.
	// +optional
	Description string `json:"description,omitempty"`

	// ReleaseName is the helm release name for the app instance. If empty, it defaults to the
	// generated child name.
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// AppConfig is the inline chart, values, and deploy options.
	AppConfig `json:",inline"`
}

// StackTemplateDefinition is the reusable payload of a StackTemplate: the declared
// parameters and the task DAG. It is embedded inline in StackTemplateSpec and used
// verbatim as a StackInstance's inline spec.template.
type StackTemplateDefinition struct {
	// Parameters declares the typed parameters (with defaults) a stack accepts, reused from AppParameter.
	// +optional
	Parameters []AppParameter `json:"parameters,omitempty"`

	// Tasks is the DAG blueprint. Each task follows the same task-level
	// mutual-exclusivity rule (ArgoCDApplication XOR App), validated at admission.
	// +optional
	Tasks []StackTask `json:"tasks,omitempty"`

	// PublishedOutputs selects which captured task outputs the stack exposes to its
	// users, optionally under a different name. The values come from the captured task
	// outputs; nothing extra is read from the cluster. They are read through the
	// instance's outputs subresource, never from its status.
	// +optional
	PublishedOutputs []StackPublishedOutput `json:"publishedOutputs,omitempty"`
}

// StackPublishedOutput exposes one captured task output under a public name.
type StackPublishedOutput struct {
	// Name is the public name of the output. Unique within the stack; renaming the
	// task output is allowed and expected.
	Name string `json:"name"`

	// FromTask selects which captured task output to publish. It must name an
	// existing task and one of its declared outputs (validated at admission).
	FromTask StackPublishedOutputFromTask `json:"fromTask"`
}

// StackPublishedOutputFromTask names one declared output of one task.
type StackPublishedOutputFromTask struct {
	// Task is the task name.
	Task string `json:"task"`

	// Output is the output name declared on that task.
	Output string `json:"output"`
}

// StackTemplateRef references a cluster-scoped StackTemplate. The values for the
// template's declared parameters live on the instance at spec.parameters.
type StackTemplateRef struct {
	// Name holds the name of the StackTemplate to reference.
	Name string `json:"name"`
}

type StackInstanceStatus struct {
	// Phase is the aggregate phase of the StackInstance. Consumers must tolerate unknown values.
	// +optional
	Phase StackInstancePhase `json:"phase,omitempty"`

	// Conditions holds several conditions the StackInstance might be in.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// Tasks is the denormalized per-task status, including not-yet-materialized tasks. It is empty
	// when the instance stopped before it could work out its task set, such as a missing
	// destination or a template that does not resolve.
	// +optional
	Tasks []StackTaskStatus `json:"tasks,omitempty"`

	// OrphanedApplications lists owned children whose task is no longer in the
	// resolved task set (template drift). Reported, not deleted, unless prunePolicy is Prune.
	// +optional
	OrphanedApplications []StackOrphanedApplication `json:"orphanedApplications,omitempty"`

	// ObservedGeneration is the generation the controller last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// StackInstancePhase is the aggregate phase of a StackInstance.
type StackInstancePhase string

const (
	StackInstancePhasePending     StackInstancePhase = "Pending"     // a prerequisite is missing and needs manual intervention
	StackInstancePhaseProgressing StackInstancePhase = "Progressing" // >=1 task still rolling out
	StackInstancePhaseHealthy     StackInstancePhase = "Healthy"     // every task reached readiness
	StackInstancePhaseDegraded    StackInstancePhase = "Degraded"    // >=1 task Failed
	StackInstancePhaseDeleting    StackInstancePhase = "Deleting"
)

// StackTaskPhase is the phase of a single task.
type StackTaskPhase string

const (
	StackTaskPhasePending     StackTaskPhase = "Pending"     // not yet materialized
	StackTaskPhaseWaiting     StackTaskPhase = "Waiting"     // waiting on dependencies
	StackTaskPhaseProgressing StackTaskPhase = "Progressing" // child rolling out
	StackTaskPhaseBlocked     StackTaskPhase = "Blocked"     // waiting on a prerequisite someone has to fix
	StackTaskPhaseHealthy     StackTaskPhase = "Healthy"     // child reached readiness
	StackTaskPhaseFailed      StackTaskPhase = "Failed"      // child failed or timed out
)

// StackTaskType is the kind of child a task materializes, detected from which of the
// mutually exclusive task payloads is set (argoCDApplication or app). It lets a status
// consumer (the UI badge) know a task's type without reading the spec, which a templateRef
// stack does not expose inline.
type StackTaskType string

const (
	// StackTaskTypeArgoCDApplication is a task whose child is an ArgoCDApplication.
	StackTaskTypeArgoCDApplication StackTaskType = "argoCDApplication"
	// StackTaskTypeApp is a task whose child is an App.
	StackTaskTypeApp StackTaskType = "app"
)

type StackTaskStatus struct {
	// Name is the task name.
	Name string `json:"name"`

	// Phase is the per-task phase. It is type-agnostic: Healthy/Failed report the task's
	// resolved readiness, not an ArgoCD-specific state.
	Phase StackTaskPhase `json:"phase"`

	// Type is the task's child kind (argoCDApplication or app). Empty on status objects
	// written before this field existed.
	// +optional
	Type StackTaskType `json:"type,omitempty"`

	// DependsOn are the resolved dependency edges for this task (inline AND templateRef),
	// so the UI can render the graph from this object alone.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// ApplicationName is the materialized child object name (empty until created).
	// +optional
	ApplicationName string `json:"applicationName,omitempty"`

	// Synced mirrors the child ArgoCD sync state. Set only for argoCDApplication tasks;
	// empty for other task types.
	// +optional
	Synced bool `json:"synced,omitempty"`

	// Health mirrors the child ArgoCD health. Set only for argoCDApplication tasks;
	// empty for other task types.
	// +optional
	Health string `json:"health,omitempty"`

	// Reason describes the reason in machine-readable form why the task is in the current phase.
	// Empty while the task is healthy.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message describes the reason in human-readable form why the task is in the current phase.
	// +optional
	Message string `json:"message,omitempty"`

	// LastTransitionTime is when this task last changed phase.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// StackOrphanedApplication is an owned child whose task is no longer in the resolved set.
type StackOrphanedApplication struct {
	// ApplicationName is the child's Kubernetes object name (what Prune acts on). The name
	// is retained for traceability; with more than one task type it may name a non-Application
	// child, so read Type to know the kind.
	ApplicationName string `json:"applicationName"`

	// Type is the orphan's child kind (argoCDApplication or app), stamped from the adapter
	// that listed it. Empty on status objects written before this field existed.
	// +optional
	Type StackTaskType `json:"type,omitempty"`

	// TaskName is the name of the task this child was created for. That task is no longer in the
	// resolved task set.
	// +optional
	TaskName string `json:"taskName,omitempty"`

	// DependsOn are the orphan's resolved dependency edges, captured when the task left the
	// resolved set and carried forward here. They give Prune the reverse-dependency order so
	// a child something still depends on is never deleted first.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackInstanceList contains a list of StackInstances
type StackInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StackInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StackInstance{}, &StackInstanceList{})
}
