package networkpolicies

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &networkPolicySyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "networkpolicy", &networkingv1.NetworkPolicy{}),
	}, nil
}

type networkPolicySyncer struct {
	translator.NamespacedTranslator
}

var _ syncer.Syncer = &networkPolicySyncer{}

func (s *networkPolicySyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*networkingv1.NetworkPolicy)))
}

func (s *networkPolicySyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	newNetworkPolicy := s.translateUpdate(pObj.(*networkingv1.NetworkPolicy), vObj.(*networkingv1.NetworkPolicy))
	if newNetworkPolicy != nil {
		translator.PrintChanges(pObj, newNetworkPolicy, ctx.Log)
	}

	return s.SyncDownUpdate(ctx, vObj, newNetworkPolicy)
}
