package ingressclasses

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (i *ingressClassSyncer) createVirtual(ctx context.Context, pIngressClass *networkingv1.IngressClass) *networkingv1.IngressClass {
	return i.TranslateMetadata(ctx, pIngressClass).(*networkingv1.IngressClass)
}

func (i *ingressClassSyncer) updateVirtual(ctx context.Context, pObj, vObj *networkingv1.IngressClass) {
	changed, updatedAnnotations, updatedLabels := i.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.Controller, pObj.Spec.Controller) {
		vObj.Spec.Controller = pObj.Spec.Controller
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.Parameters, pObj.Spec.Parameters) {
		vObj.Spec.Parameters = pObj.Spec.Parameters
	}
}
