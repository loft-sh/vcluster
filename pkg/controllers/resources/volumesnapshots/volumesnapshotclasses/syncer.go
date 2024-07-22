package volumesnapshotclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
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

var _ syncertypes.ToVirtualSyncer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(pObj.(*volumesnapshotv1.VolumeSnapshotClass), types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, false)
	ctx.Log.Infof("create VolumeSnapshotClass %s, because it does not exist in the virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

var _ syncertypes.Syncer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// We are not doing any syncing Forward for the VolumeSnapshotClasses
	// if this method is called it means that VolumeSnapshotClass was deleted in host or
	// a new VolumeSnapshotClass was created in vcluster, and it should be deleted to avoid confusion
	ctx.Log.Infof("delete VolumeSnapshotClass %s, because it does not exist in the host cluster", vObj.GetName())
	err := ctx.VirtualClient.Delete(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (s *volumeSnapshotClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	pVSC, vVSC, _, _ := synccontext.Cast[*volumesnapshotv1.VolumeSnapshotClass](ctx, pObj, vObj)
	vVSC.Annotations = pVSC.Annotations
	vVSC.Labels = pVSC.Labels
	vVSC.Driver = pVSC.Driver
	vVSC.Parameters = pVSC.Parameters
	vVSC.DeletionPolicy = pVSC.DeletionPolicy
	return ctrl.Result{}, nil
}
