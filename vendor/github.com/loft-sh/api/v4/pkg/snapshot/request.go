package snapshot

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type LongRunningRequest interface {
	GetPhase() RequestPhase
}

const (
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
	switch r {
	case RequestPhaseCreatingVolumeSnapshots:
		return RequestPhaseCreatingEtcdBackup
	case RequestPhaseCreatingEtcdBackup:
		return RequestPhaseCompleted
	case RequestPhaseCanceling:
		return RequestPhaseCanceled
	case RequestPhaseDeletingVolumeSnapshots:
		return RequestPhaseDeletingEtcdBackup
	case RequestPhaseDeletingEtcdBackup:
		return RequestPhaseDeleted
	default:
		return RequestPhaseUnknown
	}
}

type RequestMetadata struct {
	Name              string      `json:"name"`
	CreationTimestamp metav1.Time `json:"creationTimestamp,omitempty"`
}

type Request struct {
	RequestMetadata `json:"metadata,omitempty"`
	Spec            RequestSpec   `json:"spec,omitempty"`
	Status          RequestStatus `json:"status,omitempty"`
}

func (r *Request) Done() bool {
	return r.Status.Phase == RequestPhaseCompleted ||
		r.Status.Phase == RequestPhasePartiallyFailed ||
		r.Status.Phase == RequestPhaseFailed ||
		r.Status.Phase == RequestPhaseCanceled
}

func (r *Request) GetPhase() RequestPhase {
	return r.Status.Phase
}

func (r *Request) ShouldCancel(otherRequest *Request) bool {
	if otherRequest.Name == r.Name {
		return false
	}
	if otherRequest.Spec.URL != r.Spec.URL {
		return false
	}
	if otherRequest.CreationTimestamp.Time.After(r.CreationTimestamp.Time) {
		return false
	}
	return otherRequest.Status.Phase == RequestPhaseNotStarted ||
		otherRequest.Status.Phase == RequestPhaseCreatingVolumeSnapshots ||
		otherRequest.Status.Phase == RequestPhaseCreatingEtcdBackup
}

type RequestSpec struct {
	URL             string                 `json:"url,omitempty"`
	IncludeVolumes  bool                   `json:"includeVolumes,omitempty"`
	VolumeSnapshots VolumeSnapshotsRequest `json:"volumeSnapshots,omitempty"`
	Options         Options                `json:"-"`
}

type RequestStatus struct {
	Phase           RequestPhase          `json:"phase,omitempty"`
	VolumeSnapshots VolumeSnapshotsStatus `json:"volumeSnapshots,omitempty"`
	Error           SnapshotError         `json:"error,omitempty"`
}
