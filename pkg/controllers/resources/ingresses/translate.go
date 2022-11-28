package ingresses

import (
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *ingressSyncer) translate(vIngress *networkingv1.Ingress) *networkingv1.Ingress {
	newIngress := s.TranslateMetadata(vIngress).(*networkingv1.Ingress)
	newIngress.Spec = *translateSpec(vIngress.Namespace, &vIngress.Spec)
	newIngress.Annotations, _ = translateIngressAnnotations(newIngress.Annotations, vIngress.Namespace)
	return newIngress
}

func (s *ingressSyncer) TranslateMetadata(vObj client.Object) client.Object {
	return s.NamespacedTranslator.TranslateMetadata(util.UpdateAnnotations(vObj))
}

func (s *ingressSyncer) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	return s.NamespacedTranslator.TranslateMetadataUpdate(util.UpdateAnnotations(vObj), pObj)
}

func (s *ingressSyncer) translateUpdate(pObj, vObj *networkingv1.Ingress) *networkingv1.Ingress {
	var updated *networkingv1.Ingress

	translatedSpec := *translateSpec(vObj.Namespace, &vObj.Spec)
	if !equality.Semantic.DeepEqual(translatedSpec, pObj.Spec) {
		updated = newIfNil(updated, pObj)
		updated.Spec = translatedSpec
	}

	_, translatedAnnotations, translatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	translatedAnnotations, _ = translateIngressAnnotations(translatedAnnotations, vObj.Namespace)
	if !equality.Semantic.DeepEqual(translatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(translatedLabels, pObj.GetLabels()) {
		updated = newIfNil(updated, pObj)
		updated.Annotations = translatedAnnotations
		updated.Labels = translatedLabels
	}

	return updated
}

func (s *ingressSyncer) translateUpdateBackwards(pObj, vObj *networkingv1.Ingress) *networkingv1.Ingress {
	var updated *networkingv1.Ingress

	if vObj.Spec.IngressClassName == nil && pObj.Spec.IngressClassName != nil {
		updated = newIfNil(updated, vObj)
		updated.Spec.IngressClassName = pObj.Spec.IngressClassName
	}

	return updated
}

func translateSpec(namespace string, vIngressSpec *networkingv1.IngressSpec) *networkingv1.IngressSpec {
	retSpec := vIngressSpec.DeepCopy()
	if retSpec.DefaultBackend != nil {
		if retSpec.DefaultBackend.Service != nil && retSpec.DefaultBackend.Service.Name != "" {
			retSpec.DefaultBackend.Service.Name = translate.Default.PhysicalName(retSpec.DefaultBackend.Service.Name, namespace)
		}
		if retSpec.DefaultBackend.Resource != nil {
			retSpec.DefaultBackend.Resource.Name = translate.Default.PhysicalName(retSpec.DefaultBackend.Resource.Name, namespace)
		}
	}

	for i, rule := range retSpec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil && path.Backend.Service.Name != "" {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name = translate.Default.PhysicalName(retSpec.Rules[i].HTTP.Paths[j].Backend.Service.Name, namespace)
				}
				if path.Backend.Resource != nil {
					retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name = translate.Default.PhysicalName(retSpec.Rules[i].HTTP.Paths[j].Backend.Resource.Name, namespace)
				}
			}
		}
	}

	for i, tls := range retSpec.TLS {
		if tls.SecretName != "" {
			retSpec.TLS[i].SecretName = translate.Default.PhysicalName(retSpec.TLS[i].SecretName, namespace)
		}
	}

	return retSpec
}

func newIfNil(updated *networkingv1.Ingress, pObj *networkingv1.Ingress) *networkingv1.Ingress {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
