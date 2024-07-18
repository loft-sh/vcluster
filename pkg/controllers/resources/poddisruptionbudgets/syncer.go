package poddisruptionbudgets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	policyv1 "k8s.io/api/policy/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &pdbSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "podDisruptionBudget", &policyv1.PodDisruptionBudget{}, mappings.PodDisruptionBudgets()),
	}, nil
}

type pdbSyncer struct {
	syncertypes.GenericTranslator
}

func (pdb *pdbSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return pdb.SyncToHostCreate(ctx, vObj, pdb.translate(ctx, vObj.(*policyv1.PodDisruptionBudget)))
}

func (pdb *pdbSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
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
			pdb.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	pdb.translateUpdate(ctx.Context, pPDB, vPDB)

	return ctrl.Result{}, nil
}
