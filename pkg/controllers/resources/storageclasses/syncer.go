package storageclasses

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &storageClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "storageclass", &storagev1.StorageClass{}, mappings.StorageClasses(), DefaultStorageClassAnnotation),
	}, nil
}

type storageClassSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &storageClassSyncer{}

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	newStorageClass := s.translate(ctx, vObj.(*storagev1.StorageClass))
	ctx.Log.Infof("create physical storage class %s", newStorageClass.Name)
	err := ctx.PhysicalClient.Create(ctx, newStorageClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// did the storage class change?
	updated := s.translateUpdate(ctx, pObj.(*storagev1.StorageClass), vObj.(*storagev1.StorageClass))
	if updated != nil {
		ctx.Log.Infof("updating physical storage class %s, because virtual storage class has changed", updated.Name)
		translator.PrintChanges(pObj, updated, ctx.Log)
		err := ctx.PhysicalClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
