package csidrivers

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(_ *synccontext.RegisterContext) (syncer.Object, error) {
	return &csidriverSyncer{
		Translator: translator.NewMirrorPhysicalTranslator("csidriver", &storagev1.CSIDriver{}),
	}, nil
}

type csidriverSyncer struct {
	translator.Translator
}

var _ syncer.ToVirtualSyncer = &csidriverSyncer{}
var _ syncer.Syncer = &csidriverSyncer{}

func (s *csidriverSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := s.translateBackwards(ctx.Context, pObj.(*storagev1.CSIDriver))
	ctx.Log.Infof("create CSIDriver %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (s *csidriverSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if there is a change
	updated := s.translateUpdateBackwards(ctx.Context, pObj.(*storagev1.CSIDriver), vObj.(*storagev1.CSIDriver))
	if updated != nil {
		ctx.Log.Infof("update CSIDriver %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (s *csidriverSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSIDriver %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
