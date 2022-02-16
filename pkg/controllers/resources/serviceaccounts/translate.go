package serviceaccounts

import (
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	f = false
)

func (s *serviceAccountSyncer) translate(vObj client.Object) *corev1.ServiceAccount {
	pObj := s.TranslateMetadata(vObj).(*corev1.ServiceAccount)

	// Don't sync the secrets here as we will override them anyways
	pObj.Secrets = nil
	pObj.AutomountServiceAccountToken = &f
	pObj.ImagePullSecrets = nil
	return pObj
}

func (s *serviceAccountSyncer) translateUpdate(pObj, vObj *corev1.ServiceAccount) *corev1.ServiceAccount {
	var updated *corev1.ServiceAccount

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	return updated
}

func newIfNil(updated *corev1.ServiceAccount, pObj *corev1.ServiceAccount) *corev1.ServiceAccount {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
