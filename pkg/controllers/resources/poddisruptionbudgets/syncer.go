package poddisruptionbudgets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/syncer/types"
	policyv1 "k8s.io/api/policy/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (types.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.PodDisruptionBudgets())
	if err != nil {
		return nil, err
	}

	return &pdbSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "podDisruptionBudget", &policyv1.PodDisruptionBudget{}, mapper),
	}, nil
}

type pdbSyncer struct {
	types.GenericTranslator
}

func (s *pdbSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return syncer.CreateHostObject(ctx, vObj, s.translate(ctx, vObj.(*policyv1.PodDisruptionBudget)), s.EventRecorder())
}

func (s *pdbSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	vPDB := vObj.(*policyv1.PodDisruptionBudget)
	pPDB := pObj.(*policyv1.PodDisruptionBudget)

	patch, err := patcher.NewSyncerPatcher(ctx, pPDB, vPDB)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pPDB, vPDB); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	s.translateUpdate(ctx, pPDB, vPDB)
	return ctrl.Result{}, nil
}
