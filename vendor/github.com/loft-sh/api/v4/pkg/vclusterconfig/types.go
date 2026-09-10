// Package vclusterconfig contains configuration types for vCluster Platform features.
// These types are used for parsing vCluster YAML configuration and are imported by vcluster.
package vclusterconfig

import (
	"encoding/json"
	"errors"
	"time"
)

type PlatformConfig struct {
	Sleep                    *Sleep                    `json:"sleep,omitempty"     yaml:"sleep,omitempty"`
	Snapshots                *Snapshots                `json:"snapshots,omitempty" yaml:"snapshots,omitempty"`
	Deletion                 *Deletion                 `json:"deletion,omitempty"  yaml:"deletion,omitempty"`
	Platform                 *Platform                 `json:"platform,omitempty"  yaml:"platform,omitempty"`
	NetrisIntegration        *NetrisIntegration        `json:"netris,omitempty"    yaml:"netris,omitempty"`
	ArgoCDIntegration        *ArgoCDIntegration        `json:"argoCD,omitempty"`
	ArgoCDDeploy             *ArgoCDDeploy             `json:"deploy,omitempty"`
	ObservabilityIntegration *ObservabilityIntegration `json:"observability,omitempty" yaml:"observability,omitempty"`
	Stacks                   []StackConfig             `json:"stacks,omitempty"`
}

// NewDefaultPlatformConfig returns an empty platform config.
// All fields are optional and default to nil/zero values.
func NewDefaultPlatformConfig() *PlatformConfig {
	return &PlatformConfig{}
}

type Image struct {
	// Registry is the registry of the container image, e.g. my-registry.com or ghcr.io. This setting can be globally
	// overridden via the controlPlane.advanced.defaultImageRegistry option. Empty means docker hub.
	Registry string `json:"registry,omitempty"`

	// Repository is the repository of the container image, e.g. my-repo/my-image
	Repository string `json:"repository,omitempty"`

	// Tag is the tag of the container image, and is the default version.
	Tag string `json:"tag,omitempty"`
}

// Sleep holds configuration for automatically putting the tenant cluster to sleep.
// This replaces sleepMode.
type Sleep struct {
	// Auto holds automatic sleep configuration
	Auto *SleepAuto `json:"auto,omitempty" yaml:"auto,omitempty"`
}

// SleepAuto holds configuration for automatic sleep and wakeup
type SleepAuto struct {
	// AfterInactivity represents how long a vCluster can be idle before workloads are automatically put to sleep
	AfterInactivity Duration `json:"afterInactivity,omitempty" yaml:"afterInactivity,omitempty"`

	// Schedule represents a cron schedule for when to sleep workloads
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Exclude holds configuration for labels that, if present, will prevent a workload from going to sleep
	Exclude SleepAutoExclusion `json:"exclude,omitempty"`

	// Wakeup holds configuration for waking the vCluster on a schedule
	Wakeup *SleepAutoWakeup `json:"wakeup,omitempty" yaml:"wakeup,omitempty"`

	// Timezone specifies time zone used for scheduled sleep operations. Defaults to UTC.
	// Accepts the same format as time.LoadLocation() in Go (https://pkg.go.dev/time#LoadLocation).
	// The value should be a location name corresponding to a file in the IANA Time Zone database, such as "America/New_York".
	// +optional
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

// Duration allows for automatic Marshalling from strings like "1m" to a time.Duration
type Duration string

// Parse the Duration to time.Duration
func (d Duration) Parse() (time.Duration, error) {
	return time.ParseDuration(string(d))
}

// MarshalJSON implements Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	dur, err := time.ParseDuration(string(d))
	if err != nil {
		return nil, err
	}
	return json.Marshal(dur.String())
}

// UnmarshalJSON implements Marshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	sval, ok := v.(string)
	if !ok {
		return errors.New("invalid duration")
	}

	// Support empty string value
	if sval == "" {
		sval = "0"
	}

	_, err := time.ParseDuration(sval)
	if err != nil {
		return err
	}
	*d = Duration(sval)
	return nil
}

// SleepAutoWakeup holds the cron schedule to wake workloads automatically
type SleepAutoWakeup struct {
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`
}

// SleepAutoExclusion holds conifiguration for excluding workloads from sleeping by label(s)
type SleepAutoExclusion struct {
	Selector LabelSelector `json:"selector,omitempty"`
}

type LabelSelector struct {
	// Labels defines what labels should be looked for
	Labels map[string]string `json:"labels,omitempty"`
}

// Snapshots holds configuration for automatic vCluster snapshots.
// This replaces external.platform.autoSnapshot.
type Snapshots struct {
	// Auto holds automatic snapshot configuration
	// +optional
	Auto *SnapshotsAuto `json:"auto,omitempty" yaml:"auto,omitempty"`
}

// SnapshotsAuto holds automatic snapshot scheduling and retention configuration
type SnapshotsAuto struct {
	// Schedule specifies a scheduled time in Cron format, see https://en.wikipedia.org/wiki/Cron for a tenant cluster snapshot to be taken
	// +optional
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Timezone specifies time zone used for scheduled snapshot operations. Defaults to UTC.
	// Accepts the same format as time.LoadLocation() in Go (https://pkg.go.dev/time#LoadLocation).
	// The value should be a location name corresponding to a file in the IANA Time Zone database, such as "America/New_York".
	// +optional
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// Retention specifies how long snapshots will be kept
	// +optional
	Retention *SnapshotRetention `json:"retention,omitempty" yaml:"retention,omitempty"`

	// Storage specifies where the snapshot will be stored
	// +optional
	Storage *SnapshotStorage `json:"storage,omitempty" yaml:"storage,omitempty"`
}

// SnapshotStorage holds snapshot storage configuration
type SnapshotStorage struct {
	// Type specifies supported type of storage services for a snapshot S3/OCI/Container, see https://www.vcluster.com/docs/vcluster/manage/backup-restore#store-snapshots-in-s3-buckets
	// +optional
	Type string `json:"type,omitempty"`

	// S3 holds configuration for storing snapshots in S3-compatible bucket
	// +optional
	S3 SnapshotStorageS3 `json:"s3,omitempty"`

	// OCI holds configuration for storing snapshots in OCI image registries
	// +optional
	OCI SnapshotStorageOCI `json:"oci,omitempty"`

	// Container holds configuration for storing snapshots as local files inside a vCluster container
	// +optional
	Container SnapshotStorageContainer `json:"container,omitempty"`

	// Azure holds configuration for storing snapshots in Azure Blob Storage
	// +optional
	Azure SnapshotStorageAzure `json:"azure,omitempty"`
}

// SnapshotStorageS3 holds S3 storage configuration
type SnapshotStorageS3 struct {
	// Url specifies url to the storage service
	// +optional
	Url string `json:"url,omitempty"`

	// Credential secret with the S3 Credentials, it should contain AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN
	// +optional
	Credential *SnapshotSecretCredential `json:"credential,omitempty"`
}

// SnapshotStorageOCI holds OCI registry storage configuration
type SnapshotStorageOCI struct {
	// Repository OCI repository to store the snapshot
	// +optional
	Repository string `json:"repository,omitempty"`

	// Credential secret with the OCI Credentials
	// +optional
	Credential *SnapshotSecretCredential `json:"credential,omitempty"`

	// Username to authenticate with the OCI registry
	// +optional
	Username string `json:"username,omitempty"`

	// Password to authenticate with the OCI registry
	// +optional
	Password string `json:"password,omitempty"`
}

// SnapshotStorageContainer holds container local storage configuration
type SnapshotStorageContainer struct {
	// Path specifies directory to store the snapshot
	// +optional
	Path string `json:"path,omitempty"`

	// Volume specifies which volume needs to be mounted into the container to store the snapshot
	// +optional
	Volume SnapshotStorageContainerVolume `json:"volume,omitempty"`
}

// SnapshotStorageAzure holds Azure Blob Storage configuration.
type SnapshotStorageAzure struct {
	// BlobURL specifies the Azure Blob Storage URL in the format https://{account}.blob.core.windows.net/{container}/{path}
	// +optional
	BlobURL string `json:"blobUrl,omitempty"`

	// Credential secret with the Azure credentials. The secret should contain either:
	// AZURE_STORAGE_KEY (storage account access key), or
	// AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_SUBSCRIPTION_ID, AZURE_RESOURCE_GROUP (service principal)
	// +optional
	Credential *SnapshotSecretCredential `json:"credential,omitempty"`
}

// SnapshotStorageContainerVolume holds volume mount configuration
type SnapshotStorageContainerVolume struct {
	// Name to be used to mount the volume
	// +optional
	Name string `json:"name,omitempty"`

	// Path to the volume mount
	// +optional
	Path string `json:"path,omitempty"`
}

// SnapshotRetention holds snapshot retention configuration
type SnapshotRetention struct {
	// Period defines the number of days a snapshot will be kept
	// +optional
	Period int `json:"period,omitempty"`

	// MaxSnapshots defines the number of snapshots that can be taken
	// +optional
	MaxSnapshots int `json:"maxSnapshots,omitempty"`
}

// SnapshotSecretCredential holds secret reference for credentials
type SnapshotSecretCredential struct {
	// SecretName is the secret name with credential
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretNamespace is the secret namespace with credential
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`
}

// Deletion holds configuration for automatic vCluster deletion.
// This replaces external.platform.autoDelete.
type Deletion struct {
	// Prevent prevents the vCluster from being deleted
	// +optional
	Prevent bool `json:"prevent,omitempty" yaml:"prevent,omitempty"`

	// Auto holds automatic deletion configuration
	// +optional
	Auto *DeletionAuto `json:"auto,omitempty" yaml:"auto,omitempty"`
}

// DeletionAuto holds automatic deletion configuration
type DeletionAuto struct {
	// AfterInactivity specifies after how long of inactivity the tenant cluster will be deleted.
	// Uses Go duration format (e.g., "720h" for 30 days).
	// +optional
	AfterInactivity Duration `json:"afterInactivity,omitempty" yaml:"afterInactivity,omitempty"`
}

// Platform holds vCluster Platform specific configuration.
// This replaces the remaining fields from external.platform.
type Platform struct {
	// APIKey defines where to find the platform access key and host. By default, vCluster will search in the following locations in this precedence:
	// * environment variable called LICENSE
	// * secret specified under platform.apiKey.secretName
	// * secret called "vcluster-platform-api-key" in the vCluster namespace
	APIKey PlatformAPIKey `json:"apiKey,omitempty"`

	// Project specifies which platform project the vcluster should be imported to
	// +optional
	Project string `json:"project,omitempty"`
}

// PlatformAPIKey defines where to find the platform access key. The secret key name doesn't matter as long as the secret only contains a single key.
type PlatformAPIKey struct {
	// SecretName is the name of the secret where the platform access key is stored. This defaults to vcluster-platform-api-key if undefined.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Namespace defines the namespace where the access key secret should be retrieved from. If this is not equal to the namespace
	// where the vCluster instance is deployed, you need to make sure vCluster has access to this other namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// CreateRBAC will automatically create the necessary RBAC roles and role bindings to allow vCluster to read the secret specified
	// in the above namespace, if specified.
	// This defaults to true.
	// +optional
	CreateRBAC *bool `json:"createRBAC,omitempty"`
}

// NetrisIntegration holds netris integration configuration.
// This provides type safety for the previously untyped integrations.netris field.
type NetrisIntegration struct {
	// Enabled defines if netris integration is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Connector specifies the netris connector name
	// +optional
	Connector string `json:"connector,omitempty"`

	// KubeVip holds kube-vip configuration for netris
	// +optional
	KubeVip NetrisKubeVipConfig `json:"kubeVip,omitempty"`
}

// NetrisKubeVipConfig holds kube-vip configuration for netris integration
type NetrisKubeVipConfig struct {
	// ServerCluster specifies the server cluster name
	// +optional
	ServerCluster string `json:"serverCluster,omitempty"`

	// Bridge specifies the bridge interface name
	// +optional
	Bridge string `json:"bridge,omitempty"`

	// IPRange specifies the IP range for kube-vip
	// +optional
	IPRange string `json:"ipRange,omitempty"`
}

// ArgoCDIntegration holds argo cd integration configuration.
type ArgoCDIntegration struct {
	// Enabled defines if argo cd integration is enabled
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Connector specifies the argo cd connector name
	// +optional
	Connector string `json:"connector,omitempty"`
}

// ArgoCDDeploy holds argo cd deploy configuration.
type ArgoCDDeploy struct {
	// Applications specifies the applications to deploy. This requires the argo cd integration to be enabled.
	// +optional
	Applications []ArgoCDApplication `json:"applications,omitempty"`
}

// StackConfig holds a single stack declaration in vcluster.yaml (deploy.stacks). Exactly one
// of Template or TemplateRef must be set: the stack definition inline, or a reference to a
// cluster-scoped StackTemplate. The shape mirrors the StackInstance spec.
type StackConfig struct {
	// Name specifies the stable identifier of the stack.
	Name string `json:"name"`

	// DisplayName specifies the display name of the stack.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes the stack.
	// +optional
	Description string `json:"description,omitempty"`

	// Template defines the stack inline: parameter declarations and tasks, the same payload
	// a StackTemplate carries. Mutually exclusive with TemplateRef.
	// +optional
	Template *StackTemplateDefinitionConfig `json:"template,omitempty"`

	// TemplateRef references a cluster-scoped StackTemplate to resolve the tasks from.
	// Mutually exclusive with Template.
	// +optional
	TemplateRef *StackTemplateRefConfig `json:"templateRef,omitempty"`

	// Parameters specifies the values the template's tasks reference. Values the template does not
	// declare are still available to tasks, but get no default and are not validated.
	// +optional
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Defaults are applied to all tasks unless overridden per-task.
	// +optional
	Defaults *StackDefaultsConfig `json:"defaults,omitempty"`

	// PrunePolicy controls what happens to an application whose task is removed from the
	// resolved set. One of "Retain" (default) or "Prune".
	// +optional
	PrunePolicy string `json:"prunePolicy,omitempty"`
}

// StackTaskConfig holds one task in an inline stack. Each task carries exactly one typed
// payload: argoCDApplication or app.
type StackTaskConfig struct {
	// Name specifies the stable identifier of the task. DNS-label-safe, unique within the stack.
	Name string `json:"name"`

	// DependsOn lists task names that must be healthy before this task starts.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// ArgoCDApplication specifies the argo cd application this task deploys. Exactly one of
	// ArgoCDApplication or App must be set.
	// +optional
	ArgoCDApplication *ArgoCDApplicationTaskConfig `json:"argoCDApplication,omitempty"`

	// App specifies the platform app this task deploys. Exactly one of ArgoCDApplication or
	// App must be set.
	// +optional
	App *AppTaskConfig `json:"app,omitempty"`

	// Timeout bounds how long this task may take to become healthy, overriding
	// defaults.taskTimeout. Go duration string, for example "10m".
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Outputs declares values captured from this task's deployed resources that later
	// tasks reference as {{ .Outputs.task.name }}.
	// +optional
	Outputs []StackTaskOutputConfig `json:"outputs,omitempty"`
}

// StackTaskOutputConfig declares one task output.
type StackTaskOutputConfig struct {
	// Name identifies the output. Letters and digits only, unique within the task.
	Name string `json:"name"`

	// FromSecret reads the value from a Secret key on the destination cluster.
	// Exactly one of fromSecret or fromResource must be set.
	// +optional
	FromSecret *SecretOutputSourceConfig `json:"fromSecret,omitempty"`

	// FromResource reads a single field from a resource on the destination cluster.
	// Exactly one of fromSecret or fromResource must be set.
	// +optional
	FromResource *ResourceOutputSourceConfig `json:"fromResource,omitempty"`
}

// SecretOutputSourceConfig names one Secret key on the destination cluster.
type SecretOutputSourceConfig struct {
	// Namespace of the Secret. Must be a namespace this stack deploys into.
	Namespace string `json:"namespace"`

	// Name of the Secret.
	Name string `json:"name"`

	// Key inside the Secret's data.
	Key string `json:"key"`
}

// ResourceOutputSourceConfig names one field of one resource on the destination cluster.
type ResourceOutputSourceConfig struct {
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

// ArgoCDApplicationTaskConfig is the argo cd application payload of a stack task. Exactly
// one of Template or TemplateRef must be set.
type ArgoCDApplicationTaskConfig struct {
	// Template specifies the inline argo cd application template definition. Mutually
	// exclusive with TemplateRef.
	// +optional
	Template map[string]interface{} `json:"template,omitempty"`

	// TemplateRef references an ArgoCDApplicationTemplate (per-application blueprint) by
	// name. Mutually exclusive with Template.
	// +optional
	TemplateRef *ArgoCDApplicationTemplateRefConfig `json:"templateRef,omitempty"`

	// Parameters supplies values to the referenced template's declared parameters. Only
	// valid with TemplateRef; values for an inline template belong in the definition.
	// +optional
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ArgoCDApplicationTemplateRefConfig references a cluster-scoped ArgoCDApplicationTemplate.
type ArgoCDApplicationTemplateRefConfig struct {
	// Name of the ArgoCDApplicationTemplate.
	// +optional
	Name string `json:"name,omitempty"`
}

// AppTaskConfig is the app payload of a stack task. Exactly one of Template or TemplateRef
// must be set.
type AppTaskConfig struct {
	// Template specifies the inline app definition, with metadata and a spec that carries the
	// chart plus a few instance fields (displayName, description, releaseName). Mutually
	// exclusive with TemplateRef.
	// +optional
	Template map[string]interface{} `json:"template,omitempty"`

	// TemplateRef references a named app by name and version. Mutually exclusive with Template.
	// +optional
	TemplateRef *AppInstanceTemplateRefConfig `json:"templateRef,omitempty"`

	// Parameters supplies values to the referenced app. Only valid with TemplateRef; values
	// for an inline template belong in the definition.
	// +optional
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// AppInstanceTemplateRefConfig references a named app.
type AppInstanceTemplateRefConfig struct {
	// Name of the app to reference.
	// +optional
	Name string `json:"name,omitempty"`

	// Version of the app to deploy. If empty, the latest version is deployed.
	// +optional
	Version string `json:"version,omitempty"`
}

// StackTemplateDefinitionConfig is an inline stack definition: the same payload a
// cluster-scoped StackTemplate carries, declared directly in vcluster.yaml.
type StackTemplateDefinitionConfig struct {
	// Parameters declares the parameters the tasks may reference as {{ .Values.variable }}.
	// Values are supplied by the stack's parameters field.
	// +optional
	Parameters []StackParameterConfig `json:"parameters,omitempty"`

	// Tasks defines the stack DAG.
	// +optional
	Tasks []StackTaskConfig `json:"tasks,omitempty"`

	// PublishedOutputs selects which captured task outputs the stack exposes to its
	// users, under a public name. They are read through the stack instance's outputs
	// subresource.
	// +optional
	PublishedOutputs []StackPublishedOutputConfig `json:"publishedOutputs,omitempty"`
}

// StackPublishedOutputConfig exposes one captured task output under a public name.
type StackPublishedOutputConfig struct {
	// Name specifies the public name of the output. Unique within the stack; it may
	// differ from the task output's name.
	Name string `json:"name"`

	// FromTask selects which task output to publish. It must name an existing task
	// and one of its declared outputs.
	FromTask StackPublishedOutputFromTaskConfig `json:"fromTask"`
}

// StackPublishedOutputFromTaskConfig names one declared output of one task.
type StackPublishedOutputFromTaskConfig struct {
	// Task specifies the task name.
	Task string `json:"task"`

	// Output specifies the output name declared on that task.
	Output string `json:"output"`
}

// StackTemplateRefConfig references a cluster-scoped StackTemplate.
type StackTemplateRefConfig struct {
	// Name specifies the name of the StackTemplate to reference.
	Name string `json:"name"`
}

// StackParameterConfig declares one parameter of an inline stack definition. It mirrors the
// platform's app parameter fields.
type StackParameterConfig struct {
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

// StackDefaultsConfig holds defaults applied to all tasks of a stack.
type StackDefaultsConfig struct {
	// TaskTimeout is the default per-task timeout as a Go duration (e.g. "10m").
	// +optional
	TaskTimeout string `json:"taskTimeout,omitempty"`
}

// ArgoCDApplication holds argo cd application configuration.
type ArgoCDApplication struct {
	// Name specifies the stable identifier of the argo cd application. It is used to derive generated
	// ArgoCDApplication resource names and the final Argo CD application name.
	// +optional
	Name string `json:"name,omitempty"`

	// DisplayName specifies the display name of the argo cd application.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Target specifies the target of the argo cd application. This can be "vCluster" or "host". Defaults to "vCluster".
	// +optional
	Target string `json:"target,omitempty"`

	// Inline specifies the inline argo cd application definition. This requires the argo cd integration to be enabled.
	// +optional
	Inline map[string]interface{} `json:"inline,omitempty"`

	// Template specifies the argo cd application template to use. This requires the argo cd integration to be enabled.
	// +optional
	Template *ArgoCDApplicationTemplate `json:"template,omitempty"`
}

type ArgoCDApplicationTemplate struct {
	// Name specifies the name of the argo cd application template
	// +optional
	Name string `json:"name,omitempty"`

	// Parameters specifies the parameters to pass to the argo cd application template.
	// +optional
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ObservabilityIntegration holds observability integration configuration.
// The integration is considered configured when both Enabled is true and Connector is set.
type ObservabilityIntegration struct {
	// Enabled defines if the observability integration is enabled.
	// Connector must also be set for the integration to be considered configured.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Connector specifies the Platform observability connector to connect to.
	// Required when Enabled is true for the integration to be considered configured.
	// +optional
	Connector string `json:"connector,omitempty"`

	// GatewaySecret overrides where the metrics-writer Secret is delivered.
	// When unset, the shared default target is used: namespace "observability", name "metrics-writer".
	// +optional
	GatewaySecret *GatewaySecret `json:"gatewaySecret,omitempty"`
}

// GatewaySecret identifies a Secret delivery target by namespace and name.
type GatewaySecret struct {
	// Namespace is the namespace the Secret is delivered to.
	// Required once a gatewaySecrets entry is present.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Name is the name of the delivered Secret.
	// Required once a gatewaySecrets entry is present.
	// +optional
	Name string `json:"name,omitempty"`
}

const (
	// DefaultObservabilityNamespace is the default namespace the metrics-writer Secret is delivered to.
	DefaultObservabilityNamespace = "observability"
	// DefaultMetricsWriterSecretName is the default name of the delivered metrics-writer Secret.
	DefaultMetricsWriterSecretName = "metrics-writer"
)

// DeliveryTarget returns where the metrics-writer Secret should be delivered.
// It returns nil when the integration is nil or disabled, so "off" stays
// distinguishable from "default target". For an enabled integration it returns the
// configured GatewaySecret, or the default namespace/name when none is set.
func (o *ObservabilityIntegration) DeliveryTarget() *GatewaySecret {
	if o == nil || !o.Enabled {
		return nil
	}
	if o.GatewaySecret != nil {
		return o.GatewaySecret
	}
	return &GatewaySecret{Namespace: DefaultObservabilityNamespace, Name: DefaultMetricsWriterSecretName}
}
