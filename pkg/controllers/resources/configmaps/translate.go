package configmaps

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *configMapSyncer) translate(ctx context.Context, vObj client.Object) *corev1.ConfigMap {
	return s.TranslateMetadata(ctx, vObj).(*corev1.ConfigMap)
}

func (s *configMapSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.ConfigMap) {
	// check annotations & labels
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	pObj.Annotations = updatedAnnotations
	pObj.Labels = updatedLabels

	pObj.Data = vObj.Data

	pObj.BinaryData = vObj.BinaryData
}
