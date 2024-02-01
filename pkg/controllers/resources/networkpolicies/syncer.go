package networkpolicies

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"

	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &networkPolicySyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "networkpolicy", &networkingv1.NetworkPolicy{}),
	}, nil
}

type networkPolicySyncer struct {
	translator.NamespacedTranslator
}

var _ syncertypes.Syncer = &networkPolicySyncer{}

func (s *networkPolicySyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx.Context, vObj.(*networkingv1.NetworkPolicy)))
}

func (s *networkPolicySyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	newNetworkPolicy := s.translateUpdate(ctx.Context, pObj.(*networkingv1.NetworkPolicy), vObj.(*networkingv1.NetworkPolicy))
	if newNetworkPolicy != nil {
		translator.PrintChanges(pObj, newNetworkPolicy, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vObj, newNetworkPolicy)
}
