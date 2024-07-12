package storageclasses

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &storageClassSyncer{
		Translator: translator.NewClusterTranslator(ctx, "storageclass", &storagev1.StorageClass{}, DefaultStorageClassAnnotation),
	}, nil
}

type storageClassSyncer struct {
	translator.Translator
}

var _ syncer.Syncer = &storageClassSyncer{}

func (s *storageClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// did the storage class change?
	updated := s.translateUpdate(ctx.Context, pObj.(*storagev1.StorageClass), vObj.(*storagev1.StorageClass))
	if updated != nil {
		ctx.Log.Infof("updating physical storage class %s, because virtual storage class has changed", updated.Name)
		translator.PrintChanges(pObj, updated, ctx.Log)
		err := ctx.PhysicalClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	newStorageClass := s.translate(ctx.Context, vObj.(*storagev1.StorageClass))
	ctx.Log.Infof("create physical storage class %s", newStorageClass.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, newStorageClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
