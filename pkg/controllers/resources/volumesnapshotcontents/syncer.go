package volumesnapshotcontents

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	translator2 "github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PhysicalVSCGarbageCollectionFinalizer = "vcluster.loft.sh/physical-volumesnapshotcontent-gc"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.VolumeSnapshotContents())
	if err != nil {
		return nil, err
	}

	return &volumeSnapshotContentSyncer{
		GenericTranslator: translator2.NewGenericTranslator(ctx, "volume-snapshot-content", &volumesnapshotv1.VolumeSnapshotContent{}, mapper),

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type volumeSnapshotContentSyncer struct {
	syncertypes.GenericTranslator

	virtualClient client.Client
}

var _ syncertypes.OptionsProvider = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &volumeSnapshotContentSyncer{}

func (s *volumeSnapshotContentSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *volumeSnapshotContentSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*volumesnapshotv1.VolumeSnapshotContent]) (ctrl.Result, error) {
	// check if the VolumeSnapshotContent should get synced
	sync, vVS, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore this VolumeSnapshotContent resource, because there is no virtual VolumeSnapshot bound to it
		return ctrl.Result{}, nil
	}

	vVSC := s.translateBackwards(event.Host, vVS)
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vVSC, event.Host, ctx.Config.Sync.ToHost.VolumeSnapshotContents.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vVSC, nil, true)
}

func (s *volumeSnapshotContentSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*volumesnapshotv1.VolumeSnapshotContent]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil || (event.Virtual.Annotations != nil && event.Virtual.Annotations[constants.HostClusterVSCAnnotation] != "") {
		if len(event.Virtual.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			event.Virtual.Finalizers = []string{}
			ctx.Log.Infof("remove virtual VolumeSnapshotContent %s finalizers, because object should get deleted", event.Virtual.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx, event.Virtual)
		}

		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "object should get deleted")
	}

	pVSC := s.translate(ctx, event.Virtual)
	err := pro.ApplyPatchesHostObject(ctx, nil, pVSC, event.Virtual, ctx.Config.Sync.ToHost.VolumeSnapshotContents.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pVSC, nil, true)
}

func (s *volumeSnapshotContentSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*volumesnapshotv1.VolumeSnapshotContent]) (_ ctrl.Result, retErr error) {
	// check if objects are getting deleted
	if event.Virtual.GetDeletionTimestamp() != nil {
		if event.Host.GetDeletionTimestamp() == nil {
			_, err := patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "virtual VolumeSnapshotContent is being deleted")
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
			ctx.Log.Infof("update finalizers of the virtual VolumeSnapshotContent %s, because finalizers on the physical resource changed", event.Virtual.Name)
			err := s.virtualClient.Update(ctx, updated)
			if kerrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if !equality.Semantic.DeepEqual(event.Virtual.Status, event.Host.Status) {
			updated := event.Virtual.DeepCopy()
			updated.Status = event.Host.Status.DeepCopy()
			ctx.Log.Infof("update virtual VolumeSnapshotContent %s, because status has changed", event.Virtual.Name)
			err := s.virtualClient.Status().Update(ctx, updated)
			if err != nil && !kerrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the VolumeSnapshotContent should get synced
	sync, _, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		// ignore VolumeSnapshotContent object for which VolumeSnapshot was deleted,
		// it will be automatically managed by the snapshot controller based on deletion policy
		return ctrl.Result{}, nil
	}

	// update the physical VolumeSnapshotContent if the virtual has changed
	if event.Virtual.Annotations[constants.HostClusterVSCAnnotation] == "" && event.Virtual.DeletionTimestamp != nil {
		if event.Host.DeletionTimestamp != nil {
			return ctrl.Result{}, nil
		}

		return patcher.DeleteHostObjectWithOptions(ctx, event.Host, event.Virtual, "virtual VolumeSnapshotContent is being deleted", &client.DeleteOptions{
			GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
		})
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.VolumeSnapshotContents.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// update virtual object
	s.translateUpdateBackwards(event.Host, event.Virtual)

	// update virtual status
	event.Virtual.Status = event.Host.Status.DeepCopy()

	// update host object
	if event.Virtual.Annotations[constants.HostClusterVSCAnnotation] == "" {
		event.Host.Spec.DeletionPolicy = event.Virtual.Spec.DeletionPolicy
		event.Host.Spec.VolumeSnapshotClassName = event.Virtual.Spec.VolumeSnapshotClassName
	}

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *volumeSnapshotContentSyncer) shouldSync(ctx *synccontext.SyncContext, pObj *volumesnapshotv1.VolumeSnapshotContent) (bool, *volumesnapshotv1.VolumeSnapshot, error) {
	vName := mappings.HostToVirtual(ctx, pObj.Spec.VolumeSnapshotRef.Name, pObj.Spec.VolumeSnapshotRef.Namespace, nil, mappings.VolumeSnapshots())
	if vName.Name == "" {
		return false, nil, nil
	}

	vVS := &volumesnapshotv1.VolumeSnapshot{}
	err := s.virtualClient.Get(ctx, vName, vVS)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.Default.IsManaged(ctx, pObj) {
			return true, vVS, nil
		}
		return false, nil, nil
	}

	return true, vVS, nil
}

func (s *volumeSnapshotContentSyncer) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	pVSC, ok := pObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok {
		return false, nil
	}

	sync, _, err := s.shouldSync(ctx, pVSC)
	if err != nil {
		return false, nil
	}

	return sync, nil
}
