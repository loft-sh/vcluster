package telemetry

type SyncerTelemetryRequest struct {
	InstanceProperties  SyncerInstanceProperties `json:"instanceProperties,omitempty"`
	Events              []*Event                 `json:"events,omitempty"`
	TimeSinceLastUpload *int                     `json:"timeSinceLastUpload,omitempty"`
	Token               string                   `json:"token,omitempty"`
}

type SyncerInstanceProperties struct {
	// vcluster instance UID
	UID                      string `json:"uid,omitempty"`
	InstanceCreatorType      string `json:"instanceCreatorType,omitempty"`
	InstanceCreatorUID       string `json:"instanceCreatorUID,omitempty"`
	Arch                     string `json:"arch,omitempty"`
	OS                       string `json:"os,omitempty"`
	SyncerVersion            string `json:"syncerVersion,omitempty"`
	VirtualKubernetesVersion string `json:"virtualKubernetesVersion,omitempty"`
	HostKubernetesVersion    string `json:"hostKubernetesVersion,omitempty"`
	SyncerPodsReady          int    `json:"syncerPodsReady,omitempty"`
	SyncerPodsFailing        int    `json:"syncerPodsFailing,omitempty"`
	SyncerPodCreated         int    `json:"syncerPodCreated,omitempty"`
	SyncerPodRestarts        int    `json:"syncerPodRestarts,omitempty"`
	SyncerFlags              string `json:"syncerFlags,omitempty"`
	SyncerMemoryRequests     int    `json:"syncerMemoryRequests,omitempty"`
	SyncerMemoryLimits       int    `json:"syncerMemoryLimits,omitempty"`
	SyncerCpuRequests        int    `json:"syncerCpuRequests,omitempty"`
	SyncerCpuLimits          int    `json:"syncerCpuLimits,omitempty"`
	CachedObjectsCount       string `json:"cachedObjectsCount,omitempty"`
	VclusterServiceType      string `json:"vclusterServiceType,omitempty"`
}

type EventType string

const (
	EventApiRequest        EventType = "ApiRequest" // TODO: add code to record ApiRequest event
	EventResourceSync      EventType = "ResourceSync"
	EventLeadershipStarted EventType = "LeadershipStarted"
	EventLeadershipStopped EventType = "EventLeadershipStopped"
	EventSyncerStarted     EventType = "SyncerStarted" // TODO: add code that will send this after startup immediately?
)

type Event struct {
	Type EventType `json:"type,omitempty"`

	// Additional fields used by EventApiRequest and EventResourceSync

	Success        bool   `json:"success,omitempty"`
	ProcessingTime int    `json:"processingTime,omitempty"`
	Errors         string `json:"errors,omitempty"`
	Group          string `json:"group,omitempty"`
	Version        string `json:"version,omitempty"`
	Kind           string `json:"kind,omitempty"`

	// Additional fields used by EventApiRequest
	UserAgent string `json:"userAgent,omitempty"`
}
