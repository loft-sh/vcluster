package services

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *serviceSyncer) translate(ctx context.Context, vObj *corev1.Service) *corev1.Service {
	newService := s.TranslateMetadata(ctx, vObj).(*corev1.Service)
	newService.Spec.Selector = translate.Default.TranslateLabels(vObj.Spec.Selector, vObj.Namespace, nil)
	if newService.Spec.ClusterIP != "None" {
		newService.Spec.ClusterIP = ""
	}
	newService.Spec.ClusterIPs = nil

	// this rarely happens, but if services are created in the virtual
	// cluster directly circumventing the vcluster proxy, this needs to
	// be done as creating a service purely inside the
	// virtual cluster can cause a RequireDualStack ipFamily that
	// might not be supported in the host cluster, so we let the
	// host cluster decide for itself what ip family and policy
	// to use here.
	newService.Spec.IPFamilies = nil
	newService.Spec.IPFamilyPolicy = nil

	StripNodePorts(newService)
	return newService
}

func StripNodePorts(vObj *corev1.Service) {
	for i := range vObj.Spec.Ports {
		vObj.Spec.Ports[i].NodePort = 0
	}
}

func portsEqual(pObj, vObj *corev1.Service) bool {
	pSpec := pObj.Spec.DeepCopy()
	vSpec := vObj.Spec.DeepCopy()
	for i := range pSpec.Ports {
		pSpec.Ports[i].NodePort = 0
	}
	for i := range vSpec.Ports {
		vSpec.Ports[i].NodePort = 0
	}
	return equality.Semantic.DeepEqual(pSpec.Ports, vSpec.Ports)
}

func (s *serviceSyncer) translateUpdateBackwards(pObj, vObj *corev1.Service) *corev1.Service {
	var updated *corev1.Service

	if vObj.Spec.ClusterIP != pObj.Spec.ClusterIP {
		updated = translator.NewIfNil(updated, vObj)
		updated.Spec.ClusterIP = pObj.Spec.ClusterIP
	}

	if !equality.Semantic.DeepEqual(vObj.Spec.ExternalIPs, pObj.Spec.ExternalIPs) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Spec.ExternalIPs = pObj.Spec.ExternalIPs
	}

	if vObj.Spec.LoadBalancerIP != pObj.Spec.LoadBalancerIP {
		updated = translator.NewIfNil(updated, vObj)
		updated.Spec.LoadBalancerIP = pObj.Spec.LoadBalancerIP
	}

	// check if we need to sync node ports from host to virtual
	if pObj.Spec.Type == vObj.Spec.Type && portsEqual(pObj, vObj) && !equality.Semantic.DeepEqual(vObj.Spec.Ports, pObj.Spec.Ports) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Spec.Ports = pObj.Spec.Ports
	}

	return updated
}

func (s *serviceSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.Service) *corev1.Service {
	var updated *corev1.Service

	// check annotations
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	// remove the ServiceBlockDeletion annotation if it's not needed
	if vObj.Spec.ClusterIP == pObj.Spec.ClusterIP {
		delete(updatedAnnotations, ServiceBlockDeletion)
	}
	if !equality.Semantic.DeepEqual(updatedAnnotations, pObj.Annotations) || !equality.Semantic.DeepEqual(updatedLabels, pObj.Labels) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check ports
	if !equality.Semantic.DeepEqual(vObj.Spec.Ports, pObj.Spec.Ports) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.Ports = vObj.Spec.Ports

		// make sure node ports will be reset here
		StripNodePorts(updated)
	}

	// publish not ready addresses
	if vObj.Spec.PublishNotReadyAddresses != pObj.Spec.PublishNotReadyAddresses {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.PublishNotReadyAddresses = vObj.Spec.PublishNotReadyAddresses
	}

	// type
	if vObj.Spec.Type != pObj.Spec.Type {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.Type = vObj.Spec.Type
	}

	// external name
	if vObj.Spec.ExternalName != pObj.Spec.ExternalName {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.ExternalName = vObj.Spec.ExternalName
	}

	// externalTrafficPolicy
	if vObj.Spec.ExternalTrafficPolicy != pObj.Spec.ExternalTrafficPolicy {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.ExternalTrafficPolicy = vObj.Spec.ExternalTrafficPolicy
	}

	// session affinity
	if vObj.Spec.SessionAffinity != pObj.Spec.SessionAffinity {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.SessionAffinity = vObj.Spec.SessionAffinity
	}

	// sessionAffinityConfig
	if !equality.Semantic.DeepEqual(vObj.Spec.SessionAffinityConfig, pObj.Spec.SessionAffinityConfig) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.SessionAffinityConfig = vObj.Spec.SessionAffinityConfig
	}

	// load balancer source ranges
	if !equality.Semantic.DeepEqual(vObj.Spec.LoadBalancerSourceRanges, pObj.Spec.LoadBalancerSourceRanges) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.LoadBalancerSourceRanges = vObj.Spec.LoadBalancerSourceRanges
	}

	// healthCheckNodePort
	if vObj.Spec.HealthCheckNodePort != pObj.Spec.HealthCheckNodePort {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.HealthCheckNodePort = vObj.Spec.HealthCheckNodePort
	}

	// translate selector
	translated := pObj.DeepCopy()
	translated.Spec.Selector = translate.Default.TranslateLabels(vObj.Spec.Selector, vObj.Namespace, nil)
	if !equality.Semantic.DeepEqual(translated.Spec.Selector, pObj.Spec.Selector) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec.Selector = translated.Spec.Selector
	}

	return updated
}
