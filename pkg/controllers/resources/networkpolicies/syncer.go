package networkpolicies

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/syncer/types"
	networkingv1 "k8s.io/api/networking/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (types.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.NetworkPolicies())
	if err != nil {
		return nil, err
	}

	return &networkPolicySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "networkpolicy", &networkingv1.NetworkPolicy{}, mapper),
	}, nil
}

type networkPolicySyncer struct {
	types.GenericTranslator
}

var _ types.Syncer = &networkPolicySyncer{}

func (s *networkPolicySyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return syncer.CreateHostObject(ctx, vObj, s.translate(ctx, vObj.(*networkingv1.NetworkPolicy)), s.EventRecorder())
}

func (s *networkPolicySyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	s.translateUpdate(ctx, pObj.(*networkingv1.NetworkPolicy), vObj.(*networkingv1.NetworkPolicy))

	return ctrl.Result{}, nil
}
