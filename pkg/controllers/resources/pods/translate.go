package pods

import (
	"context"
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *podSyncer) translate(ctx *synccontext.SyncContext, vPod *corev1.Pod) (*corev1.Pod, error) {
	kubeIP, err := s.findKubernetesIP(ctx)
	if err != nil {
		return nil, err
	}

	dnsIP, err := s.findKubernetesDNSIP(ctx)
	if err != nil {
		return nil, err
	}

	// get services for pod
	serviceList := &corev1.ServiceList{}
	err = ctx.VirtualClient.List(context.Background(), serviceList, client.InNamespace(vPod.Namespace))
	if err != nil {
		return nil, err
	}

	ptrServiceList := make([]*corev1.Service, 0, len(serviceList.Items))
	for _, svc := range serviceList.Items {
		s := svc
		ptrServiceList = append(ptrServiceList, &s)
	}

	pPod, err := s.podTranslator.Translate(vPod, ptrServiceList, dnsIP, kubeIP)
	if err != nil {
		return nil, err
	}

	return pPod, err
}

func (s *podSyncer) translateUpdate(pObj, vObj *corev1.Pod) (*corev1.Pod, error) {
	return s.podTranslator.Diff(vObj, pObj)
}

func (s *podSyncer) findKubernetesIP(ctx *synccontext.SyncContext) (string, error) {
	pService := &corev1.Service{}
	err := ctx.CurrentNamespaceClient.Get(context.TODO(), types.NamespacedName{
		Name:      s.serviceName,
		Namespace: ctx.CurrentNamespace,
	}, pService)
	if err != nil {
		return "", err
	}

	return pService.Spec.ClusterIP, nil
}

func (s *podSyncer) findKubernetesDNSIP(ctx *synccontext.SyncContext) (string, error) {
	ip := s.translateAndFindService(ctx, "kube-system", "kube-dns")
	if ip == "" {
		return "", fmt.Errorf("waiting for DNS service IP")
	}

	return ip, nil
}

func (s *podSyncer) translateAndFindService(ctx *synccontext.SyncContext, namespace, name string) string {
	pName := translate.PhysicalName(name, namespace)
	pService := &corev1.Service{}
	err := ctx.PhysicalClient.Get(context.TODO(), types.NamespacedName{
		Name:      pName,
		Namespace: ctx.TargetNamespace,
	}, pService)
	if err != nil {
		return ""
	}

	return pService.Spec.ClusterIP
}
