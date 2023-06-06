package poddisruptionbudgets

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (pdb *pdbSyncer) translate(ctx context.Context, vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	newPDB := pdb.TranslateMetadata(ctx, vObj).(*policyv1.PodDisruptionBudget)
	if newPDB.Spec.Selector != nil {
		newPDB.Spec.Selector = translate.Default.TranslateLabelSelector(newPDB.Spec.Selector)
	}
	return newPDB
}

func (pdb *pdbSyncer) translateUpdate(ctx context.Context, pObj, vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	var updated *policyv1.PodDisruptionBudget

	// check max available and min available in spec
	if !equality.Semantic.DeepEqual(vObj.Spec.MaxUnavailable, pObj.Spec.MaxUnavailable) ||
		!equality.Semantic.DeepEqual(vObj.Spec.MinAvailable, pObj.Spec.MinAvailable) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.MaxUnavailable = vObj.Spec.MaxUnavailable
		updated.Spec.MinAvailable = vObj.Spec.MinAvailable
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := pdb.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check LabelSelector
	vObjLabelSelector := translate.Default.TranslateLabelSelector(vObj.Spec.Selector)
	if !equality.Semantic.DeepEqual(vObjLabelSelector, pObj.Spec.Selector) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.Selector = vObjLabelSelector
	}

	return updated
}
