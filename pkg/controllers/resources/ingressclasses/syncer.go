package ingressclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.IngressClasses())
	if err != nil {
		return nil, err
	}

	return &ingressClassSyncer{
		Mapper: mapper,
	}, nil
}

type ingressClassSyncer struct {
	synccontext.Mapper
}

func (i *ingressClassSyncer) Name() string {
	return "ingressclass"
}

func (i *ingressClassSyncer) Resource() client.Object {
	return &networkingv1.IngressClass{}
}

var _ syncertypes.ToVirtualSyncer = &ingressClassSyncer{}
var _ syncertypes.Syncer = &ingressClassSyncer{}

func (i *ingressClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(pObj.(*networkingv1.IngressClass), types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, false)
	ctx.Log.Infof("create ingress class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
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
	pIngressClass, vIngressClass, _, _ := synccontext.Cast[*networkingv1.IngressClass](ctx, pObj, vObj)
	vIngressClass.Annotations = pIngressClass.Annotations
	vIngressClass.Labels = pIngressClass.Labels
	vIngressClass.Spec.Controller = pIngressClass.Spec.Controller
	vIngressClass.Spec.Parameters = pIngressClass.Spec.Parameters
	return ctrl.Result{}, nil
}

func (i *ingressClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual ingress class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, vObj)
}
