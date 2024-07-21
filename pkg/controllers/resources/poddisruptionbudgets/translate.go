package poddisruptionbudgets

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
)

func (pdb *pdbSyncer) translate(ctx *synccontext.SyncContext, vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	newPDB := pdb.TranslateMetadata(ctx, vObj).(*policyv1.PodDisruptionBudget)
	if newPDB.Spec.Selector != nil {
		newPDB.Spec.Selector = translate.Default.TranslateLabelSelector(newPDB.Spec.Selector)
	}
	return newPDB
}

func (pdb *pdbSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *policyv1.PodDisruptionBudget) {
	pObj.Spec.MaxUnavailable = vObj.Spec.MaxUnavailable
	pObj.Spec.MinAvailable = vObj.Spec.MinAvailable

	// check annotations
	_, updatedAnnotations, updatedLabels := pdb.TranslateMetadataUpdate(ctx, vObj, pObj)
	pObj.Annotations = updatedAnnotations
	pObj.Labels = updatedLabels

	// check LabelSelector
	pObj.Spec.Selector = translate.Default.TranslateLabelSelector(vObj.Spec.Selector)
}
