package volumes

import corev1 "k8s.io/api/core/v1"

const (
	SnapshotClassNameLabel         = "vcluster.loft.sh/csi-volumesnapshot-class"
	PersistentVolumeClaimNameLabel = "vcluster.loft.sh/csi-volumesnapshot-class"

	RequestPhaseNotStarted SnapshotRequestPhase = ""
	RequestPhaseInProgress SnapshotRequestPhase = "InProgress"
	RequestPhaseCleaningUp SnapshotRequestPhase = "CleaningUp"
	RequestPhaseCompleted  SnapshotRequestPhase = "Completed"
	RequestPhaseSkipped    SnapshotRequestPhase = "Skipped"
	RequestPhaseFailed     SnapshotRequestPhase = "Failed"
)

type SnapshotRequestPhase string

// SnapshotRequest is a request for creating PVC snapshots.
type SnapshotRequest struct {
	Spec   SnapshotRequestSpec   `json:"spec,omitempty"`
	Status SnapshotRequestStatus `json:"status,omitempty"`
}

// SnapshotRequestSpec specifies how to create PVC snapshots.
type SnapshotRequestSpec struct {
	VolumeSnapshotConfigs SnapshotConfigs `json:"volumeSnapshotConfigs,omitempty"`
}

// SnapshotRequestStatus shows the current status of the snapshot request.
type SnapshotRequestStatus struct {
	Phase     SnapshotRequestPhase `json:"phase,omitempty"`
	Snapshots Snapshots            `json:"snapshots,omitempty"`
}

// SnapshotConfigs specifies how to create snapshots for multiple PVCs.
type SnapshotConfigs []SnapshotConfig

// SnapshotConfig specifies how to create a snapshot for a PVC.
type SnapshotConfig struct {
	PersistentVolumeClaim corev1.PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// CSIDriver that provisioned the PVC.
	CSIDriver string `json:"csiDriver"`

	// VolumeSnapshotClassName to use when creating a VolumeSnapshot resource.
	VolumeSnapshotClassName string `json:"volumeSnapshotClassName,omitempty"`
}

// Snapshots is a map that specifies for which PVCs the snapshots have been successfully created.
type Snapshots map[string]SnapshotStatus

// SnapshotStatus shows the current status of a single PVC snapshot.
type SnapshotStatus struct {
	Phase          SnapshotRequestPhase `json:"phase,omitempty"`
	SnapshotHandle string               `json:"snapshotHandle,omitempty"`
}

func (s SnapshotStatus) Equals(other SnapshotStatus) bool {
	return s.Phase == other.Phase && s.SnapshotHandle == other.SnapshotHandle
}

func (s SnapshotStatus) Done() bool {
	return s.Phase == RequestPhaseCompleted || s.Phase == RequestPhaseSkipped || s.Phase == RequestPhaseFailed
}
