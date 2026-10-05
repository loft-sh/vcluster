package pods

import (
	"errors"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *podSyncer) translate(ctx *synccontext.SyncContext, vPod *corev1.Pod) (*corev1.Pod, error) {
	kubeIP, dnsIP, ptrServiceList, err := s.getK8sIPDNSIPServiceList(ctx, vPod)
	if err != nil {
		return nil, err
	}

	pPod, err := s.podTranslator.Translate(ctx, vPod, ptrServiceList, dnsIP, kubeIP)
	if err != nil {
		return nil, err
	}

	return pPod, err
}

func (s *podSyncer) getK8sIPDNSIPServiceList(ctx *synccontext.SyncContext, vPod *corev1.Pod) (string, string, []*corev1.Service, error) {
	kubeIP, err := s.findKubernetesIP(ctx)
	if err != nil {
		return "", "", nil, err
	}

	dnsIP, err := s.findKubernetesDNSIP(ctx)
	if err != nil {
		return "", "", nil, err
	}

	// get services for pod
	serviceList := &corev1.ServiceList{}
	err = ctx.VirtualClient.List(ctx, serviceList, client.InNamespace(vPod.Namespace))
	if err != nil {
		return "", "", nil, err
	}

	ptrServiceList := make([]*corev1.Service, 0, len(serviceList.Items))
	for _, svc := range serviceList.Items {
		s := svc
		ptrServiceList = append(ptrServiceList, &s)
	}
	return kubeIP, dnsIP, ptrServiceList, nil
}

func (s *podSyncer) findKubernetesIP(ctx *synccontext.SyncContext) (string, error) {
	pService := &corev1.Service{}
	err := ctx.CurrentNamespaceClient.Get(ctx, types.NamespacedName{
		Name:      s.serviceName,
		Namespace: ctx.CurrentNamespace,
	}, pService)
	if err != nil {
		return "", err
	}

	return pService.Spec.ClusterIP, nil
}

func (s *podSyncer) findKubernetesDNSIP(ctx *synccontext.SyncContext) (string, error) {
	if specialservices.Default == nil {
		return "", errors.New("specialservices default not initialized")
	}

	// translate service name
	pService := mappings.VirtualToHostName(ctx, specialservices.DefaultKubeDNSServiceName, specialservices.DefaultKubeDNSServiceNamespace, mappings.Services())

	// first try to find the actual synced service, then fallback to a different if we have a suffix (only in the case of integrated coredns)
	pClient, namespace := specialservices.Default.DNSNamespace(ctx)
	ip := s.translateAndFindDNSService(
		ctx,
		pClient,
		namespace,
		pService,
	)
	if ip == "" {
		return "", fmt.Errorf("waiting for DNS service IP")
	}

	return ip, nil
}

func (s *podSyncer) translateAndFindDNSService(ctx *synccontext.SyncContext, kubeClient client.Client, namespace, name string) string {
	pService := &corev1.Service{}
	err := kubeClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pService)
	if err != nil {
		klog.FromContext(ctx).V(1).Info("Error trying to find dns service", "error", err)
		return ""
	}

	return pService.Spec.ClusterIP
}
