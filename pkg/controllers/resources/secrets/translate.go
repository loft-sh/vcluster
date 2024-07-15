package secrets

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

func (s *secretSyncer) create(ctx context.Context, vObj *corev1.Secret) *corev1.Secret {
	newSecret := s.TranslateMetadata(ctx, vObj).(*corev1.Secret)
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return newSecret
}
