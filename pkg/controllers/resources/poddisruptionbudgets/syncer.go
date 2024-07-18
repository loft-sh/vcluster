package poddisruptionbudgets

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	policyv1 "k8s.io/api/policy/v1"
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

func (pdb *pdbSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vPDB := vObj.(*policyv1.PodDisruptionBudget)
	pPDB := pObj.(*policyv1.PodDisruptionBudget)
	newPDB := pdb.translateUpdate(ctx, pPDB, vPDB)
	if newPDB != nil {
		translator.PrintChanges(pObj, newPDB, ctx.Log)
	}

	return pdb.SyncToHostUpdate(ctx, vObj, newPDB)
}
