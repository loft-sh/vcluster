package endpoints

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *syncer) translate(vObj client.Object) (*corev1.Endpoints, error) {
	newObj, err := s.translator.Translate(vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	// translate the addresses
	endpoints := newObj.(*corev1.Endpoints)
	for i, subset := range endpoints.Subsets {
		for j, addr := range subset.Addresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				endpoints.Subsets[i].Addresses[j].TargetRef.Name = translate.PhysicalName(addr.TargetRef.Name, addr.TargetRef.Namespace)
				endpoints.Subsets[i].Addresses[j].TargetRef.Namespace = s.targetNamespace

				// TODO: set the actual values here
				endpoints.Subsets[i].Addresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].Addresses[j].TargetRef.ResourceVersion = ""
			}
		}
		for j, addr := range subset.NotReadyAddresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Name = translate.PhysicalName(addr.TargetRef.Name, addr.TargetRef.Namespace)
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Namespace = s.targetNamespace

				// TODO: set the actual values here
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.ResourceVersion = ""
			}
		}
	}

	// make sure we delete the control-plane.alpha.kubernetes.io/leader annotation
	// that will disable endpoint slice mirroring otherwise
	if endpoints.Annotations != nil {
		delete(endpoints.Annotations, "control-plane.alpha.kubernetes.io/leader")
	}

	return endpoints, nil
}

func (s *syncer) translateUpdate(pObj, vObj *corev1.Endpoints) (*corev1.Endpoints, error) {
	var updated *corev1.Endpoints

	// translate endpoints
	translated, err := s.translate(vObj)
	if err != nil {
		return nil, err
	}
	
	// check subsets
	if !equality.Semantic.DeepEqual(translated.Subsets, pObj.Subsets) {
		updated = newIfNil(updated, pObj)
		updated.Subsets = translated.Subsets
	}

	// check annotations
	if !equality.Semantic.DeepEqual(translated.Annotations, pObj.Annotations) {
		updated = newIfNil(updated, pObj)
		updated.Annotations = translated.Annotations
	}

	// check labels
	if !equality.Semantic.DeepEqual(translated.Labels, pObj.Labels) {
		updated = newIfNil(updated, pObj)
		updated.Labels = translated.Labels
	}

	return updated, nil
}

func newIfNil(updated *corev1.Endpoints, pObj *corev1.Endpoints) *corev1.Endpoints {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
