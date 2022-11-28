package poddisruptionbudgets

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (pdb *pdbSyncer) translate(vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	newPDB := pdb.TranslateMetadata(vObj).(*policyv1.PodDisruptionBudget)
	if newPDB.Spec.Selector != nil {
		newPDB.Spec.Selector = translate.Default.TranslateLabelSelector(newPDB.Spec.Selector)
	}
	return newPDB
}

func (pdb *pdbSyncer) translateUpdate(pObj, vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	var updated *policyv1.PodDisruptionBudget

	// check max available and min available in spec
	if !equality.Semantic.DeepEqual(vObj.Spec.MaxUnavailable, pObj.Spec.MaxUnavailable) ||
		!equality.Semantic.DeepEqual(vObj.Spec.MinAvailable, pObj.Spec.MinAvailable) {
		updated = newIfNil(updated, pObj)
		updated.Spec.MaxUnavailable = vObj.Spec.MaxUnavailable
		updated.Spec.MinAvailable = vObj.Spec.MinAvailable
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := pdb.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check LabelSelector
	vObjLabelSelector := translate.Default.TranslateLabelSelector(vObj.Spec.Selector)
	if !equality.Semantic.DeepEqual(vObjLabelSelector, pObj.Spec.Selector) {
		updated = newIfNil(updated, pObj)
		updated.Spec.Selector = vObjLabelSelector
	}

	return updated
}

func newIfNil(updated *policyv1.PodDisruptionBudget, pObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
