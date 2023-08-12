package configmaps

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *configMapSyncer) translate(ctx context.Context, vObj client.Object) *corev1.ConfigMap {
	return s.TranslateMetadata(ctx, vObj).(*corev1.ConfigMap)
}

func (s *configMapSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.ConfigMap) *corev1.ConfigMap {
	var updated *corev1.ConfigMap

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Data = vObj.Data
	}

	// check binary data
	if !equality.Semantic.DeepEqual(vObj.BinaryData, pObj.BinaryData) {
		updated = translator.NewIfNil(updated, pObj)
		updated.BinaryData = vObj.BinaryData
	}

	return updated
}
