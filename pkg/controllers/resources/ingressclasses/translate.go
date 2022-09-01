package ingressclasses

import (
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (i *ingressClassSyncer) translateBackwards(pIngressClass *networkingv1.IngressClass) *networkingv1.IngressClass {
	return i.TranslateMetadata(pIngressClass).(*networkingv1.IngressClass)
}

func (i *ingressClassSyncer) translateUpdateBackwards(pObj, vObj *networkingv1.IngressClass) *networkingv1.IngressClass {
	var updated *networkingv1.IngressClass

	changed, updatedAnnotations, updatedLabels := i.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.Controller, pObj.Spec.Controller) {
		updated = newIfNil(updated, vObj)
		updated.Spec.Controller = pObj.Spec.Controller
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.Parameters, pObj.Spec.Parameters) {
		updated = newIfNil(updated, vObj)
		updated.Spec.Parameters = pObj.Spec.Parameters
	}

	return updated
}

func newIfNil(updated, obj *networkingv1.IngressClass) *networkingv1.IngressClass {
	if updated == nil {
		return obj.DeepCopy()
	}

	return updated
}
