// Package vclusterconfig contains configuration types for vCluster Platform features.
// These types are used for parsing vCluster YAML configuration and are imported by vcluster.
package vclusterconfig

import (
	"encoding/json"
	"errors"
	"time"
)

type PlatformConfig struct {
	Sleep             *Sleep             `json:"sleep,omitempty"     yaml:"sleep,omitempty"`
	Snapshots         *Snapshots         `json:"snapshots,omitempty" yaml:"snapshots,omitempty"`
	Deletion          *Deletion          `json:"deletion,omitempty"  yaml:"deletion,omitempty"`
	Platform          *Platform          `json:"platform,omitempty"  yaml:"platform,omitempty"`
	NetrisIntegration *NetrisIntegration `json:"netris,omitempty"    yaml:"netris,omitempty"`
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

// Sleep holds configuration for automatically putting the virtual cluster to sleep.
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
	// Schedule specifies a scheduled time in Cron format, see https://en.wikipedia.org/wiki/Cron for a virtual cluster snapshot to be taken
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

	// Volumes specifies configuration for volume snapshots
	// +optional
	Volumes *SnapshotVolumes `json:"volumes,omitempty" yaml:"volumes,omitempty"`
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

// SnapshotVolumes holds volume snapshot configuration
type SnapshotVolumes struct {
	// Enabled specifies whether a snapshot should also include volumes in the snapshot
	// +optional
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
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
	// AfterInactivity specifies after how long of inactivity the virtual cluster will be deleted.
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
