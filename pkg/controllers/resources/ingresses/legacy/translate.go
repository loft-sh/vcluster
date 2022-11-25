package legacy

import (
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *ingressSyncer) TranslateMetadata(vObj client.Object) client.Object {
	return s.NamespacedTranslator.TranslateMetadata(util.UpdateAnnotations(vObj))
}

func (s *ingressSyncer) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	return s.NamespacedTranslator.TranslateMetadataUpdate(util.UpdateAnnotations(vObj), pObj)
}

func (s *ingressSyncer) translate(vIngress *networkingv1beta1.Ingress) *networkingv1beta1.Ingress {
	newIngress := s.TranslateMetadata(vIngress).(*networkingv1beta1.Ingress)
	newIngress.Spec = *translateSpec(vIngress.Namespace, &vIngress.Spec)
	return newIngress
}

func (s *ingressSyncer) translateUpdate(pObj, vObj *networkingv1beta1.Ingress) *networkingv1beta1.Ingress {
	var updated *networkingv1beta1.Ingress

	translatedSpec := *translateSpec(vObj.Namespace, &vObj.Spec)
	if !equality.Semantic.DeepEqual(translatedSpec, pObj.Spec) {
		updated = newIfNil(updated, pObj)
		updated.Spec = translatedSpec
	}

	changed, translatedAnnotations, translatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = translatedAnnotations
		updated.Labels = translatedLabels
	}

	return updated
}

func (s *ingressSyncer) translateUpdateBackwards(pObj, vObj *networkingv1beta1.Ingress) *networkingv1beta1.Ingress {
	var updated *networkingv1beta1.Ingress

	if vObj.Spec.IngressClassName == nil && pObj.Spec.IngressClassName != nil {
		updated = newIfNil(updated, vObj)
		updated.Spec.IngressClassName = pObj.Spec.IngressClassName
	}

	return updated
}

func translateSpec(namespace string, vIngressSpec *networkingv1beta1.IngressSpec) *networkingv1beta1.IngressSpec {
	retSpec := vIngressSpec.DeepCopy()
	if retSpec.Backend != nil {
		if retSpec.Backend.ServiceName != "" {
			retSpec.Backend.ServiceName = translate.Default.PhysicalName(retSpec.Backend.ServiceName, namespace)
		}
		if retSpec.Backend.Resource != nil {
			retSpec.Backend.Resource.Name = translate.Default.PhysicalName(retSpec.Backend.Resource.Name, namespace)
		}
	}

	for i, rule := range retSpec.Rules {
		if rule.HTTP != nil {
			for j, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName != "" {
					retSpec.Rules[i].HTTP.Paths[j].Backend.ServiceName = translate.Default.PhysicalName(retSpec.Rules[i].HTTP.Paths[j].Backend.ServiceName, namespace)
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

func newIfNil(updated *networkingv1beta1.Ingress, pObj *networkingv1beta1.Ingress) *networkingv1beta1.Ingress {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
