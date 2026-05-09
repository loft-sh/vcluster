package backendtlspolicies

import (
	"fmt"

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

type backendTLSPolicySyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var (
	_ syncertypes.Object             = &backendTLSPolicySyncer{}
	_ syncertypes.Syncer             = &backendTLSPolicySyncer{}
	_ syncertypes.OptionsProvider    = &backendTLSPolicySyncer{}
	_ syncertypes.ControllerModifier = &backendTLSPolicySyncer{}
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.BackendTLSPolicies())
	if err != nil {
		return nil, err
	}

	return &backendTLSPolicySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "backendtlspolicy", &gatewayv1.BackendTLSPolicy{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

func (s *backendTLSPolicySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.BackendTLSPolicy](s)
}

func (s *backendTLSPolicySyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

func (s *backendTLSPolicySyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return routetranslate.RegisterReferencedWatches(ctx, builder, s.GroupVersionKind(), mappings.Services(), mappings.ConfigMaps(), mappings.Secrets())
}

func (s *backendTLSPolicySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.BackendTLSPolicy]) (ctrl.Result, error) {
	return gatewaysync.CreateToHost(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.BackendTLSPolicyPatches, func() (*gatewayv1.BackendTLSPolicy, error) {
		return s.translate(ctx, event.Virtual)
	})
}

func (s *backendTLSPolicySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.BackendTLSPolicy]) (ctrl.Result, error) {
	var hSpec *gatewayv1.BackendTLSPolicySpec
	return gatewaysync.Sync(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.BackendTLSPolicyPatches,
		func() (err error) {
			hSpec, err = translateSpecToHost(ctx, event.Virtual, false)
			return err
		},
		func() error {
			// Status translation is independent of spec sync; on failure keep applying
			// the spec but surface the error so the policy is requeued and status retried.
			vStatus, statusErr := translateStatusToVirtual(ctx, event.Host, event.Virtual.Namespace, event.Host.Status)
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

func (s *backendTLSPolicySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.BackendTLSPolicy]) (ctrl.Result, error) {
	return gatewaysync.CreateToVirtual(ctx, event, s.EventRecorder(), ctx.Config.Sync.ToHost.GatewayAPI.BackendTLSPolicyPatches, func() *gatewayv1.BackendTLSPolicy {
		return translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	})
}
