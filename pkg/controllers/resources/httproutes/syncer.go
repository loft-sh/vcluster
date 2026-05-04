package httproutes

import (
	"fmt"

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type httpRouteSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object             = &httpRouteSyncer{}
	_ syncertypes.Syncer             = &httpRouteSyncer{}
	_ syncertypes.OptionsProvider    = &httpRouteSyncer{}
	_ syncertypes.ControllerModifier = &httpRouteSyncer{}
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.HTTPRoutes())
	if err != nil {
		return nil, err
	}

	return &httpRouteSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "httproute", &gatewayv1.HTTPRoute{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

func (s *httpRouteSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.HTTPRoute](s)
}

func (s *httpRouteSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *httpRouteSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Gateways(), mappings.Services())
}

func (s *httpRouteSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.HTTPRoute]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, httpRouteDeleteReason(event.Virtual))
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutePatches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *httpRouteSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.HTTPRoute]) (_ ctrl.Result, retErr error) {
	hSpec, err := specToHost(ctx, event.Virtual, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to translate spec: %w", err)
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutePatches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	// Mutations until return are included in this deferred patch payload.
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

	vStatus, err := statusToVirtual(ctx, event.Host, event.Virtual.Namespace, event.Host.Status)
	if err != nil {
		s.EventRecorder().Eventf(
			event.Virtual,
			nil,
			"Warning",
			"SyncWarning",
			fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
			"Error translating status: %v",
			err,
		)
	} else {
		event.Virtual.Status = vStatus
	}
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Host.Spec = *hSpec

	return ctrl.Result{}, retErr
}

func (s *httpRouteSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.HTTPRoute]) (ctrl.Result, error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vRoute := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vRoute, event.Host, ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutePatches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vRoute, s.EventRecorder(), true)
}

func httpRouteDeleteReason(route *gatewayv1.HTTPRoute) string {
	if route != nil && route.DeletionTimestamp != nil {
		return "virtual object was deleted by user"
	}

	return "host object was deleted"
}
