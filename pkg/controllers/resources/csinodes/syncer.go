package csinodes

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &csinodeSyncer{
		Translator:     translator.NewMirrorPhysicalTranslator("csinode", &storagev1.CSINode{}),
		physicalClient: ctx.PhysicalManager.GetClient(),
	}, nil
}

type csinodeSyncer struct {
	translator.Translator
	physicalClient client.Client
}

var _ syncer.UpSyncer = &csinodeSyncer{}
var _ syncer.Syncer = &csinodeSyncer{}

// TODO, only sync selected nodes
// look up matching node name, don't enqueue if not synced

func (s *csinodeSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	node := &corev1.Node{}
	s.physicalClient.Get(ctx.Context, types)
	vObj := s.translateBackwards(pObj.(*storagev1.CSINode))
	ctx.Log.Infof("create CSINode %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (s *csinodeSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if there is a change
	updated := s.translateUpdateBackwards(pObj.(*storagev1.CSINode), vObj.(*storagev1.CSINode))
	if updated != nil {
		ctx.Log.Infof("update CSINode %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (s *csinodeSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSINode %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
