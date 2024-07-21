package configmaps

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *configMapSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *corev1.ConfigMap {
	return s.TranslateMetadata(ctx, vObj).(*corev1.ConfigMap)
}

func (s *configMapSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *corev1.ConfigMap) {
	// check annotations & labels
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	pObj.Annotations = updatedAnnotations
	pObj.Labels = updatedLabels

	// bidirectional sync
	source, target := synccontext.SyncSourceTarget(ctx, pObj, vObj)
	target.Data = source.Data
	target.BinaryData = source.BinaryData
}
