package ingresses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Ingresses())
	if err != nil {
		return nil, err
	}

	return &ingressSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "ingress", &networkingv1.Ingress{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*networkingv1.Ingress](s)
}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.Ingress]) (ctrl.Result, error) {
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Ingresses.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Ingresses.Translate))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	event.TargetObject().Spec.IngressClassName = event.SourceObject().Spec.IngressClassName
	event.Virtual.Status = event.Host.Status
	s.translateUpdate(ctx, event.Host, event.Virtual)
	return ctrl.Result{}, nil
}

func (s *ingressSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vIngress := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vIngress, event.Host, ctx.Config.Sync.ToHost.Ingresses.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateVirtualObject(ctx, event.Host, vIngress, s.EventRecorder())
}
