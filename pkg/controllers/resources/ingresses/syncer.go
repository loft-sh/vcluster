package ingresses

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/selector"
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

	physicalClusterClient, err := kubernetes.NewForConfig(ctx.PhysicalManager.GetConfig())
	if err != nil {
		return nil, err
	}

	return &ingressSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "ingress", &networkingv1.Ingress{}, mapper),
		Importer:          pro.NewImporter(mapper),

		labelSelector:         ctx.Config.Sync.FromHost.IngressClasses.Selector,
		physicalClusterClient: physicalClusterClient,

		// exclude "field.cattle.io/publicEndpoints" annotation used by Rancher, similar to service syncer
		excludedAnnotations: []string{services.RancherPublicEndpointsAnnotation},
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	labelSelector         config.StandardLabelSelector
	physicalClusterClient kubernetes.Interface
	excludedAnnotations   []string
}

var _ syncertypes.OptionsProvider = &ingressSyncer{}

func (s *ingressSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*networkingv1.Ingress](s)
}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.Ingress]) (ctrl.Result, error) {
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

func (s *ingressSyncer) ExcludeVirtual(obj client.Object) bool {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok ||
		((ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName == "") && selector.IsLabelSelectorEmpty(s.labelSelector)) {
		return false
	}

	ingressClass, err := s.physicalClusterClient.NetworkingV1().IngressClasses().
		Get(context.Background(), *ingress.Spec.IngressClassName, metav1.GetOptions{})
	if err != nil {
		return true
	}

	return !selector.StandardLabelSelectorMatches(ingressClass, s.labelSelector)
}

func (s *ingressSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}
