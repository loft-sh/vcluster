package pods

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *syncer) translate(vPod *corev1.Pod) (*corev1.Pod, error) {
	kubeIP, err := s.findKubernetesIP()
	if err != nil {
		return nil, err
	}

	dnsIP, err := s.findKubernetesDNSIP()
	if err != nil {
		return nil, err
	}

	// get services for pod
	serviceList := &corev1.ServiceList{}
	err = s.virtualClient.List(context.Background(), serviceList, client.InNamespace(vPod.Namespace))
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

func (s *syncer) translateUpdate(pObj, vObj *corev1.Pod) (*corev1.Pod, error) {
	return s.podTranslator.Diff(vObj, pObj)
}

func (s *syncer) findKubernetesIP() (string, error) {
	pService := &corev1.Service{}
	err := s.serviceClient.Get(context.TODO(), types.NamespacedName{
		Name:      s.serviceName,
		Namespace: s.serviceNamespace,
	}, pService)
	if err != nil {
		return "", err
	}

	return pService.Spec.ClusterIP, nil
}

func (s *syncer) findKubernetesDNSIP() (string, error) {
	ip := s.translateAndFindService("kube-system", "kube-dns")
	if ip == "" {
		return "", fmt.Errorf("waiting for DNS service IP")
	}

	return ip, nil
}

func (s *syncer) translateAndFindService(namespace, name string) string {
	pName := translate.PhysicalName(name, namespace)
	pService := &corev1.Service{}
	err := s.localClient.Get(context.TODO(), types.NamespacedName{
		Name:      pName,
		Namespace: s.targetNamespace,
	}, pService)
	if err != nil {
		return ""
	}

	return pService.Spec.ClusterIP
}
