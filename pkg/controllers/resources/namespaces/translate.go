package namespaces

import (
	"maps"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *namespaceSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *corev1.Namespace {
	newNamespace := s.TranslateMetadata(ctx, vObj).(*corev1.Namespace)

	if newNamespace.Labels == nil {
		newNamespace.Labels = map[string]string{}
	}

	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		newNamespace.Labels[k] = v
	}

	return newNamespace
}

func (s *namespaceSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *corev1.Namespace) {
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if updatedLabels == nil {
		updatedLabels = map[string]string{}
	}
	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		updatedLabels[k] = v
	}
	// set the kubernetes.io/metadata.name label
	updatedLabels[corev1.LabelMetadataName] = pObj.Name
	// check if any labels or annotations changed
	if !maps.Equal(updatedAnnotations, pObj.GetAnnotations()) || !maps.Equal(updatedLabels, pObj.GetLabels()) {
		pObj.Annotations = updatedAnnotations
		pObj.Labels = updatedLabels
	}
}
