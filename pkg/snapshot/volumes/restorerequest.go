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

// RestoreStatus shows the current status of a single PVC restore.
type RestoreStatus struct {
	Phase SnapshotRequestPhase `json:"phase,omitempty"`
	Error RestoreError         `json:"error,omitempty"`
}

func (s RestoreStatus) Equals(other RestoreStatus) bool {
	return s.Phase == other.Phase &&
		s.Error.Equals(other.Error)
}

func (s RestoreStatus) Done() bool {
	return s.Phase == RequestPhaseCompleted || s.Phase == RequestPhaseSkipped || s.Phase == RequestPhaseFailed
}

type RestoreError struct {
	Message string `json:"message,omitempty"`
}

func (err RestoreError) Equals(other RestoreError) bool {
	return err.Message == other.Message
}
