package serviceaccounts

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
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

func (s *serviceAccountSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.ServiceAccount) *corev1.ServiceAccount {
	var updated *corev1.ServiceAccount

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	return updated
}
