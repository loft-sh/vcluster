package snapshot

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	RequestPhaseCanceling RequestPhase = "Canceling"
	RequestPhaseCanceled  RequestPhase = "Canceled"

	RequestPhaseDeleting                RequestPhase = "Deleting"
	RequestPhaseDeletingVolumeSnapshots RequestPhase = "DeletingVolumeSnapshots"
	RequestPhaseDeletingEtcdBackup      RequestPhase = "DeletingEtcdBackup"
	RequestPhaseDeleted                 RequestPhase = "Deleted"

	RequestPhaseUnknown RequestPhase = "Unknown"
)

type RequestPhase string

func (r RequestPhase) Next() RequestPhase {
	var next RequestPhase
	switch r {
	case RequestPhaseCreatingVolumeSnapshots:
		next = RequestPhaseCreatingEtcdBackup
	case RequestPhaseCreatingEtcdBackup:
		next = RequestPhaseCompleted
	case RequestPhaseCanceling:
		next = RequestPhaseCanceled
	case RequestPhaseDeletingVolumeSnapshots:
		next = RequestPhaseDeletingEtcdBackup
	case RequestPhaseDeletingEtcdBackup:
		next = RequestPhaseDeleted
	default:
		next = RequestPhaseUnknown
	}

	return next
}

type RequestMetadata struct {
	Name              string      `json:"name"`
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`
}
