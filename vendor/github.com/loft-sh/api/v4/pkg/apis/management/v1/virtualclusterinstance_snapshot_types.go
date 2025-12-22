package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +subresource-request
type VirtualClusterInstanceSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status VirtualClusterInstanceSnapshotStatus `json:"status,omitempty"`
}

type VirtualClusterInstanceSnapshotStatus struct {
	SnapshotsTaken []SnapshotTaken `json:"snapshotTaken,omitempty"`
}

const (
	// SnapshotStored status will be deprecated in favor of SnapshotCompleted to match the vcluster snapshot status
	SnapshotStored          SnapshotTakenStatus = "Stored"
	SnapshotCompleted       SnapshotTakenStatus = "Completed"
	SnapshotScheduled       SnapshotTakenStatus = "Scheduled"
	SnapshotFailed          SnapshotTakenStatus = "Failed"
	SnapshotPartiallyFailed SnapshotTakenStatus = "PartiallyFailed"
	SnapshotInProgress      SnapshotTakenStatus = "InProgress"
	SnapshotNotFound        SnapshotTakenStatus = "NotFound"

	SnapshotRequestPhaseCompleted       SnapshotRequestPhase = "Completed"
	SnapshotRequestPhasePartiallyFailed SnapshotRequestPhase = "PartiallyFailed"
	SnapshotRequestPhaseFailed          SnapshotRequestPhase = "Failed"
)

type SnapshotTakenStatus string
type SnapshotRequestPhase string

// SnapshotTaken is the auto snapshot metadata of a snapshot taken from the vcluster.
type SnapshotTaken struct {
	Id        string              `json:"id,omitempty"`
	Url       string              `json:"url,omitempty"`
	Timestamp string              `json:"timestamp,omitempty"`
	Reason    string              `json:"reason,omitempty"`
	Request   SnapshotRequest     `json:"snapshotRequest,omitempty"`
	TotalPV   int                 `json:"totalPV"`
	Status    SnapshotTakenStatus `json:"status,omitempty"`
}

// SnapshotRequest is the request to take a snapshot from vcluster
// this struct is a copy from the vcluster snapshot request object

// SnapshotRequest is the request to take a snapshot of a volume.
type SnapshotRequest struct {
	Metadata SnapshotRequestMetadata `json:"metadata,omitempty"`
	Status   SnapshotRequestStatus   `json:"status"`
}

// SnapshotRequestMetadata is the metadata of the snapshot request.
type SnapshotRequestMetadata struct {
	Name              string      `json:"name"`
	CreationTimestamp metav1.Time `json:"creationTimestamp"`
}

// SnapshotRequestStatus shows the overall status of the snapshot request.
type SnapshotRequestStatus struct {
	Phase           SnapshotRequestPhase         `json:"phase,omitempty"`
	VolumeSnapshots VolumeSnapshotsRequestStatus `json:"volumeSnapshots"`
	Error           SnapshotRequestError         `json:"error,omitempty"`
}

// VolumeSnapshotsRequestStatus shows the current status of the snapshot request.
type VolumeSnapshotsRequestStatus struct {
	Phase     string                                 `json:"phase,omitempty"`
	Snapshots map[string]VolumeSnapshotRequestStatus `json:"snapshots,omitempty"`
	Error     SnapshotRequestError                   `json:"error"`
}

// SnapshotStatus shows the current status of a single PVC snapshot.
type VolumeSnapshotRequestStatus struct {
	Phase string               `json:"phase,omitempty"`
	Error SnapshotRequestError `json:"error"`
}

// SnapshotError describes the error that occurred while taking the snapshot.
type SnapshotRequestError struct {
	Message string `json:"message,omitempty"`
}
