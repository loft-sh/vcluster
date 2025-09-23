package volumes

import corev1 "k8s.io/api/core/v1"

const (
	SnapshotClassNameLabel = "vcluster.loft.sh/csi-volumesnapshot-class"

	RequestPhaseNotStarted SnapshotRequestPhase = ""
	RequestPhaseInProgress SnapshotRequestPhase = "InProgress"
	RequestPhaseCleaningUp SnapshotRequestPhase = "CleaningUp"
	RequestPhaseCompleted  SnapshotRequestPhase = "Completed"
	RequestPhaseSkipped    SnapshotRequestPhase = "Skipped"
	RequestPhaseFailed     SnapshotRequestPhase = "Failed"
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

type SnapshotRequestPhase string

// SnapshotsStatus shows the current status of the snapshot request.
type SnapshotsStatus struct {
	Phase     SnapshotRequestPhase      `json:"phase,omitempty"`
	Snapshots map[string]SnapshotStatus `json:"snapshots,omitempty"`
	Error     SnapshotError             `json:"error,omitempty"`
}

// SnapshotStatus shows the current status of a single PVC snapshot.
type SnapshotStatus struct {
	Phase          SnapshotRequestPhase `json:"phase,omitempty"`
	SnapshotHandle string               `json:"snapshotHandle,omitempty"`
	Error          SnapshotError        `json:"error,omitempty"`
}

type SnapshotError struct {
	Message string `json:"message,omitempty"`
}

func (err SnapshotError) Equals(other SnapshotError) bool {
	return err.Message == other.Message
}

func (s SnapshotStatus) Equals(other SnapshotStatus) bool {
	return s.Phase == other.Phase &&
		s.SnapshotHandle == other.SnapshotHandle &&
		s.Error.Equals(other.Error)
}

func (s SnapshotStatus) Done() bool {
	return s.Phase == RequestPhaseCompleted || s.Phase == RequestPhaseSkipped || s.Phase == RequestPhaseFailed
}
