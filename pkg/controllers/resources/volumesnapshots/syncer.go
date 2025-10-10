package volumesnapshots

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	translator2 "github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.VolumeSnapshots())
	if err != nil {
		return nil, err
	}

	return &volumeSnapshotSyncer{
		GenericTranslator: translator2.NewGenericTranslator(ctx, "volume-snapshot", &volumesnapshotv1.VolumeSnapshot{}, mapper),
	}, nil
}

type volumeSnapshotSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.OptionsProvider = &volumeSnapshotSyncer{}

func (s *volumeSnapshotSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &volumeSnapshotSyncer{}

func (s *volumeSnapshotSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *volumeSnapshotSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*volumesnapshotv1.VolumeSnapshot]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		// delete volume snapshot immediately
		if len(event.Virtual.GetFinalizers()) > 0 || (event.Virtual.GetDeletionGracePeriodSeconds() != nil && *event.Virtual.GetDeletionGracePeriodSeconds() > 0) {
			event.Virtual.SetFinalizers([]string{})
			event.Virtual.SetDeletionGracePeriodSeconds(&zero)
			err := ctx.VirtualClient.Update(ctx, event.Virtual)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.VolumeSnapshots.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *volumeSnapshotSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*volumesnapshotv1.VolumeSnapshot]) (_ ctrl.Result, retErr error) {
	if event.Host.DeletionTimestamp != nil {
		if event.Virtual.DeletionTimestamp == nil {
			_, err := patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.Host, "physical volume snapshot is being deleted", &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
			if err != nil {
				return ctrl.Result{}, err
			}
		} else if *event.Virtual.DeletionGracePeriodSeconds != *event.Host.DeletionGracePeriodSeconds {
			_, err := patcher.DeleteVirtualObjectWithOptions(ctx, event.Virtual, event.Host, fmt.Sprintf("with grace period seconds %v", *event.Host.DeletionGracePeriodSeconds), &client.DeleteOptions{GracePeriodSeconds: event.Host.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(event.Virtual.UID))})
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// sync finalizers and status to allow tracking of the deletion progress
		// TODO: refactor finalizer syncing and handling
		// we can not add new finalizers from physical to virtual once it has deletionTimestamp, we can only remove finalizers

		if !equality.Semantic.DeepEqual(event.Virtual.Finalizers, event.Host.Finalizers) {
			updated := event.Virtual.DeepCopy()
			updated.Finalizers = event.Host.Finalizers
			ctx.Log.Infof("update finalizers of the virtual VolumeSnapshot %s, because finalizers on the physical resource changed", event.Virtual.Name)
			err := ctx.VirtualClient.Update(ctx, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if !equality.Semantic.DeepEqual(event.Virtual.Status, event.Host.Status) {
			updated := event.Virtual.DeepCopy()
			updated.Status = event.Host.Status.DeepCopy()
			ctx.Log.Infof("update virtual VolumeSnapshot %s, because status has changed", event.Virtual.Name)
			err := ctx.VirtualClient.Status().Update(ctx, updated)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if event.Virtual.DeletionTimestamp != nil {
		if event.Host.DeletionTimestamp == nil {
			return patcher.DeleteHostObjectWithOptions(ctx, event.Host, event.Virtual, "virtual volume snapshot is being deleted", &client.DeleteOptions{
				GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
			})
		}

		return ctrl.Result{}, nil
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.VolumeSnapshots.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}

		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// check backwards update
	event.Virtual.Finalizers = event.Host.Finalizers

	// check backwards status
	event.Virtual.Status = event.Host.Status.DeepCopy()

	// forward update
	event.Host.Spec.VolumeSnapshotClassName = event.Virtual.Spec.VolumeSnapshotClassName

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*volumesnapshotv1.VolumeSnapshot]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}
