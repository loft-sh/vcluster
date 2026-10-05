package referencegrants

import (
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

type referenceGrantSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object             = &referenceGrantSyncer{}
	_ syncertypes.Syncer             = &referenceGrantSyncer{}
	_ syncertypes.OptionsProvider    = &referenceGrantSyncer{}
	_ syncertypes.ControllerModifier = &referenceGrantSyncer{}
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.ReferenceGrants())
	if err != nil {
		return nil, err
	}

	return &referenceGrantSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "referencegrant", &gatewayv1.ReferenceGrant{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

func (s *referenceGrantSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.ReferenceGrant](s)
}

func (s *referenceGrantSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *referenceGrantSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Services(), mappings.Secrets(), mappings.ConfigMaps())
}

func (s *referenceGrantSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.ReferenceGrant]) (ctrl.Result, error) {
	return gatewaysync.CreateToHost(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Patches, func() (*gatewayv1.ReferenceGrant, error) {
		return s.translate(ctx, event.Virtual)
	})
}

func (s *referenceGrantSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.ReferenceGrant]) (ctrl.Result, error) {
	var hSpec *gatewayv1.ReferenceGrantSpec
	return gatewaysync.Sync(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Patches,
		func() (err error) {
			hSpec, err = specToHost(ctx, event.Virtual, false)
			return err
		},
		func() error {
			event.Host.Spec = *hSpec
			return nil
		},
	)
}

func (s *referenceGrantSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.ReferenceGrant]) (ctrl.Result, error) {
	return gatewaysync.CreateToVirtual(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Patches, func() *gatewayv1.ReferenceGrant {
		return translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	})
}
