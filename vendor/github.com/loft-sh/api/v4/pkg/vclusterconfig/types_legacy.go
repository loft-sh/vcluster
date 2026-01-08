package vclusterconfig

// LegacyPlatformConfig describes platform configuration for a vCluster. This is provided through the vcluster.yaml
// under "external.platform".
//
// Deprecated: Use the new top-level config types instead:
//   - AutoSleep -> use top-level Sleep
//   - AutoDelete -> use top-level Deletion
//   - AutoSnapshot -> use top-level Snapshots
//   - APIKey, Project -> use top-level Platform
type LegacyPlatformConfig struct {
	APIKey interface{} `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`

	// AutoSleep holds configuration for automatic sleep and wakeup
	// Deprecated: Use top-level sleepMode instead.
	// +optional
	AutoSleep *LegacyPlatformAutoSleep `json:"autoSleep,omitempty" yaml:"autoSleep,omitempty"`

	// AutoDelete holds configuration for automatic delete
	// Deprecated: Use top-level Deletion instead.
	// +optional
	AutoDelete *LegacyAutoDelete `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`

	// Project holds project name where vcluster should be imported
	Project string `json:"project,omitempty" yaml:"project,omitempty"`

	// AutoSnapshot holds configuration for automatic snapshot of vclusters
	// Deprecated: Use top-level Snapshots instead.
	// +optional
	AutoSnapshot *LegacyAutoSnapshot `json:"autoSnapshot,omitempty" yaml:"autoSnapshot,omitempty"`
}

// LegacyPlatformAutoSleep holds configuration for automatic sleep and wakeup.
//
// Deprecated: Use top-level sleepMode instead.
type LegacyPlatformAutoSleep struct {
	// AfterInactivity specifies after how many seconds of inactivity the virtual cluster should sleep
	// +optional
	AfterInactivity int64 `json:"afterInactivity,omitempty" yaml:"afterInactivity,omitempty"`

	// Schedule specifies scheduled virtual cluster sleep in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// Note: timezone defined in the schedule string will be ignored. Use ".Timezone" field instead.
	// +optional
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Timezone specifies time zone used for scheduled virtual cluster operations. Defaults to UTC.
	// Accepts the same format as time.LoadLocation() in Go (https://pkg.go.dev/time#LoadLocation).
	// The value should be a location name corresponding to a file in the IANA Time Zone database, such as "America/New_York".
	// +optional
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// AutoWakeup holds configuration for automatic wakeup
	// +optional
	AutoWakeup *LegacyPlatformAutoWakeup `json:"autoWakeup,omitempty" yaml:"autoWakeup,omitempty"`
}

// LegacyPlatformAutoWakeup holds configuration for automatic wakeup.
//
// Deprecated: Use top-level sleepMode.autoWakeup instead.
type LegacyPlatformAutoWakeup struct {
	// Schedule specifies scheduled wakeup from sleep in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// Note: timezone defined in the schedule string will be ignored. The timezone for the autoSleep schedule will be
	// used
	// +optional
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`
}

// LegacyAutoDelete holds configuration for automatic delete.
//
// Deprecated: Use top-level Deletion instead.
type LegacyAutoDelete struct {
	// AfterInactivity specifies after how many seconds of inactivity the virtual cluster be deleted
	// +optional
	AfterInactivity int64 `json:"afterInactivity,omitempty" yaml:"afterInactivity,omitempty"`
}

// LegacyAutoSnapshot holds configuration for automatic snapshot of vclusters.
//
// Deprecated: Use top-level Snapshots instead.
type LegacyAutoSnapshot struct {
	// Enable defines whether auto snapshot is enabled for the virtual cluster
	// +optional
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Timezone specifies time zone used for scheduled virtual cluster operations. Defaults to UTC.
	// Accepts the same format as time.LoadLocation() in Go (https://pkg.go.dev/time#LoadLocation).
	// The value should be a location name corresponding to a file in the IANA Time Zone database, such as "America/New_York".
	// +optional
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// Schedule specifies a scheduled time in Cron format, see https://en.wikipedia.org/wiki/Cron for a virtual cluster snapshot to be taken
	// +optional
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Volumes specifies a set of configuration for the volume snapshot
	// +optional
	Volumes LegacyVolumeSnapshot `json:"volumes" yaml:"volumes"`

	// Storage specifies a set of configuration where the snapshot will be stored
	// +optional
	Storage LegacyScheduledSnapshotStorage `json:"storage,omitempty" yaml:"storage,omitempty"`

	// Retention specifies a set of configuration for how long the snapshot will be kept
	// +optional
	Retention SnapshotRetention `json:"retention,omitempty" yaml:"retention,omitempty"`
}

// LegacyVolumeSnapshot holds volume snapshot configuration.
//
// Deprecated: Use SnapshotVolumes instead.
type LegacyVolumeSnapshot struct {
	// Enabled specifies whether a snapshot should also include volumes in the snapshot
	// +optional
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

// LegacyScheduledSnapshotStorage holds snapshot storage configuration.
//
// Deprecated: Use SnapshotStorage instead.
type LegacyScheduledSnapshotStorage struct {
	// Type specifies supported type of storage services for a snapshot S3/OCI/Container, see https://www.vcluster.com/docs/vcluster/manage/backup-restore#store-snapshots-in-s3-buckets
	// +optional
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// S3 holds configuration for storing snapshots in S3-compatible bucket
	// +optional
	S3 LegacyS3Storage `json:"s3,omitempty" yaml:"s3,omitempty"`

	// OCI holds configuration for storing snapshots in OCI image registries
	// +optional
	OCI LegacyOCIStorage `json:"oci,omitempty" yaml:"oci,omitempty"`

	// Container holds configuration for storing snapshots as local files inside a vCluster container
	// +optional
	Container LegacyContainerStorage `json:"container,omitempty" yaml:"container,omitempty"`
}

// LegacyS3Storage holds S3 storage configuration.
//
// Deprecated: Use SnapshotStorageS3 instead.
type LegacyS3Storage struct {
	// Url specifies url to the storage service
	// +optional
	Url string `json:"url,omitempty" yaml:"url,omitempty"`

	// Credential secret with the S3 Credentials, it should contain AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN
	// +optional
	Credential *SnapshotSecretCredential `json:"credential,omitempty" yaml:"credential,omitempty"`
}

// LegacyOCIStorage holds OCI registry storage configuration.
//
// Deprecated: Use SnapshotStorageOCI instead.
type LegacyOCIStorage struct {
	// Repository OCI repository to store the snapshot
	// +optional
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`

	// Credential secret with the OCI Credentials
	// +optional
	Credential *SnapshotSecretCredential `json:"credential,omitempty" yaml:"credential,omitempty"`

	// Username to authenticate with the OCI registry
	// +optional
	Username string `json:"username,omitempty" yaml:"username,omitempty" url:"username"`

	// Password to authenticate with the OCI registry
	// +optional
	Password string `json:"password,omitempty" yaml:"password,omitempty" url:"password,base64"`
}

// LegacyContainerStorage holds container local storage configuration.
//
// Deprecated: Use SnapshotStorageContainer instead.
type LegacyContainerStorage struct {
	// Path specifies directory to store the snapshot
	// +optional
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Volume specifies which volume needs to be mounted into the container to store the snapshot
	// +optional
	Volume LegacyContainerStorageVolume `json:"volume,omitempty" yaml:"volume,omitempty"`
}

// LegacyContainerStorageVolume holds volume mount configuration.
//
// Deprecated: Use SnapshotStorageContainerVolume instead.
type LegacyContainerStorageVolume struct {
	// Name to be used to mount the volume
	// +optional
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Path to the volume mount
	// +optional
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

// LegacySleepMode holds the deprecated sleepMode configuration format.
//
// Deprecated: Use the top-level Sleep type instead.
type LegacySleepMode struct {
	Enabled    bool                   `json:"enabled,omitempty"    yaml:"enabled,omitempty"`
	TimeZone   string                 `json:"timeZone,omitempty"   yaml:"timeZone,omitempty"`
	AutoSleep  *LegacySleepAuto       `json:"autoSleep,omitempty"  yaml:"autoSleep,omitempty"`
	AutoWakeup *LegacySleepAutoWakeup `json:"autoWakeup,omitempty" yaml:"autoWakeup,omitempty"`
}

// LegacySleepAuto holds configuration for automatic sleep and wakeup
//
// Deprecated: Use SleepAuto instead.
type LegacySleepAuto struct {
	// AfterInactivity represents how long a vCluster can be idle before workloads are automatically put to sleep
	AfterInactivity Duration `json:"afterInactivity,omitempty" yaml:"afterInactivity,omitempty"`

	// Schedule represents a cron schedule for when to sleep workloads
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Exclude holds configuration for labels that, if present, will prevent a workload from going to sleep
	Exclude SleepAutoExclusion `json:"exclude,omitempty"`
}

// LegacySleepAutoWakeup holds the cron schedule to wake workloads automatically
//
// Deprecated: Use SleepAutoWakeup instead.
type LegacySleepAutoWakeup struct {
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`
}
