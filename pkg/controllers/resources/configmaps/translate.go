package configmaps

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *configMapSyncer) translate(vObj client.Object) *corev1.ConfigMap {
	return s.TranslateMetadata(vObj).(*corev1.ConfigMap)
}

func (s *configMapSyncer) translateUpdate(pObj, vObj *corev1.ConfigMap) *corev1.ConfigMap {
	var updated *corev1.ConfigMap

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = newIfNil(updated, pObj)
		updated.Data = vObj.Data
	}

	// check binary data
	if !equality.Semantic.DeepEqual(vObj.BinaryData, pObj.BinaryData) {
		updated = newIfNil(updated, pObj)
		updated.BinaryData = vObj.BinaryData
	}

	return updated
}

func newIfNil(updated *corev1.ConfigMap, pObj *corev1.ConfigMap) *corev1.ConfigMap {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
