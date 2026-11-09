package volumesnapshotclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.VolumeSnapshotClasses())
	if err != nil {
		return nil, err
	}

	return &volumeSnapshotClassSyncer{
		Mapper: mapper,
	}, nil
}

type volumeSnapshotClassSyncer struct {
	synccontext.Mapper
}

func (s *volumeSnapshotClassSyncer) Name() string {
	return "volumesnapshotclass"
}

func (s *volumeSnapshotClassSyncer) Resource() client.Object {
	return &volumesnapshotv1.VolumeSnapshotClass{}
}

var _ syncertypes.Syncer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *volumeSnapshotClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*volumesnapshotv1.VolumeSnapshotClass]) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name}, false)

	// Apply pro patches
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.VolumeSnapshotClasses.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying pro patches: %w", err)
	}

	ctx.Log.Infof("create VolumeSnapshotClass %s, because it does not exist in the virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *volumeSnapshotClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*volumesnapshotv1.VolumeSnapshotClass]) (ctrl.Result, error) {
	// We are not doing any syncing Forward for the VolumeSnapshotClasses
	// if this method is called it means that VolumeSnapshotClass was deleted in host or
	// a new VolumeSnapshotClass was created in vcluster, and it should be deleted to avoid confusion
	ctx.Log.Infof("delete VolumeSnapshotClass %s, because it does not exist in the host cluster", event.Virtual.Name)
	err := ctx.VirtualClient.Delete(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (s *volumeSnapshotClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*volumesnapshotv1.VolumeSnapshotClass]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.VolumeSnapshotClasses.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	event.Virtual.Annotations = event.Host.Annotations
	event.Virtual.Labels = event.Host.Labels
	event.Virtual.Driver = event.Host.Driver
	event.Virtual.Parameters = event.Host.Parameters
	event.Virtual.DeletionPolicy = event.Host.DeletionPolicy
	return ctrl.Result{}, nil
}
