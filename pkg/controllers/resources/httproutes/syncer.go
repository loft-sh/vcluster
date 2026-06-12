package httproutes

import (
	"fmt"

	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/gatewaysync"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
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
	builder = gatewayauthz.RegisterHTTPRouteWatches(ctx, builder)
	return routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Gateways(), mappings.Services())
}

func (s *httpRouteSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.HTTPRoute]) (ctrl.Result, error) {
	return gatewaysync.CreateToHost(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Patches, func() (*gatewayv1.HTTPRoute, error) {
		return s.translate(ctx, event.Virtual)
	})
}

func (s *httpRouteSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.HTTPRoute]) (ctrl.Result, error) {
	var hSpec *gatewayv1.HTTPRouteSpec
	return gatewaysync.Sync(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Patches,
		func() (err error) {
			hSpec, err = specToHost(ctx, event.Virtual, false)
			return err
		},
		func() error {
			// Status translation is independent of spec sync; on failure keep applying
			// the spec but surface the error so the route is requeued and status retried.
			vStatus, statusErr := statusToVirtual(ctx, event.Host, event.Virtual, event.Host.Status)
			if statusErr == nil {
				event.Virtual.Status = vStatus
			}

			// Preserve any host-only managed rule first so it lands at hSpec.Rules[0]
			// before preserveRequestMirrorFilters runs; that way name-based correlation
			// keeps any mirror filter on the managed rule without depending on positional
			// fallback.
			preserveHostRule(event.Host.Spec, hSpec, event.Host.Annotations)
			preserveRequestMirrorFilters(event.Host.Spec, hSpec, event.Host.Annotations)
			event.Host.Spec = *hSpec

			if statusErr != nil {
				return fmt.Errorf("translate status: %w", statusErr)
			}
			return nil
		},
	)
}

func (s *httpRouteSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.HTTPRoute]) (ctrl.Result, error) {
	return gatewaysync.CreateToVirtual(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Patches, func() *gatewayv1.HTTPRoute {
		return translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	})
}
