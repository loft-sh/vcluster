package ingressclasses

import (
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/patcher"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	networkingv1 "k8s.io/api/networking/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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
	vObj := i.createVirtual(ctx.Context, pObj.(*networkingv1.IngressClass))
	ctx.Log.Infof("create ingress class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (i *ingressClassSyncer) Sync(ctx *synccontext.SyncContext, pObj, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// cast objects
	pNamespace, vNamespace, _, _ := synccontext.Cast[*networkingv1.IngressClass](ctx, pObj, vObj)

	updated := i.updateVirtual(ctx.Context, pNamespace, vNamespace)
	if updated != nil {
		ctx.Log.Infof("update ingress class %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		vObj = updated
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (i *ingressClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual ingress class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}
