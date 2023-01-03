package namespaces

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
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
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}
