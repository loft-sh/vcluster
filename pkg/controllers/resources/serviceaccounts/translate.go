package serviceaccounts

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	f = false
)

func (s *serviceAccountSyncer) translate(ctx context.Context, vObj client.Object) *corev1.ServiceAccount {
	pObj := s.TranslateMetadata(ctx, vObj).(*corev1.ServiceAccount)

	// Don't sync the secrets here as we will override them anyways
	pObj.Secrets = nil
	pObj.AutomountServiceAccountToken = &f
	pObj.ImagePullSecrets = nil
	return pObj
}

func (s *serviceAccountSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.ServiceAccount) {
	// check annotations & labels
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	pObj.Labels = updatedLabels
	pObj.Annotations = updatedAnnotations
}
