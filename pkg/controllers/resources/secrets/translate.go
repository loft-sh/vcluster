package secrets

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *secretSyncer) translate(vObj *corev1.Secret) *corev1.Secret {
	newSecret := s.TranslateMetadata(vObj).(*corev1.Secret)
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return newSecret
}

func (s *secretSyncer) translateUpdate(pObj, vObj *corev1.Secret) *corev1.Secret {
	var updated *corev1.Secret

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Data = vObj.Data
	}

	// check secret type
	if vObj.Type != pObj.Type && vObj.Type != corev1.SecretTypeServiceAccountToken {
		updated = translator.NewIfNil(updated, pObj)
		updated.Type = vObj.Type
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}
