package ingressclasses

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	networkingv1 "k8s.io/api/networking/v1"
)

func (i *ingressClassSyncer) createVirtual(ctx *synccontext.SyncContext, pIngressClass *networkingv1.IngressClass) *networkingv1.IngressClass {
	return i.TranslateMetadata(ctx, pIngressClass).(*networkingv1.IngressClass)
}

func (i *ingressClassSyncer) updateVirtual(ctx *synccontext.SyncContext, pObj, vObj *networkingv1.IngressClass) {
	changed, updatedAnnotations, updatedLabels := i.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	vObj.Spec.Controller = pObj.Spec.Controller
	vObj.Spec.Parameters = pObj.Spec.Parameters
}
