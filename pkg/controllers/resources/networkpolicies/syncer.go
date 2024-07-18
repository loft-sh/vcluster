package networkpolicies

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &networkPolicySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "networkpolicy", &networkingv1.NetworkPolicy{}, mappings.NetworkPolicies()),
	}, nil
}

type networkPolicySyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &networkPolicySyncer{}

func (s *networkPolicySyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx, vObj.(*networkingv1.NetworkPolicy)))
}

func (s *networkPolicySyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	newNetworkPolicy := s.translateUpdate(ctx, pObj.(*networkingv1.NetworkPolicy), vObj.(*networkingv1.NetworkPolicy))
	if newNetworkPolicy != nil {
		translator.PrintChanges(pObj, newNetworkPolicy, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vObj, newNetworkPolicy)
}
