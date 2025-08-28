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
	SnapshotStored     SnapshotTakenStatus = "Stored"
	SnapshotScheduled  SnapshotTakenStatus = "Scheduled"
	SnapshotFailed     SnapshotTakenStatus = "Failed"
	SnapshotInProgress SnapshotTakenStatus = "InProgress"
	SnapshotNotFound   SnapshotTakenStatus = "NotFound"
	SnapshotError      SnapshotTakenStatus = "Error"
)

type SnapshotTakenStatus string

type SnapshotTaken struct {
	Id        string              `json:"id,omitempty"`
	Url       string              `json:"url,omitempty"`
	Timestamp string              `json:"timestamp,omitempty"`
	Reason    string              `json:"reason,omitempty"`
	Status    SnapshotTakenStatus `json:"status,omitempty"`
}
