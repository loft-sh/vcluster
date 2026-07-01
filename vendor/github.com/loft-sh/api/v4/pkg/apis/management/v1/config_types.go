package v1

import (
	auditv1 "github.com/loft-sh/api/v4/pkg/apis/audit/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	uiv1 "github.com/loft-sh/api/v4/pkg/apis/ui/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config holds the loft configuration
// +k8s:openapi-gen=true
// +resource:path=configs,rest=ConfigREST
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// ConfigSpec holds the specification
type ConfigSpec struct {
	// Raw holds the raw config
	// +optional
	Raw []byte `json:"raw,omitempty"`
}

// ConfigStatus holds the status, which is the parsed raw config
type ConfigStatus struct {
	// Authentication holds the information for authentication
	// +optional
	Authentication storagev1.Authentication `json:"auth,omitempty"`

	// DEPRECATED: Configure the OIDC clients using either the OIDC Client UI or a secret. By default, vCluster Platform as an OIDC Provider is enabled but does not function without OIDC clients.
	// +optional
	OIDC *OIDC `json:"oidc,omitempty"`

	// Apps holds configuration around apps
	// +optional
	Apps *Apps `json:"apps,omitempty"`

	// Audit holds audit configuration
	// +optional
	Audit *Audit `json:"audit,omitempty"`

	// LoftHost holds the domain where the loft instance is hosted. This should not include https or http. E.g. loft.my-domain.com
	// +optional
	LoftHost string `json:"loftHost,omitempty"`

	// ProjectNamespacePrefix holds the prefix for loft project namespaces. Omitted defaults to "p-"
	// +optional
	ProjectNamespacePrefix *string `json:"projectNamespacePrefix,omitempty"`

	// DEPRECATED: DevPodSubDomain holds a subdomain in the following form *.workspace.my-domain.com
	// DevPodSubDomain holds a subdomain in the following form *.workspace.my-domain.com
	// +optional
	DevPodSubDomain string `json:"devPodSubDomain,omitempty"`

	// UISettings holds the settings for modifying the Loft user interface
	// +optional
	UISettings *uiv1.UISettingsConfig `json:"uiSettings,omitempty"`

	// VaultIntegration holds the vault integration configuration
	// +optional
	VaultIntegration *storagev1.VaultIntegrationSpec `json:"vault,omitempty"`

	// DisableLoftConfigEndpoint will disable setting config via the UI and config.management.loft.sh endpoint
	DisableConfigEndpoint bool `json:"disableConfigEndpoint,omitempty"`

	// AuthenticateVersionEndpoint will force authentication for the '/version' endpoint. Will only work with vCluster v0.27 & later
	AuthenticateVersionEndpoint bool `json:"authenticateVersionEndpoint,omitempty"`

	// Cloud holds the settings to be used exclusively in vCluster Cloud based
	// environments and deployments.
	Cloud *Cloud `json:"cloud,omitempty"`

	// CostControl holds the settings related to the Cost Control ROI dashboard and its metrics gathering infrastructure
	CostControl *CostControl `json:"costControl,omitempty"`

	// PlatformDB holds the settings related to the postgres database that platform uses to store data
	PlatformDB *PlatformDB `json:"platformDB,omitempty"`

	// ImageBuilder holds the settings related to the image builder
	ImageBuilder *ImageBuilder `json:"imageBuilder,omitempty"`

	// Database represents the database connection settings when deploying the platform with an embedded Kubernetes backed by kine
	Database *DatabaseKine `json:"database,omitempty"`
}

// Audit holds the audit configuration options for loft. Changing any options will require a loft restart
// to take effect.
type Audit struct {
	// If audit is enabled and incoming api requests will be logged based on the supplied policy.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// If true, the agent will not send back any audit logs to Loft itself.
	// +optional
	DisableAgentSyncBack bool `json:"disableAgentSyncBack,omitempty"`

	// Level is an optional log level for audit logs. Cannot be used together with policy
	// +optional
	Level int `json:"level,omitempty"`

	// The audit policy to use and log requests. By default loft will not log anything
	// +optional
	Policy AuditPolicy `json:"policy,omitempty"`

	// DataStoreEndpoint is an endpoint to store events in.
	// +optional
	DataStoreEndpoint string `json:"dataStoreEndpoint,omitempty"`

	// DataStoreMaxAge is the maximum number of hours to retain old log events in the datastore
	// +optional
	DataStoreMaxAge *int `json:"dataStoreTTL,omitempty"`

	// The path where to save the audit log files. This is required if audit is enabled. Backup log files will
	// be retained in the same directory.
	// +optional
	Path string `json:"path,omitempty"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	// +optional
	MaxAge int `json:"maxAge,omitempty"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	// +optional
	MaxBackups int `json:"maxBackups,omitempty"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	// +optional
	MaxSize int `json:"maxSize,omitempty"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	// +optional
	Compress bool `json:"compress,omitempty"`
}

// AuditPolicy describes the audit policy to use for loft
type AuditPolicy struct {
	// Rules specify the audit Level a request should be recorded at.
	// A request may match multiple rules, in which case the FIRST matching rule is used.
	// The default audit level is None, but can be overridden by a catch-all rule at the end of the list.
	// PolicyRules are strictly ordered.
	Rules []AuditPolicyRule `json:"rules,omitempty"`

	// OmitStages is a list of stages for which no events are created. Note that this can also
	// be specified per rule in which case the union of both are omitted.
	// +optional
	OmitStages []auditv1.Stage `json:"omitStages,omitempty"`
}

// AuditPolicyRule describes a policy for auditing
type AuditPolicyRule struct {
	// The Level that requests matching this rule are recorded at.
	Level auditv1.Level `json:"level"`

	// The users (by authenticated user name) this rule applies to.
	// An empty list implies every user.
	// +optional
	Users []string `json:"users,omitempty"`
	// The user groups this rule applies to. A user is considered matching
	// if it is a member of any of the UserGroups.
	// An empty list implies every user group.
	// +optional
	UserGroups []string `json:"userGroups,omitempty"`

	// The verbs that match this rule.
	// An empty list implies every verb.
	// +optional
	Verbs []string `json:"verbs,omitempty"`

	// Rules can apply to API resources (such as "pods" or "secrets"),
	// non-resource URL paths (such as "/api"), or neither, but not both.
	// If neither is specified, the rule is treated as a default for all URLs.

	// Resources that this rule matches. An empty list implies all kinds in all API groups.
	// +optional
	Resources []GroupResources `json:"resources,omitempty"`
	// Namespaces that this rule matches.
	// The empty string "" matches non-namespaced resources.
	// An empty list implies every namespace.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// NonResourceURLs is a set of URL paths that should be audited.
	// *s are allowed, but only as the full, final step in the path.
	// Examples:
	//  "/metrics" - Log requests for apiserver metrics
	//  "/healthz*" - Log all health checks
	// +optional
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`

	// OmitStages is a list of stages for which no events are created. Note that this can also
	// be specified policy wide in which case the union of both are omitted.
	// An empty list means no restrictions will apply.
	// +optional
	OmitStages []auditv1.Stage `json:"omitStages,omitempty" protobuf:"bytes,8,rep,name=omitStages"`

	// RequestTargets is a list of request targets for which events are created.
	// An empty list implies every request.
	// +optional
	RequestTargets []auditv1.RequestTarget `json:"requestTargets,omitempty"`

	// Clusters that this rule matches. Only applies to cluster requests.
	// If this is set, no events for non cluster requests will be created.
	// An empty list means no restrictions will apply.
	// +optional
	Clusters []string `json:"clusters,omitempty"`
}

// GroupResources represents resource kinds in an API group.
type GroupResources struct {
	// Group is the name of the API group that contains the resources.
	// The empty string represents the core API group.
	// +optional
	Group string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	// Resources is a list of resources this rule applies to.
	//
	// For example:
	// 'pods' matches pods.
	// 'pods/log' matches the log subresource of pods.
	// '*' matches all resources and their subresources.
	// 'pods/*' matches all subresources of pods.
	// '*/scale' matches all scale subresources.
	//
	// If wildcard is present, the validation rule will ensure resources do not
	// overlap with each other.
	//
	// An empty list implies all resources and subresources in this API groups apply.
	// +optional
	Resources []string `json:"resources,omitempty" protobuf:"bytes,2,rep,name=resources"`
	// ResourceNames is a list of resource instance names that the policy matches.
	// Using this field requires Resources to be specified.
	// An empty list implies that every instance of the resource is matched.
	// +optional
	ResourceNames []string `json:"resourceNames,omitempty" protobuf:"bytes,3,rep,name=resourceNames"`
}

// Apps holds configuration for apps that should be shown
type Apps struct {
	// If this option is true, loft will not try to parse the default apps
	// +optional
	NoDefault bool `json:"noDefault,omitempty"`

	// These are additional repositories that are parsed by loft
	// +optional
	Repositories []storagev1.HelmChartRepository `json:"repositories,omitempty"`

	// Predefined apps that can be selected in the Spaces > Space menu
	// +optional
	PredefinedApps []PredefinedApp `json:"predefinedApps,omitempty"`
}

// PredefinedApp holds information about a predefined app
type PredefinedApp struct {
	// Chart holds the repo/chart name of the predefined app
	// +optional
	Chart string `json:"chart"`

	// InitialVersion holds the initial version of this app.
	// This version will be selected automatically.
	// +optional
	InitialVersion string `json:"initialVersion,omitempty"`

	// InitialValues holds the initial values for this app.
	// The values will be prefilled automatically. There are certain
	// placeholders that can be used within the values that are replaced
	// by the loft UI automatically.
	// +optional
	InitialValues string `json:"initialValues,omitempty"`

	// Holds the cluster names where to display this app
	// +optional
	Clusters []string `json:"clusters,omitempty"`

	// Title is the name that should be displayed for the predefined app.
	// If empty the chart name is used.
	// +optional
	Title string `json:"title,omitempty"`

	// IconURL specifies an url to the icon that should be displayed for this app.
	// If none is specified the icon from the chart metadata is used.
	// +optional
	IconURL string `json:"iconUrl,omitempty"`

	// ReadmeURL specifies an url to the readme page of this predefined app. If empty
	// an url will be constructed to artifact hub.
	// +optional
	ReadmeURL string `json:"readmeUrl,omitempty"`
}

// OIDC holds oidc provider relevant information
type OIDC struct {
	// If true indicates that loft will act as an OIDC server
	Enabled bool `json:"enabled,omitempty"`

	// If true indicates that loft will allow wildcard '*' in client redirectURIs
	WildcardRedirect bool `json:"wildcardRedirect,omitempty"`

	// The clients that are allowed to request loft tokens
	Clients []OIDCClientSpec `json:"clients,omitempty"`
}

// Authentication and related connector types moved to storage/v1 — see
// staging/src/github.com/loft-sh/api/v4/pkg/apis/storage/v1/authentication_types.go.
// management/v1 still references the type via storagev1.Authentication on
// ConfigStatus and elsewhere; storage/v1 owns the canonical definition so
// the storage/v1.Tenant CRD can reference it without an import cycle.

type Cloud struct {
	// ReleaseChannel specifies the release channel for the cloud configuration.
	// This can be used to determine which updates or versions are applied.
	ReleaseChannel string `json:"releaseChannel,omitempty"`

	// MaintenanceWindow specifies the maintenance window for the cloud configuration.
	// This is a structured representation of the time window during which maintenance can occur.
	MaintenanceWindow MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type MaintenanceWindow struct {
	// DayOfWeek specifies the day of the week for the maintenance window.
	// It should be a string representing the day, e.g., "Monday", "Tuesday", etc.
	DayOfWeek string `json:"dayOfWeek,omitempty"`

	// TimeWindow specifies the time window for the maintenance.
	// It should be a string representing the time range in 24-hour format, in UTC, e.g., "02:00-03:00".
	TimeWindow string `json:"timeWindow,omitempty"`
}

type CostControl struct {
	// Enabled specifies whether the ROI dashboard should be available in the UI, and if the metrics infrastructure
	// that provides dashboard data is deployed
	Enabled *bool `json:"enabled,omitempty"`

	// Global are settings for globally managed components
	Global CostControlGlobalConfig `json:"global,omitempty"`

	// Cluster are settings for each cluster's managed components. These settings apply to all connected clusters
	// unless overridden by modifying the Cluster's spec
	Cluster CostControlClusterConfig `json:"cluster,omitempty"`

	// Settings specify price-related settings that are taken into account for the ROI dashboard calculations.
	Settings *CostControlSettings `json:"settings,omitempty"`
}

type PlatformDB struct {
	// StorageClass sets the storage class for the PersistentVolumeClaim used by the platform database statefulSet.
	StorageClass string `json:"storageClass,omitempty"`
}

type CostControlGlobalConfig struct {
	// Metrics these settings apply to metric infrastructure used to aggregate metrics across all connected clusters
	Metrics *storagev1.Metrics `json:"metrics,omitempty"`
}

type CostControlClusterConfig struct {
	// Metrics are settings applied to metric infrastructure in each connected cluster. These can be overridden in
	// individual clusters by modifying the Cluster's spec
	Metrics *storagev1.Metrics `json:"metrics,omitempty"`

	// OpenCost are settings applied to OpenCost deployments in each connected cluster. These can be overridden in
	// individual clusters by modifying the Cluster's spec
	OpenCost *storagev1.OpenCost `json:"opencost,omitempty"`
}

type CostControlSettings struct {
	// PriceCurrency specifies the currency.
	PriceCurrency string `json:"priceCurrency,omitempty"`

	// AvgCPUPricePerNode specifies the average CPU price per node.
	AvgCPUPricePerNode *CostControlResourcePrice `json:"averageCPUPricePerNode,omitempty"`

	// AvgRAMPricePerNode specifies the average RAM price per node.
	AvgRAMPricePerNode *CostControlResourcePrice `json:"averageRAMPricePerNode,omitempty"`

	// GPUSettings specifies GPU related settings.
	GPUSettings *CostControlGPUSettings `json:"gpuSettings,omitempty"`

	// ControlPlanePricePerCluster specifies the price of one physical cluster.
	ControlPlanePricePerCluster *CostControlResourcePrice `json:"controlPlanePricePerCluster,omitempty"`
}

type CostControlGPUSettings struct {
	// Enabled specifies whether GPU settings should be available in the UI.
	Enabled bool `json:"enabled,omitempty"`

	// AvgGPUPrice specifies the average GPU price.
	AvgGPUPrice *CostControlResourcePrice `json:"averageGPUPrice,omitempty"`
}

type CostControlResourcePrice struct {
	// Price specifies the price.
	Price float64 `json:"price,omitempty"`

	// TimePeriod specifies the time period for the price.
	TimePeriod string `json:"timePeriod,omitempty"`
}

type ImageBuilder struct {
	// Enabled specifies whether the remote image builder should be available.
	// If it's not available building ad-hoc images from a devcontainer.json is not supported
	Enabled *bool `json:"enabled,omitempty"`

	// Replicas is the number of desired replicas.
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources are compute resource required by the buildkit containers
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type DatabaseKine struct {
	// Enabled defines if the database should be used.
	Enabled bool `json:"enabled,omitempty"`

	// DataSource is the kine dataSource to use for the database. This depends on the database format.
	// This is optional for the external database. Examples:
	// * mysql: mysql://username:password@tcp(hostname:3306)/k3s
	// * postgres: postgres://username:password@hostname:5432/k3s
	DataSource string `json:"dataSource,omitempty"`

	// IdentityProvider is the kine identity provider to use when generating temporary authentication tokens for enhanced security.
	// This is optional for the external database. Examples:
	// * aws: RDS IAM Authentication
	IdentityProvider string `json:"identityProvider,omitempty"`

	// KeyFile is the key file to use for the database. This is optional.
	KeyFile string `json:"keyFile,omitempty"`

	// CertFile is the cert file to use for the database. This is optional.
	CertFile string `json:"certFile,omitempty"`

	// CaFile is the ca file to use for the database. This is optional.
	CaFile string `json:"caFile,omitempty"`

	// ExtraArgs are additional arguments to pass to Kine.
	ExtraArgs []string `json:"extraArgs,omitempty"`
}
