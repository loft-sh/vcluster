package snapshot

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

const (
	SnapshotClassNameLabel = "vcluster.loft.sh/csi-volumesnapshot-class"

	VolumeSnapshotPhaseNotStarted          VolumeSnapshotRequestPhase = ""
	VolumeSnapshotPhaseSkipped             VolumeSnapshotRequestPhase = "Skipped"
	VolumeSnapshotPhaseInProgress          VolumeSnapshotRequestPhase = "InProgress"
	VolumeSnapshotPhaseCompletedCleaningUp VolumeSnapshotRequestPhase = "CompletedCleaningUp"
	VolumeSnapshotPhaseCompleted           VolumeSnapshotRequestPhase = "Completed"
	VolumeSnapshotPhasePartiallyFailed     VolumeSnapshotRequestPhase = "PartiallyFailed"
	VolumeSnapshotPhaseFailed              VolumeSnapshotRequestPhase = "Failed"
	VolumeSnapshotPhaseFailedCleaningUp    VolumeSnapshotRequestPhase = "FailedCleaningUp"

	VolumeSnapshotPhaseCanceling VolumeSnapshotRequestPhase = "Canceling"
	VolumeSnapshotPhaseCanceled  VolumeSnapshotRequestPhase = "Canceled"

	VolumeSnapshotPhaseDeleting VolumeSnapshotRequestPhase = "Deleting"
	VolumeSnapshotPhaseDeleted  VolumeSnapshotRequestPhase = "Deleted"

	// VolumeSnapshotPhaseUndefined is a special request phase used in case of an error
	// in volume snapshot phase transition.
	VolumeSnapshotPhaseUndefined VolumeSnapshotRequestPhase = "Undefined"
)

var (
	ErrNextVolumeSnapshotPhaseNotDefined   = errors.New("next phase not defined")
	ErrFailedVolumeSnapshotPhaseNotDefined = errors.New("failure phase transition not defined")
)

// VolumeSnapshotRequestPhase describes the current state of the volume snapshot creation process.
type VolumeSnapshotRequestPhase string

// Next returns the next phase in the volume snapshot creation process. In case phase transition is
// not defined, it returns Undefined.
func (s VolumeSnapshotRequestPhase) Next() VolumeSnapshotRequestPhase {
	switch s {
	case VolumeSnapshotPhaseNotStarted:
		return VolumeSnapshotPhaseInProgress
	case VolumeSnapshotPhaseInProgress:
		return VolumeSnapshotPhaseCompletedCleaningUp
	case VolumeSnapshotPhaseCompletedCleaningUp:
		return VolumeSnapshotPhaseCompleted
	case VolumeSnapshotPhaseFailedCleaningUp:
		return VolumeSnapshotPhaseFailed
	case VolumeSnapshotPhaseCanceling:
		return VolumeSnapshotPhaseCanceled
	case VolumeSnapshotPhaseDeleting:
		return VolumeSnapshotPhaseDeleted
	default:
		return VolumeSnapshotPhaseUndefined
	}
}

// Failed returns the next phase in the volume snapshot creation process in case of an error in the
// current phase.
func (s VolumeSnapshotRequestPhase) Failed() VolumeSnapshotRequestPhase {
	switch s {
	case VolumeSnapshotPhaseInProgress:
		return VolumeSnapshotPhaseFailedCleaningUp
	case VolumeSnapshotPhaseCompletedCleaningUp:
		return VolumeSnapshotPhaseFailedCleaningUp
	default:
		return VolumeSnapshotPhaseFailed
	}
}

// VolumeSnapshotsRequest specifies how to create snapshots for multiple PVCs.
type VolumeSnapshotsRequest struct {
	Requests []VolumeSnapshotRequest `json:"requests,omitempty"`
}

// VolumeSnapshotRequest specifies how to create a snapshot for a PVC.
type VolumeSnapshotRequest struct {
	PersistentVolumeClaim corev1.PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// CSIDriver that provisioned the PVC.
	CSIDriver string `json:"csiDriver"`

	// VolumeSnapshotClassName to use when creating a VolumeSnapshot resource.
	VolumeSnapshotClassName string `json:"volumeSnapshotClassName,omitempty"`
}

// VolumeSnapshotsStatus shows the current status of all PVC snapshots in a snapshot request.
type VolumeSnapshotsStatus struct {
	Phase     VolumeSnapshotRequestPhase      `json:"phase,omitempty"`
	Snapshots map[string]VolumeSnapshotStatus `json:"snapshots,omitempty"`
	Error     SnapshotError                   `json:"error,omitempty"`
}

func (s VolumeSnapshotsStatus) Done() bool {
	done := s.Phase == VolumeSnapshotPhaseCompleted ||
		s.Phase == VolumeSnapshotPhasePartiallyFailed ||
		s.Phase == VolumeSnapshotPhaseFailed ||
		s.Phase == VolumeSnapshotPhaseSkipped ||
		s.Phase == VolumeSnapshotPhaseCanceled ||
		s.Phase == VolumeSnapshotPhaseDeleted
	if !done {
		return false
	}

	for _, status := range s.Snapshots {
		if !status.Done() {
			return false
		}
	}

	return true
}

func (s VolumeSnapshotsStatus) IsDeletingVolumeSnapshots() bool {
	return s.Phase == VolumeSnapshotPhaseDeleting || s.Phase == VolumeSnapshotPhaseCanceling
}

func (s VolumeSnapshotsStatus) RecreateVolumeSnapshotsWhenDeleting() bool {
	return s.Phase == VolumeSnapshotPhaseDeleting
}

// VolumeSnapshotStatus shows the current status of a single PVC snapshot.
type VolumeSnapshotStatus struct {
	Phase          VolumeSnapshotRequestPhase `json:"phase,omitempty"`
	SnapshotHandle string                     `json:"snapshotHandle,omitempty"`
	Error          SnapshotError              `json:"error,omitempty"`
}

func (s VolumeSnapshotStatus) Equals(other VolumeSnapshotStatus) bool {
	return s.Phase == other.Phase &&
		s.SnapshotHandle == other.SnapshotHandle &&
		s.Error.Equals(other.Error)
}

func (s VolumeSnapshotStatus) Done() bool {
	return s.Phase == VolumeSnapshotPhaseCompleted || s.Phase == VolumeSnapshotPhaseSkipped || s.Phase == VolumeSnapshotPhaseFailed
}

func (s VolumeSnapshotStatus) CleaningUp() bool {
	return s.Phase == VolumeSnapshotPhaseCompletedCleaningUp || s.Phase == VolumeSnapshotPhaseFailedCleaningUp
}

func (s VolumeSnapshotStatus) IsDeletingVolumeSnapshot() bool {
	return s.Phase == VolumeSnapshotPhaseDeleting || s.Phase == VolumeSnapshotPhaseCanceling
}

func (s VolumeSnapshotStatus) RecreateVolumeSnapshotWhenDeleting() bool {
	return s.Phase == VolumeSnapshotPhaseDeleting
}

func (s VolumeSnapshotStatus) IsVolumeSnapshotMaybeCreated() bool {
	return s.Phase == VolumeSnapshotPhaseNotStarted ||
		s.Phase == VolumeSnapshotPhaseInProgress ||
		s.Phase == VolumeSnapshotPhaseCompletedCleaningUp ||
		s.Phase == VolumeSnapshotPhaseCompleted
}
