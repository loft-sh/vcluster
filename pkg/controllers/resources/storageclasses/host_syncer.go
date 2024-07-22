package storageclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewHostStorageClassSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.StorageClasses())
	if err != nil {
		return nil, err
	}

	return &hostStorageClassSyncer{
		Mapper: mapper,
	}, nil
}

type hostStorageClassSyncer struct {
	synccontext.Mapper
}

func (s *hostStorageClassSyncer) Name() string {
	return "host-storageclass"
}

func (s *hostStorageClassSyncer) Resource() client.Object {
	return &storagev1.StorageClass{}
}

var _ syncertypes.ToVirtualSyncer = &hostStorageClassSyncer{}

func (s *hostStorageClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(pObj.(*storagev1.StorageClass), types.NamespacedName{Name: pObj.GetName()}, false)
	ctx.Log.Infof("create storage class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

var _ syncertypes.Syncer = &hostStorageClassSyncer{}

func (s *hostStorageClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// check if there is a change
	pSC, vSC, _, _ := synccontext.Cast[*storagev1.StorageClass](ctx, pObj, vObj)
	vSC.Annotations = pSC.Annotations
	vSC.Labels = pSC.Labels
	vSC.Provisioner = pSC.Provisioner
	vSC.Parameters = pSC.Parameters
	vSC.ReclaimPolicy = pSC.ReclaimPolicy
	vSC.MountOptions = pSC.MountOptions
	vSC.AllowVolumeExpansion = pSC.AllowVolumeExpansion
	vSC.VolumeBindingMode = pSC.VolumeBindingMode
	vSC.AllowedTopologies = pSC.AllowedTopologies
	return ctrl.Result{}, nil
}

func (s *hostStorageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual storage class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, vObj)
}
