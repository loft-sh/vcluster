package pods

import (
	"context"
	"fmt"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *podSyncer) translate(ctx *synccontext.SyncContext, vPod *corev1.Pod) (*corev1.Pod, error) {
	kubeIP, dnsIP, ptrServiceList, err := s.getK8sIPDNSIPServiceList(ctx, vPod)
	if err != nil {
		return nil, err
	}

	pPod, err := s.podTranslator.Translate(ctx.Context, vPod, ptrServiceList, dnsIP, kubeIP)
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
	err = ctx.VirtualClient.List(ctx.Context, serviceList, client.InNamespace(vPod.Namespace))
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

func (s *podSyncer) translateUpdate(ctx context.Context, pClient client.Client, pObj, vObj *corev1.Pod) (*corev1.Pod, error) {
	secret, err := podtranslate.GetSecretIfExists(ctx, pClient, vObj.Name, vObj.Namespace)
	if err != nil {
		return nil, err
	} else if secret != nil {
		// check if owner is vcluster service, if so, modify to pod as owner
		err := podtranslate.SetPodAsOwner(ctx, pObj, pClient, secret)
		if err != nil {
			return nil, err
		}
	}

	return s.podTranslator.Diff(ctx, vObj, pObj)
}

func (s *podSyncer) findKubernetesIP(ctx *synccontext.SyncContext) (string, error) {
	pService := &corev1.Service{}
	err := ctx.CurrentNamespaceClient.Get(ctx.Context, types.NamespacedName{
		Name:      s.serviceName,
		Namespace: ctx.CurrentNamespace,
	}, pService)
	if err != nil {
		return "", err
	}

	return pService.Spec.ClusterIP, nil
}

func (s *podSyncer) findKubernetesDNSIP(ctx *synccontext.SyncContext) (string, error) {
	pClient, namespace := specialservices.Default.DNSNamespace(ctx)

	// first try to find the actual synced service, then fallback to a different if we have a suffix (only in the case of integrated coredns)
	ip := s.translateAndFindService(
		ctx,
		pClient,
		namespace,
		translate.Default.PhysicalName(specialservices.DefaultKubeDNSServiceName, specialservices.DefaultKubeDNSServiceNamespace),
	)
	if ip == "" {
		return "", fmt.Errorf("waiting for DNS service IP")
	}

	return ip, nil
}

func (s *podSyncer) translateAndFindService(ctx *synccontext.SyncContext, kubeClient client.Client, namespace, name string) string {
	pService := &corev1.Service{}
	err := kubeClient.Get(ctx.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, pService)
	if err != nil {
		return ""
	}

	return pService.Spec.ClusterIP
}
