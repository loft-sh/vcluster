package volumes

import (
	"errors"

	snapshotTypes "github.com/loft-sh/vcluster/pkg/snapshot/types"
	corev1 "k8s.io/api/core/v1"
)

const (
	SnapshotClassNameLabel = "vcluster.loft.sh/csi-volumesnapshot-class"

	RequestPhaseNotStarted          SnapshotRequestPhase = ""
	RequestPhaseSkipped             SnapshotRequestPhase = "Skipped"
	RequestPhaseInProgress          SnapshotRequestPhase = "InProgress"
	RequestPhaseCompletedCleaningUp SnapshotRequestPhase = "CompletedCleaningUp"
	RequestPhaseCompleted           SnapshotRequestPhase = "Completed"
	RequestPhasePartiallyFailed     SnapshotRequestPhase = "PartiallyFailed"
	RequestPhaseFailed              SnapshotRequestPhase = "Failed"
	RequestPhaseFailedCleaningUp    SnapshotRequestPhase = "FailedCleaningUp"

	RequestPhaseCanceling SnapshotRequestPhase = "Canceling"
	RequestPhaseCanceled  SnapshotRequestPhase = "Canceled"

	// RequestPhaseUndefined is a special request phase used in case of an error
	// in volume snapshot phase transition.
	RequestPhaseUndefined SnapshotRequestPhase = "Undefined"
)

var (
	ErrNextPhaseNotDefined   error = errors.New("next phase not defined")
	ErrFailedPhaseNotDefined error = errors.New("failure phase transition not defined")
)

// SnapshotRequestPhase describes the current state of the snapshot creation process.
type SnapshotRequestPhase string

// Next returns the next phase in the volume snapshot creation process. In case phase transition is
// not defined, it returns Undefined.
func (s SnapshotRequestPhase) Next() SnapshotRequestPhase {
	var next SnapshotRequestPhase
	switch s {
	case RequestPhaseNotStarted:
		next = RequestPhaseInProgress
	case RequestPhaseInProgress:
		next = RequestPhaseCompletedCleaningUp
	case RequestPhaseCompletedCleaningUp:
		next = RequestPhaseCompleted
	case RequestPhaseFailedCleaningUp:
		next = RequestPhaseFailed
	case RequestPhaseCanceling:
		next = RequestPhaseCanceled
	default:
		next = RequestPhaseUndefined
	}
	return next
}

// Failed returns the next phase in the volume snapshot creation process in case of an error in the
// current phase.
func (s SnapshotRequestPhase) Failed() SnapshotRequestPhase {
	var next SnapshotRequestPhase
	switch s {
	case RequestPhaseInProgress:
		next = RequestPhaseFailedCleaningUp
	case RequestPhaseCompletedCleaningUp:
		next = RequestPhaseFailedCleaningUp
	default:
		next = RequestPhaseFailed
	}
	return next
}

// SnapshotsRequest specifies how to create snapshots for multiple PVCs.
type SnapshotsRequest struct {
	Requests []SnapshotRequest `json:"requests,omitempty"`
}

// SnapshotRequest specifies how to create a snapshot for a PVC.
type SnapshotRequest struct {
	PersistentVolumeClaim corev1.PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// CSIDriver that provisioned the PVC.
	CSIDriver string `json:"csiDriver"`

	// VolumeSnapshotClassName to use when creating a VolumeSnapshot resource.
	VolumeSnapshotClassName string `json:"volumeSnapshotClassName,omitempty"`
}

// SnapshotsStatus shows the current status of the overall volume snapshot (all PVCs in a snapshot request).
type SnapshotsStatus struct {
	Phase     SnapshotRequestPhase        `json:"phase,omitempty"`
	Snapshots map[string]SnapshotStatus   `json:"snapshots,omitempty"`
	Error     snapshotTypes.SnapshotError `json:"error,omitempty"`
}

// Done returns true if the process of taking all volume snapshots has finished, otherwise it
// returns false.
func (s SnapshotsStatus) Done() bool {
	// check overall snapshots status
	done := s.Phase == RequestPhaseCompleted ||
		s.Phase == RequestPhasePartiallyFailed ||
		s.Phase == RequestPhaseFailed ||
		s.Phase == RequestPhaseSkipped ||
		s.Phase == RequestPhaseCanceled
	if !done {
		return false
	}

	// check every volume snapshot status
	for _, status := range s.Snapshots {
		if !status.Done() {
			return false
		}
	}

	// taking snapshot has not yet started, or it is still in progress
	return true
}

// SnapshotStatus shows the current status of a single PVC snapshot.
type SnapshotStatus struct {
	Phase          SnapshotRequestPhase        `json:"phase,omitempty"`
	SnapshotHandle string                      `json:"snapshotHandle,omitempty"`
	Error          snapshotTypes.SnapshotError `json:"error,omitempty"`
}

// Equals checks if the snapshot status is identical to another snapshot status.
func (s SnapshotStatus) Equals(other SnapshotStatus) bool {
	return s.Phase == other.Phase &&
		s.SnapshotHandle == other.SnapshotHandle &&
		s.Error.Equals(other.Error)
}

// Done returns true if the process of taking a volume snapshot has finished, otherwise it returns
// false.
func (s SnapshotStatus) Done() bool {
	return s.Phase == RequestPhaseCompleted || s.Phase == RequestPhaseSkipped || s.Phase == RequestPhaseFailed
}

// CleaningUp returns true if the volume snapshot is still being cleaned up.
func (s SnapshotStatus) CleaningUp() bool {
	return s.Phase == RequestPhaseCompletedCleaningUp || s.Phase == RequestPhaseFailedCleaningUp
}
