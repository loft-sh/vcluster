package types

/*
* Keep the dependencies of this package minimal to make it easy to import
 */

type SyncerTelemetryConfig struct {
	Disabled           string `json:"disabled,omitempty"`
	InstanceCreator    string `json:"instanceCreator,omitempty"`
	InstanceCreatorUID string `json:"instanceCreatorUID,omitempty"`
}
type SyncerTelemetryRequest struct {
	InstanceProperties  SyncerInstanceProperties `json:"instanceProperties,omitempty"`
	Events              []*Event                 `json:"events,omitempty"`
	TimeSinceLastUpload *int                     `json:"timeSinceLastUpload,omitempty"`
	Token               string                   `json:"token,omitempty"`
}

type KubernetesVersion struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
}

type SyncerInstanceProperties struct {
	// vcluster instance UID
	UID                      string             `json:"uid,omitempty"`
	InstanceCreator          string             `json:"instanceCreator,omitempty"`
	InstanceCreatorUID       string             `json:"instanceCreatorUID,omitempty"`
	Arch                     string             `json:"arch,omitempty"`
	OS                       string             `json:"os,omitempty"`
	SyncerVersion            string             `json:"syncerVersion,omitempty"`
	SyncerFlags              string             `json:"syncerFlags,omitempty"`
	VclusterServiceType      string             `json:"vclusterServiceType,omitempty"`
	VirtualKubernetesVersion *KubernetesVersion `json:"virtualKubernetesVersion,omitempty"`
	HostKubernetesVersion    *KubernetesVersion `json:"hostKubernetesVersion,omitempty"`
	SyncerPodsReady          int                `json:"syncerPodsReady,omitempty"`
	SyncerPodsFailing        int                `json:"syncerPodsFailing,omitempty"`
	SyncerPodCreated         int                `json:"syncerPodCreated,omitempty"`
	SyncerPodRestarts        int                `json:"syncerPodRestarts,omitempty"`
	SyncerMemoryRequests     int                `json:"syncerMemoryRequests,omitempty"`
	SyncerMemoryLimits       int                `json:"syncerMemoryLimits,omitempty"`
	SyncerCPURequests        int                `json:"syncerCPURequests,omitempty"`
	SyncerCPULimits          int                `json:"syncerCPULimits,omitempty"`
}

type EventType string

const (
	EventAPIRequest        EventType = "APIRequest" // TODO: add code to record EventAPIRequest event
	EventResourceSync      EventType = "ResourceSync"
	EventLeadershipStarted EventType = "LeadershipStarted"
	EventLeadershipStopped EventType = "EventLeadershipStopped"
	EventSyncerStarted     EventType = "SyncerStarted"
)

type Event struct {
	Type EventType `json:"type,omitempty"`
	// Time represents Unix timestampt in microseconds
	Time int `json:"time,omitempty"`

	// Additional fields used by EventAPIRequest and EventResourceSync
	Success        bool   `json:"success,omitempty"`
	ProcessingTime int    `json:"processingTime,omitempty"`
	Errors         string `json:"errors,omitempty"`
	Group          string `json:"group,omitempty"`
	Version        string `json:"version,omitempty"`
	Kind           string `json:"kind,omitempty"`

	// Additional fields used by EventAPIRequest
	UserAgent string `json:"userAgent,omitempty"`
}

type SyncerFlags struct {
	SetFlags    map[string]bool `json:"setFlags,omitempty"`
	Controllers []string        `json:"controllers,omitempty"`
}
