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
