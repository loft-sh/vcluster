package services

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
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
