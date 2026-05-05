package gateways

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
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewaySyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object          = &gatewaySyncer{}
	_ syncertypes.Syncer          = &gatewaySyncer{}
	_ syncertypes.OptionsProvider = &gatewaySyncer{}
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

func (s *gatewaySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	if s.skipSync(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.Gateways.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *gatewaySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.Gateway]) (_ ctrl.Result, retErr error) {
	if s.skipSync(ctx, event.Virtual) {
		return ctrl.Result{}, nil
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Gateways.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(
				event.Virtual,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
				"Error syncing: %v",
				retErr,
			)
		}
	}()

	event.Virtual.Spec.GatewayClassName, event.Host.Spec.GatewayClassName = patcher.CopyBidirectional(
		event.VirtualOld.Spec.GatewayClassName,
		event.Virtual.Spec.GatewayClassName,
		event.HostOld.Spec.GatewayClassName,
		event.Host.Spec.GatewayClassName,
	)

	event.Virtual.Status = event.Host.Status
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	hSpec, err := translateListeners(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to translate listeners: %w", err)
	}

	event.Host.Spec = *hSpec

	return ctrl.Result{}, nil
}

func (s *gatewaySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vGateway := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vGateway, event.Host, ctx.Config.Sync.ToHost.Gateways.Patches, false)
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
