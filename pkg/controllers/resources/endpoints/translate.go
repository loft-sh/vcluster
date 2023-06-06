package endpoints

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *endpointsSyncer) translate(ctx context.Context, vObj client.Object) *corev1.Endpoints {
	endpoints := s.TranslateMetadata(ctx, vObj).(*corev1.Endpoints)
	s.translateSpec(endpoints)

	// make sure we delete the control-plane.alpha.kubernetes.io/leader annotation
	// that will disable endpoint slice mirroring otherwise
	if endpoints.Annotations != nil {
		delete(endpoints.Annotations, "control-plane.alpha.kubernetes.io/leader")
	}

	return endpoints
}

func (s *endpointsSyncer) translateSpec(endpoints *corev1.Endpoints) {
	// translate the addresses
	for i, subset := range endpoints.Subsets {
		for j, addr := range subset.Addresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				endpoints.Subsets[i].Addresses[j].TargetRef.Name = translate.Default.PhysicalName(addr.TargetRef.Name, addr.TargetRef.Namespace)
				endpoints.Subsets[i].Addresses[j].TargetRef.Namespace = translate.Default.PhysicalNamespace(endpoints.Namespace)

				// TODO: set the actual values here
				endpoints.Subsets[i].Addresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].Addresses[j].TargetRef.ResourceVersion = ""
			}
		}
		for j, addr := range subset.NotReadyAddresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Name = translate.Default.PhysicalName(addr.TargetRef.Name, addr.TargetRef.Namespace)
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Namespace = translate.Default.PhysicalNamespace(endpoints.Namespace)

				// TODO: set the actual values here
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.ResourceVersion = ""
			}
		}
	}
}

func (s *endpointsSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.Endpoints) *corev1.Endpoints {
	var updated *corev1.Endpoints

	// check subsets
	translated := vObj.DeepCopy()
	s.translateSpec(translated)
	if !equality.Semantic.DeepEqual(translated.Subsets, pObj.Subsets) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Subsets = translated.Subsets
	}

	// check annotations & labels
	_, annotations, labels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	delete(annotations, "control-plane.alpha.kubernetes.io/leader")
	if !equality.Semantic.DeepEqual(annotations, pObj.Annotations) || !equality.Semantic.DeepEqual(labels, pObj.Labels) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = annotations
		updated.Labels = labels
	}

	return updated
}
