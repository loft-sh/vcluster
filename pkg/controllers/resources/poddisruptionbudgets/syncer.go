package poddisruptionbudgets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMapper(ctx, &policyv1.PodDisruptionBudget{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &pdbSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "podDisruptionBudget", &policyv1.PodDisruptionBudget{}, mapper),
	}, nil
}

type pdbSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &pdbSyncer{}

func (s *pdbSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *pdbSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*policyv1.PodDisruptionBudget]) (ctrl.Result, error) {
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	newPDB := s.translate(ctx, event.Virtual)

	err := pro.ApplyPatchesHostObject(ctx, nil, newPDB, event.Virtual, ctx.Config.Sync.ToHost.PodDisruptionBudgets.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	return syncer.CreateHostObject(ctx, event.Virtual, newPDB, s.EventRecorder())
}

func (s *pdbSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*policyv1.PodDisruptionBudget]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.PodDisruptionBudgets.Patches, false))
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

	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Annotations = translate.VirtualAnnotations(event.Host, event.Virtual)
		event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	} else {
		event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	return ctrl.Result{}, nil
}

func (s *pdbSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*policyv1.PodDisruptionBudget]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
}
