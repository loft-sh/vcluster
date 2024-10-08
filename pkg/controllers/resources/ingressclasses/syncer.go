package ingressclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(_ *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMirrorMapper(&networkingv1.IngressClass{})
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

var _ syncertypes.Syncer = &ingressClassSyncer{}

func (i *ingressClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(i)
}

func (i *ingressClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.IngressClass]) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, false)

	// Apply pro patches
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.IngressClasses.Patches)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create ingress class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (i *ingressClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.IngressClass]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.IngressClasses.Patches))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// cast objects
	event.Virtual.Annotations = event.Host.Annotations
	event.Virtual.Labels = event.Host.Labels
	event.Virtual.Spec.Controller = event.Host.Spec.Controller
	event.Virtual.Spec.Parameters = event.Host.Spec.Parameters
	return ctrl.Result{}, nil
}

func (i *ingressClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.IngressClass]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual ingress class %s, because physical object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
