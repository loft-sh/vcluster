package gateways

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/gatewaysync"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewaySyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object             = &gatewaySyncer{}
	_ syncertypes.Syncer             = &gatewaySyncer{}
	_ syncertypes.OptionsProvider    = &gatewaySyncer{}
	_ syncertypes.ControllerModifier = &gatewaySyncer{}
	_ syncertypes.IndicesRegisterer  = &gatewaySyncer{}
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Gateways())
	if err != nil {
		return nil, err
	}

	return &gatewaySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "gateway", &gatewayv1.Gateway{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

func (s *gatewaySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.Gateway](s)
}

func (s *gatewaySyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *gatewaySyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	if !ctx.Config.Sync.FromHost.GatewayClasses.Enabled {
		return nil
	}

	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &gatewayv1.Gateway{}, constants.IndexByGatewayClass, func(rawObj client.Object) []string {
		gateway, ok := rawObj.(*gatewayv1.Gateway)
		if !ok || gateway.Spec.GatewayClassName == "" {
			return nil
		}

		return []string{string(gateway.Spec.GatewayClassName)}
	})
}

func (s *gatewaySyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	var err error
	builder = gatewayauthz.RegisterGatewayWatches(ctx, builder)

	builder, err = routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Secrets())
	if err != nil {
		return nil, err
	}

	// Re-enqueue Gateways when their host GatewayClass changes only if from-host
	// GatewayClass sync is enabled — that is the only path through which a host
	// label change can flip skipSync. Without it, the watch produces work that
	// always falls through skipSync as no-op.
	if !ctx.Config.Sync.FromHost.GatewayClasses.Enabled {
		return builder, nil
	}

	return builder.WatchesRawSource(source.Kind(ctx.HostManager.GetCache(), &gatewayv1.GatewayClass{}, handler.TypedEnqueueRequestsFromMapFunc(func(mapCtx context.Context, gatewayClass *gatewayv1.GatewayClass) []reconcile.Request {
		if gatewayClass == nil {
			return nil
		}

		gatewayList := &gatewayv1.GatewayList{}
		err := ctx.VirtualManager.GetClient().List(mapCtx, gatewayList, client.MatchingFields{constants.IndexByGatewayClass: gatewayClass.Name})
		if err != nil {
			klog.FromContext(mapCtx).Error(err, "list virtual Gateways by host GatewayClass", "gatewayClass", gatewayClass.Name)
			return nil
		}

		requests := make([]reconcile.Request, 0, len(gatewayList.Items))
		for _, gateway := range gatewayList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: gateway.Namespace,
				Name:      gateway.Name,
			}})
		}

		klog.FromContext(mapCtx).V(5).Info("re-enqueued virtual Gateways for host GatewayClass change", "gatewayClass", gatewayClass.Name, "count", len(requests))

		return requests
	}))), nil
}

func (s *gatewaySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, gatewaysync.DeleteReason(event.Virtual))
	}

	if s.skipSync(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		if gatewayauthz.IsNotPermitted(err) {
			gatewaysync.RecordRefNotPermitted(s.EventRecorder(), event.Virtual, err)
			return ctrl.Result{}, nil
		}

		gatewaysync.RecordSyncError(s.EventRecorder(), event.Virtual, err)
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.GatewayAPI.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *gatewaySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.Gateway]) (_ ctrl.Result, retErr error) {
	if s.skipSync(ctx, event.Virtual) {
		s.EventRecorder().Eventf(
			event.Virtual,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
			"Deleting host Gateway %q because GatewayClass %q is no longer eligible for sync",
			event.Host.Name,
			event.Virtual.Spec.GatewayClassName,
		)
		return patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "gateway class is no longer eligible for sync")
	}

	hSpec, err := listenersToHost(ctx, event.Virtual, false)
	if err != nil {
		if gatewayauthz.IsNotPermitted(err) {
			gatewaysync.RecordRefNotPermitted(s.EventRecorder(), event.Virtual, err)
			return patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "virtual reference is not permitted")
		}

		gatewaysync.RecordSyncError(s.EventRecorder(), event.Virtual, err)
		return ctrl.Result{}, fmt.Errorf("failed to translate listeners: %w", err)
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.GatewayAPI.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	// Mutations until return are included in this deferred patch payload.
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			gatewaysync.RecordSyncError(s.EventRecorder(), event.Virtual, retErr)
		}
	}()

	// Resolve GatewayClassName bidirectionally. For stable drift, keep the host
	// aligned with the virtual Gateway because the tenant spec is the desired state.
	virtualGatewayClassChanged := event.VirtualOld.Spec.GatewayClassName != event.Virtual.Spec.GatewayClassName
	hostGatewayClassChanged := event.HostOld.Spec.GatewayClassName != event.Host.Spec.GatewayClassName
	event.Virtual.Spec.GatewayClassName, hSpec.GatewayClassName = patcher.CopyBidirectional(
		event.VirtualOld.Spec.GatewayClassName,
		event.Virtual.Spec.GatewayClassName,
		event.HostOld.Spec.GatewayClassName,
		event.Host.Spec.GatewayClassName,
	)
	if !virtualGatewayClassChanged && !hostGatewayClassChanged {
		hSpec.GatewayClassName = event.Virtual.Spec.GatewayClassName
	}

	// Host status is mirrored as reported by the Gateway controller. In
	// single-namespace mode the host spec may be constrained more narrowly than
	// the virtual spec, so route attachment counts reflect host-controller state.
	event.Virtual.Status = event.Host.Status
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Host.Spec = *hSpec

	return ctrl.Result{}, nil
}

func (s *gatewaySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vGateway := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vGateway, event.Host, ctx.Config.Sync.ToHost.GatewayAPI.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vGateway, s.EventRecorder(), true)
}

func (s *gatewaySyncer) skipSync(ctx *synccontext.SyncContext, gw *gatewayv1.Gateway) bool {
	if !ctx.Config.Sync.FromHost.GatewayClasses.Enabled ||
		ctx.Config.Sync.FromHost.GatewayClasses.Selector.Empty() ||
		gw.Spec.GatewayClassName == "" {
		return false
	}

	gwClass := &gatewayv1.GatewayClass{}
	err := ctx.HostClient.Get(ctx.Context, types.NamespacedName{Name: string(gw.Spec.GatewayClassName)}, gwClass)
	if err != nil || gwClass.GetDeletionTimestamp() != nil {
		s.EventRecorder().Eventf(
			gw, nil, "Warning", "SyncWarning", fmt.Sprintf("Sync%s", gw.GetObjectKind().GroupVersionKind().Kind),
			"The GatewayClass %q specified in Gateway %q could not be found on the host cluster: %v",
			gw.Spec.GatewayClassName,
			gw.GetName(),
			err,
		)

		return true
	}

	matches, err := ctx.Config.Sync.FromHost.GatewayClasses.Selector.Matches(gwClass)
	if err != nil {
		s.EventRecorder().Eventf(
			gw,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", gw.GetObjectKind().GroupVersionKind().Kind),
			"Gateway %q sync skipped. The GatewayClass %q in the host could not be checked against the selector under 'sync.fromHost.GatewayClasses.selector': %s",
			gw.GetName(),
			gwClass.GetName(),
			err,
		)
		return true
	}

	if !matches {
		s.EventRecorder().Eventf(
			gw,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", gw.GetObjectKind().GroupVersionKind().Kind),
			"Gateway %q sync skipped. The GatewayClass %q does not match the selector under 'sync.fromHost.GatewayClasses.selector'",
			gw.GetName(),
			gwClass.GetName(),
		)
		return true
	}

	return false
}
