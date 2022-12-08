package namespaces

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *namespaceSyncer) translate(vObj client.Object) *corev1.Namespace {
	newNamespace := s.TranslateMetadata(vObj).(*corev1.Namespace)
	return newNamespace
}

func (s *namespaceSyncer) translateUpdate(pObj, vObj *corev1.Namespace) *corev1.Namespace {
	var updated *corev1.Namespace
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}

func newIfNil(updated *corev1.Namespace, obj *corev1.Namespace) *corev1.Namespace {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
