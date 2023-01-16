package storageclasses

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewHostStorageClassSyncer(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &hostStorageClassSyncer{
		Translator: translator.NewMirrorPhysicalTranslator("host-storageclass", &storagev1.StorageClass{}),
	}, nil
}

type hostStorageClassSyncer struct {
	translator.Translator
}

var _ syncer.UpSyncer = &hostStorageClassSyncer{}

func (s *hostStorageClassSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := s.translateBackwards(pObj.(*storagev1.StorageClass))
	ctx.Log.Infof("create storage class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

var _ syncer.Syncer = &hostStorageClassSyncer{}

func (s *hostStorageClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if there is a change
	updated := s.translateUpdateBackwards(pObj.(*storagev1.StorageClass), vObj.(*storagev1.StorageClass))
	if updated != nil {
		ctx.Log.Infof("update storage class %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (s *hostStorageClassSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual storage class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
