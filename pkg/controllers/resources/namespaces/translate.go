package namespaces

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *namespaceSyncer) translate(ctx context.Context, vObj client.Object) *corev1.Namespace {
	newNamespace := s.TranslateMetadata(ctx, vObj).(*corev1.Namespace)

	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		newNamespace.Labels[k] = v
	}

	return newNamespace
}

func (s *namespaceSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.Namespace) *corev1.Namespace {
	var updated *corev1.Namespace

	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		updatedLabels[k] = v
	}
	// set the kubernetes.io/metadata.name label
	updatedLabels[corev1.LabelMetadataName] = pObj.Name
	// check if any labels or annotations changed
	if !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}
