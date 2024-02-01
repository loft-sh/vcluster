package poddisruptionbudgets

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	policyv1 "k8s.io/api/policy/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &pdbSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "podDisruptionBudget", &policyv1.PodDisruptionBudget{}),
	}, nil
}

type pdbSyncer struct {
	translator.NamespacedTranslator
}

func (pdb *pdbSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return pdb.SyncToHostCreate(ctx, vObj, pdb.translate(ctx.Context, vObj.(*policyv1.PodDisruptionBudget)))
}

func (pdb *pdbSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vPDB := vObj.(*policyv1.PodDisruptionBudget)
	pPDB := pObj.(*policyv1.PodDisruptionBudget)
	newPDB := pdb.translateUpdate(ctx.Context, pPDB, vPDB)
	if newPDB != nil {
		translator.PrintChanges(pObj, newPDB, ctx.Log)
	}

	return pdb.SyncToHostUpdate(ctx, vObj, newPDB)
}
