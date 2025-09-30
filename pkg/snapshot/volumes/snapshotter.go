package volumes

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// ErrPersistentVolumeNotSupported is an error that indicates that the snapshotter does not support
	// creating snapshots for the specified PersistentVolume.
	ErrPersistentVolumeNotSupported = errors.New("PersistentVolume is not supported by the snapshotter")
)

type PersistentVolumeReference struct {
	PersistentVolumeClaim types.NamespacedName
	PersistentVolumeName  string
}

type CreateSnapshotsResult struct {
	SnapshottedPersistentVolumes []PersistentVolumeReference
}

// Snapshotter creates and restores persistent volume snapshots.
type Snapshotter interface {
	// CheckIfPersistentVolumeIsSupported checks if the snapshotter can create a volume snapshot of
	// the specified persistent volume.
	//
	//   - If it can create snapshots for the specified persistent volume, then this function returns
	//     nil.
	//
	//   - If the persistent volume is not supported by definition (e.g. CSI snapshotter cannot create
	//   snapshots for volumes that are not handled by CSI drivers), the function returns
	//   ErrPersistentVolumeNotSupported (or an error that wraps ErrPersistentVolumeNotSupported).
	CheckIfPersistentVolumeIsSupported(pv *corev1.PersistentVolume) error

	// Reconcile volume snapshots request.
	Reconcile(ctx context.Context, requestObj runtime.Object, requestName string, spec *SnapshotsRequest, status *SnapshotsStatus) error

	// Cleanup does any necessary clean up of the cluster after taking the snapshot of the volumes.
	// E.g. it can remove all the resources that were created by the snapshotter in order to create
	// volume snapshots.
	Cleanup(ctx context.Context) error
}
