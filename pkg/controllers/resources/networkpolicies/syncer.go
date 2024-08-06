package networkpolicies

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMapper(ctx, &networkingv1.NetworkPolicy{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &networkPolicySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "networkpolicy", &networkingv1.NetworkPolicy{}, mapper),
	}, nil
}

type networkPolicySyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &networkPolicySyncer{}

func (s *networkPolicySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*networkingv1.NetworkPolicy](s)
}

func (s *networkPolicySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.NetworkPolicy]) (ctrl.Result, error) {
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	return syncer.CreateHostObject(ctx, event.Virtual, s.translate(ctx, event.Virtual), s.EventRecorder())
}

func (s *networkPolicySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.NetworkPolicy]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	s.translateUpdate(event.Host, event.Virtual)
	return ctrl.Result{}, nil
}

func (s *networkPolicySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.NetworkPolicy]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
}
