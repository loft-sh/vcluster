package configmaps

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *syncer) translate(vObj client.Object) (*corev1.ConfigMap, error) {
	newObj, err := s.translator.Translate(vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newConfigMap := newObj.(*corev1.ConfigMap)
	return newConfigMap, nil
}

func (s *syncer) translateUpdate(pObj, vObj *corev1.ConfigMap) *corev1.ConfigMap {
	var updated *corev1.ConfigMap

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

	// check annotations
	updatedAnnotations := s.translator.TranslateAnnotations(vObj, pObj)
	if !equality.Semantic.DeepEqual(updatedAnnotations, pObj.Annotations) {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
	}

	// check labels
	updatedLabels := s.translator.TranslateLabels(vObj)
	if !equality.Semantic.DeepEqual(updatedLabels, pObj.Labels) {
		updated = newIfNil(updated, pObj)
		updated.Labels = updatedLabels
	}

	return updated
}

func newIfNil(updated *corev1.ConfigMap, pObj *corev1.ConfigMap) *corev1.ConfigMap {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
