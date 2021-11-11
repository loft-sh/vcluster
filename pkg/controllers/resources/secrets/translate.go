package secrets

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *syncer) translate(vObj *corev1.Secret) (*corev1.Secret, error) {
	newObj, err := s.translator.Translate(vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newSecret := newObj.(*corev1.Secret)
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return newSecret, nil
}

func (s *syncer) translateUpdate(pObj, vObj *corev1.Secret) *corev1.Secret {
	var updated *corev1.Secret

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = newIfNil(updated, pObj)
		updated.Data = vObj.Data
	}

	// check secret type
	if vObj.Type != pObj.Type && vObj.Type != corev1.SecretTypeServiceAccountToken {
		updated = newIfNil(updated, pObj)
		updated.Type = vObj.Type
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

func newIfNil(updated *corev1.Secret, pObj *corev1.Secret) *corev1.Secret {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
