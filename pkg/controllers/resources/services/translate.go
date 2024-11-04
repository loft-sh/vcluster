package services

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *serviceSyncer) translate(ctx *synccontext.SyncContext, vObj *corev1.Service) *corev1.Service {
	newService := translate.HostMetadata(vObj, s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj), s.excludedAnnotations...)
	newService.Spec.Selector = translate.HostLabelsMap(vObj.Spec.Selector, nil, vObj.Namespace, false)
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

func (s *serviceSyncer) translateToVirtual(ctx *synccontext.SyncContext, pObj *corev1.Service) *corev1.Service {
	newService := translate.VirtualMetadata(pObj, s.HostToVirtual(ctx, types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, pObj), s.excludedAnnotations...)
	newService.Spec.Selector = translate.VirtualLabelsMap(pObj.Spec.Selector, nil)

	// this rarely happens, but if services are created in the virtual
	// cluster directly circumventing the vcluster proxy, this needs to
	// be done as creating a service purely inside the
	// virtual cluster can cause a RequireDualStack ipFamily that
	// might not be supported in the host cluster, so we let the
	// host cluster decide for itself what ip family and policy
	// to use here.
	newService.Spec.IPFamilies = nil
	newService.Spec.IPFamilyPolicy = nil
	return newService
}

func StripNodePorts(vObj *corev1.Service) {
	for i := range vObj.Spec.Ports {
		vObj.Spec.Ports[i].NodePort = 0
	}
}

// AlignSpecWithServiceType removes any fields that are invalid for the specific service type
func AlignSpecWithServiceType(svc *corev1.Service) {
	if svc == nil || svc.Spec.Type == "" {
		return
	}

	// Default to ClusterIP if type is not specified
	if svc.Spec.Type == "" {
		svc.Spec.Type = corev1.ServiceTypeClusterIP
	}

	switch svc.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		cleanClusterIPFields(svc)
	case corev1.ServiceTypeNodePort:
		cleanNodePortFields(svc)
	case corev1.ServiceTypeLoadBalancer:
		cleanLoadBalancerFields(svc)
	case corev1.ServiceTypeExternalName:
		cleanExternalNameFields(svc)
	}
}

func cleanClusterIPFields(svc *corev1.Service) {
	// Clear fields not valid for ClusterIP
	svc.Spec.ExternalTrafficPolicy = ""
	svc.Spec.HealthCheckNodePort = 0
	svc.Spec.LoadBalancerIP = ""
	svc.Spec.LoadBalancerSourceRanges = nil
	svc.Spec.LoadBalancerClass = nil
	svc.Spec.ExternalName = ""
	svc.Spec.ExternalIPs = nil
	svc.Spec.AllocateLoadBalancerNodePorts = nil
}

func cleanNodePortFields(svc *corev1.Service) {
	// NodePort can have all ClusterIP fields plus some additional ones
	// Clear fields not valid for NodePort
	svc.Spec.LoadBalancerIP = ""
	svc.Spec.LoadBalancerSourceRanges = nil
	svc.Spec.LoadBalancerClass = nil
	svc.Spec.ExternalName = ""
}

func cleanLoadBalancerFields(svc *corev1.Service) {
	// LoadBalancer can have all NodePort fields plus some additional ones
	// Only need to clear ExternalName as it inherits from NodePort
	svc.Spec.ExternalName = ""
}

func cleanExternalNameFields(svc *corev1.Service) {
	// ExternalName services should only have metadata, type, and externalName
	svc.Spec.Ports = nil
	svc.Spec.ClusterIP = ""
	svc.Spec.ExternalIPs = nil
	svc.Spec.LoadBalancerIP = ""
	svc.Spec.LoadBalancerSourceRanges = nil
	svc.Spec.LoadBalancerClass = nil
	svc.Spec.ExternalTrafficPolicy = ""
	svc.Spec.HealthCheckNodePort = 0
	svc.Spec.PublishNotReadyAddresses = false
	svc.Spec.SessionAffinity = ""
	svc.Spec.SessionAffinityConfig = nil
	svc.Spec.IPFamilies = nil
	svc.Spec.IPFamilyPolicy = nil
	svc.Spec.AllocateLoadBalancerNodePorts = nil
	svc.Spec.InternalTrafficPolicy = nil
}
