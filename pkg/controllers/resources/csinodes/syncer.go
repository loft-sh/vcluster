package csinodes

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &csinodeSyncer{
		Translator:    translator.NewMirrorPhysicalTranslator("csinode", &storagev1.CSINode{}),
		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type csinodeSyncer struct {
	translator.Translator
	virtualClient client.Client
}

var _ syncertypes.ToVirtualSyncer = &csinodeSyncer{}
var _ syncertypes.Syncer = &csinodeSyncer{}

func (s *csinodeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	// look up matching node name, don't sync if not found
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx.Context, types.NamespacedName{Name: pObj.GetName()}, node)
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}
	vObj := s.translateBackwards(ctx.Context, pObj.(*storagev1.CSINode))
	ctx.Log.Infof("create CSINode %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (s *csinodeSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// look up matching node name, delete csinode if not found
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx.Context, types.NamespacedName{Name: pObj.GetName()}, node)
	if kerrors.IsNotFound(err) {
		ctx.Log.Infof("delete virtual CSINode %s, because corresponding node object is missing", vObj.GetName())
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
	} else if err != nil {
		return ctrl.Result{}, err
	}
	// check if there is a change
	updated := s.translateUpdateBackwards(ctx.Context, pObj.(*storagev1.CSINode), vObj.(*storagev1.CSINode))
	if updated != nil {
		ctx.Log.Infof("update CSINode %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (s *csinodeSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSINode %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
