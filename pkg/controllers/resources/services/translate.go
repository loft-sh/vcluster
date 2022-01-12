package services

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *serviceSyncer) translate(vObj *corev1.Service) *corev1.Service {
	newService := s.TranslateMetadata(vObj).(*corev1.Service)
	newService.Spec.Selector = nil
	newService.Spec.ClusterIP = ""
	newService.Spec.ClusterIPs = nil
	return newService
}

func (s *serviceSyncer) translateUpdateBackwards(pObj, vObj *corev1.Service) *corev1.Service {
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

func (s *serviceSyncer) translateUpdate(pObj, vObj *corev1.Service) *corev1.Service {
	var updated *corev1.Service

	// check annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check ports
	if !equality.Semantic.DeepEqual(vObj.Spec.Ports, pObj.Spec.Ports) {
		updated = newIfNil(updated, pObj)
		updated.Spec.Ports = vObj.Spec.Ports
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
