package volumes

import (
	snapshotTypes "github.com/loft-sh/vcluster/pkg/snapshot/types"
	corev1 "k8s.io/api/core/v1"
)

const (
	SnapshotClassNameLabel = "vcluster.loft.sh/csi-volumesnapshot-class"

	RequestPhaseNotStarted      SnapshotRequestPhase = ""
	RequestPhaseSkipped         SnapshotRequestPhase = "Skipped"
	RequestPhaseInProgress      SnapshotRequestPhase = "InProgress"
	RequestPhaseCompleted       SnapshotRequestPhase = "Completed"
	RequestPhasePartiallyFailed SnapshotRequestPhase = "PartiallyFailed"
	RequestPhaseFailed          SnapshotRequestPhase = "Failed"
)

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

// SnapshotRequestPhase describes the current state of the snapshot creation process.
type SnapshotRequestPhase string

// SnapshotsStatus shows the current status of the snapshot request.
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
		s.Phase == RequestPhaseSkipped
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
