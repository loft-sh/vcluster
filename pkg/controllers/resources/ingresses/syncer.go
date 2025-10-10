package ingresses

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
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

		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher, similar to service syncer
		excludedAnnotations: []string{services.RancherPublicEndpointsAnnotation},
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	excludedAnnotations []string
}

var _ syncertypes.OptionsProvider = &ingressSyncer{}

func (s *ingressSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.Ingress]) (ctrl.Result, error) {
	if s.applyLimitByClass(ctx, event.Virtual) {
		s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncWarning", "did not sync ingress %q to host because it does not match the selector under 'sync.fromHost.ingressClasses.selector'", event.Virtual.GetName())
		return ctrl.Result{}, nil
	}

	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Ingresses.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	if s.applyLimitByClass(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Ingresses.Patches, false))
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

	event.Virtual.Spec.IngressClassName, event.Host.Spec.IngressClassName = patcher.CopyBidirectional(
		event.VirtualOld.Spec.IngressClassName,
		event.Virtual.Spec.IngressClassName,
		event.HostOld.Spec.IngressClassName,
		event.Host.Spec.IngressClassName,
	)
	event.Virtual.Status = event.Host.Status
	s.translateUpdate(ctx, event)
	return ctrl.Result{}, nil
}

func (s *ingressSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vIngress := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host), s.excludedAnnotations...)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vIngress, event.Host, ctx.Config.Sync.ToHost.Ingresses.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vIngress, s.EventRecorder(), true)
}

func (s *ingressSyncer) applyLimitByClass(ctx *synccontext.SyncContext, virtual *networkingv1.Ingress) bool {
	if !ctx.Config.Sync.FromHost.IngressClasses.Enabled ||
		ctx.Config.Sync.FromHost.IngressClasses.Selector.Empty() ||
		virtual.Spec.IngressClassName == nil ||
		*virtual.Spec.IngressClassName == "" {
		return false
	}

	pIngressClass := &networkingv1.IngressClass{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{Name: *virtual.Spec.IngressClassName}, pIngressClass)
	if err != nil || pIngressClass.GetDeletionTimestamp() != nil {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync ingress %q to host because the ingress class %q couldn't be reached in the host: %s", virtual.GetName(), *virtual.Spec.IngressClassName, err)
		return true
	}
	matches, err := ctx.Config.Sync.FromHost.IngressClasses.Selector.Matches(pIngressClass)
	if err != nil {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync ingress %q to host because the ingress class %q in the host could not be checked against the selector under 'sync.fromHost.ingressClasses.selector': %s", virtual.GetName(), pIngressClass.GetName(), err)
		return true
	}
	if !matches {
		s.EventRecorder().Eventf(virtual, "Warning", "SyncWarning", "did not sync ingress %q to host because the ingress class %q in the host does not match the selector under 'sync.fromHost.ingressClasses.selector'", virtual.GetName(), pIngressClass.GetName())
		return true
	}

	return false
}
