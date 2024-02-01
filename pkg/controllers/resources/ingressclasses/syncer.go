package ingressclasses

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(*synccontext.RegisterContext) (syncer.Object, error) {
	return &ingressClassSyncer{
		Translator: translator.NewMirrorPhysicalTranslator("ingressclass", &networkingv1.IngressClass{}),
	}, nil
}

type ingressClassSyncer struct {
	translator.Translator
}

var _ syncer.ToVirtualSyncer = &ingressClassSyncer{}
var _ syncer.Syncer = &ingressClassSyncer{}

func (i *ingressClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := i.translateBackwards(ctx.Context, pObj.(*networkingv1.IngressClass))
	ctx.Log.Infof("create ingress class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (i *ingressClassSyncer) Sync(ctx *synccontext.SyncContext, pObj, vObj client.Object) (ctrl.Result, error) {
	updated := i.translateUpdateBackwards(ctx.Context, pObj.(*networkingv1.IngressClass), vObj.(*networkingv1.IngressClass))
	if updated != nil {
		ctx.Log.Infof("update ingress class %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (i *ingressClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual ingress class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
