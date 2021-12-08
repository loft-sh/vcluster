package services

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *syncer) translate(vObj *corev1.Service) (*corev1.Service, error) {
	newObj, err := s.translator.Translate(vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newService := newObj.(*corev1.Service)
	newService.Spec.Selector = nil
	newService.Spec.ClusterIP = ""
	newService.Spec.ClusterIPs = nil
	return newService, nil
}

func (s *syncer) translateUpdateBackwards(pObj, vObj *corev1.Service) *corev1.Service {
	var updated *corev1.Service

	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP {
		updated = newIfNil(updated, vObj)
		updated.Spec.ClusterIP = pObj.Spec.ClusterIP
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.ExternalIPs, pObj.Spec.ExternalIPs) {
		updated = newIfNil(updated, vObj)
		updated.Spec.ExternalIPs = pObj.Spec.ExternalIPs
	}

	if vObj.Spec.LoadBalancerIP != pObj.Spec.LoadBalancerIP {
		updated = newIfNil(updated, vObj)
		updated.Spec.LoadBalancerIP = pObj.Spec.LoadBalancerIP
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.LoadBalancerSourceRanges, pObj.Spec.LoadBalancerSourceRanges) {
		updated = newIfNil(updated, vObj)
		updated.Spec.LoadBalancerSourceRanges = pObj.Spec.LoadBalancerSourceRanges
	}

	return updated
}

func (s *syncer) translateUpdate(pObj, vObj *corev1.Service) *corev1.Service {
	var updated *corev1.Service

	// check ports
	if !equality.Semantic.DeepEqual(vObj.Spec.Ports, pObj.Spec.Ports) {
		updated = newIfNil(updated, pObj)
		updated.Spec.Ports = vObj.Spec.Ports
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

	// publish not ready addresses
	if vObj.Spec.PublishNotReadyAddresses != pObj.Spec.PublishNotReadyAddresses {
		updated = newIfNil(updated, pObj)
		updated.Spec.PublishNotReadyAddresses = vObj.Spec.PublishNotReadyAddresses
	}

	// type
	if vObj.Spec.Type != pObj.Spec.Type {
		updated = newIfNil(updated, pObj)
		updated.Spec.Type = vObj.Spec.Type
	}

	// external name
	if vObj.Spec.ExternalName != pObj.Spec.ExternalName {
		updated = newIfNil(updated, pObj)
		updated.Spec.ExternalName = vObj.Spec.ExternalName
	}

	// externalTrafficPolicy
	if vObj.Spec.ExternalTrafficPolicy != pObj.Spec.ExternalTrafficPolicy {
		updated = newIfNil(updated, pObj)
		updated.Spec.ExternalTrafficPolicy = vObj.Spec.ExternalTrafficPolicy
	}

	// session affinity
	if vObj.Spec.SessionAffinity != pObj.Spec.SessionAffinity {
		updated = newIfNil(updated, pObj)
		updated.Spec.SessionAffinity = vObj.Spec.SessionAffinity
	}

	// sessionAffinityConfig
	if !equality.Semantic.DeepEqual(vObj.Spec.SessionAffinityConfig, pObj.Spec.SessionAffinityConfig) {
		updated = newIfNil(updated, pObj)
		updated.Spec.SessionAffinityConfig = vObj.Spec.SessionAffinityConfig
	}

	// healthCheckNodePort
	if vObj.Spec.HealthCheckNodePort != pObj.Spec.HealthCheckNodePort {
		updated = newIfNil(updated, pObj)
		updated.Spec.HealthCheckNodePort = vObj.Spec.HealthCheckNodePort
	}

	return updated
}

func newIfNil(updated *corev1.Service, pObj *corev1.Service) *corev1.Service {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
