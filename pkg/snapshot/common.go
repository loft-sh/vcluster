package snapshot

import "k8s.io/apimachinery/pkg/apis/meta/v1"

type LongRunningRequest interface {
	GetPhase() RequestPhase
}

const (
	APIVersion = "v1beta1"

	RequestPhaseNotStarted              RequestPhase = ""
	RequestPhaseCreatingVolumeSnapshots RequestPhase = "CreatingVolumeSnapshots"
	RequestPhaseCreatingEtcdBackup      RequestPhase = "CreatingEtcdBackup"
	RequestPhaseCompleted               RequestPhase = "Completed"
	RequestPhasePartiallyFailed         RequestPhase = "PartiallyFailed"
	RequestPhaseFailed                  RequestPhase = "Failed"
)

type RequestPhase string

type RequestMetadata struct {
	Name              string  `json:"name"`
	CreationTimestamp v1.Time `json:"creationTimestamp,omitempty"`
}
