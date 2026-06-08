package tlsroutes

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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type tlsRouteSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object             = &tlsRouteSyncer{}
	_ syncertypes.Syncer             = &tlsRouteSyncer{}
	_ syncertypes.OptionsProvider    = &tlsRouteSyncer{}
	_ syncertypes.ControllerModifier = &tlsRouteSyncer{}
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.TLSRoutes())
	if err != nil {
		return nil, err
	}

	return &tlsRouteSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "tlsroute", &gatewayv1alpha2.TLSRoute{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

func (s *tlsRouteSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1alpha2.TLSRoute](s)
}

func (s *tlsRouteSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *tlsRouteSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	builder = gatewayauthz.RegisterTLSRouteWatches(ctx, builder)
	return routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Gateways(), mappings.Services())
}

func (s *tlsRouteSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1alpha2.TLSRoute]) (ctrl.Result, error) {
	return gatewaysync.CreateToHost(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.TLSRoutes.Patches, func() (*gatewayv1alpha2.TLSRoute, error) {
		return s.translate(ctx, event.Virtual)
	})
}

func (s *tlsRouteSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1alpha2.TLSRoute]) (ctrl.Result, error) {
	var hSpec *gatewayv1alpha2.TLSRouteSpec
	return gatewaysync.Sync(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.TLSRoutes.Patches,
		func() (err error) {
			hSpec, err = specToHost(ctx, event.Virtual, false)
			return err
		},
		func() error {
			// Status translation is independent of spec sync; on failure keep applying
			// the spec but surface the error so the route is requeued and status retried.
			vStatus, statusErr := statusToVirtual(ctx, event.Host, event.Virtual.Namespace, event.Host.Status)
			if statusErr == nil {
				event.Virtual.Status = vStatus
			}
			event.Host.Spec = *hSpec

			if statusErr != nil {
				return fmt.Errorf("translate status: %w", statusErr)
			}
			return nil
		},
	)
}

func (s *tlsRouteSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1alpha2.TLSRoute]) (ctrl.Result, error) {
	return gatewaysync.CreateToVirtual(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.TLSRoutes.Patches, func() *gatewayv1alpha2.TLSRoute {
		return translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	})
}
