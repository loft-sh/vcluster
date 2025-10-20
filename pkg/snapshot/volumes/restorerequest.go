package volumes

import corev1 "k8s.io/api/core/v1"

// RestoreRequestSpec specifies how to restore volumes from snapshots.
type RestoreRequestSpec struct {
	Requests []RestoreRequest `json:"requests,omitempty"`
}

// RestoreRequest specifies how to restore a volume from a snapshot.
type RestoreRequest struct {
	// PersistentVolumeClaim to restore.
	PersistentVolumeClaim corev1.PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// CSIDriver that provisions the PVC.
	CSIDriver string `json:"csiDriver"`

	// VolumeSnapshotClassName to use when creating the pre-provisioned VolumeSnapshot resource from
	// which the PersistentVolumeClaim is restored.
	VolumeSnapshotClassName string `json:"volumeSnapshotClassName,omitempty"`

	// SnapshotHandle specifies the snapshot identifier (in the storage backend). It is used to
	// create the pre-provisioned VolumeSnapshotContent resource from which the
	// PersistentVolumeClaim is restored.
	SnapshotHandle string `json:"snapshotHandle,omitempty"`
}

// RestoreRequestStatus shows the current status of the restore request.
type RestoreRequestStatus struct {
	Phase                  SnapshotRequestPhase     `json:"phase,omitempty"`
	PersistentVolumeClaims map[string]RestoreStatus `json:"persistentVolumeClaims,omitempty"`
	Error                  RestoreError             `json:"error,omitempty"`
}

// Done returns true if the process of restoring all volumes has finished, otherwise it returns
// false.
func (s RestoreRequestStatus) Done() bool {
	// check overall restores status
	done := s.Phase == RequestPhaseCompleted ||
		s.Phase == RequestPhasePartiallyFailed ||
		s.Phase == RequestPhaseFailed ||
		s.Phase == RequestPhaseSkipped
	if !done {
		return false
	}

	// check every volume restore status
	for _, status := range s.PersistentVolumeClaims {
		if !status.Done() {
			return false
		}
	}

	// restoring volumes have not yet started, or it is still in progress
	return true
}

// RestoreStatus shows the current status of a single PVC restore.
type RestoreStatus struct {
	Phase SnapshotRequestPhase `json:"phase,omitempty"`
	Error RestoreError         `json:"error,omitempty"`
}

// Equals checks if the restore status is identical to another restore status.
func (s RestoreStatus) Equals(other RestoreStatus) bool {
	return s.Phase == other.Phase &&
		s.Error.Equals(other.Error)
}

// Done returns true if the process of restoring a volume has finished, otherwise it returns
// false.
func (s RestoreStatus) Done() bool {
	return s.Phase == RequestPhaseCompleted || s.Phase == RequestPhaseSkipped || s.Phase == RequestPhaseFailed
}

func (s RestoreStatus) CleaningUp() bool {
	return s.Phase == RequestPhaseCompletedCleaningUp || s.Phase == RequestPhaseFailedCleaningUp
}

// RestoreError describes the error that occurred while restoring the volume.
type RestoreError struct {
	Message string `json:"message,omitempty"`
}

// Equals checks if the restore error is identical to another restore error.
func (err RestoreError) Equals(other RestoreError) bool {
	return err.Message == other.Message
}
