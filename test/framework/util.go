package framework

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

func (f *Framework) WaitForPodRunning(podName string, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second*5, PollTimeout, true, func(ctx context.Context) (bool, error) {
		pPodName := translate.Default.HostName(nil, podName, ns)
		pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		vpod, err := f.VClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if vpod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		return true, nil
	})
}

func (f *Framework) WaitForPodToComeUpWithReadinessConditions(podName string, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		pPodName := translate.Default.HostName(nil, podName, ns)
		pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		if len(pod.Status.Conditions) < 5 {
			return false, nil
		}
		return true, nil
	})
}

func (f *Framework) WaitForPodToComeUpWithEphemeralContainers(podName string, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		pPodName := translate.Default.HostName(nil, podName, ns)
		pod, err := f.HostClient.CoreV1().Pods(pPodName.Namespace).Get(ctx, pPodName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		if len(pod.Spec.EphemeralContainers) < 1 {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) WaitForPersistentVolumeClaimBound(pvcName, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		pPvcName := translate.Default.HostName(nil, pvcName, ns)
		pvc, err := f.HostClient.CoreV1().PersistentVolumeClaims(pPvcName.Namespace).Get(ctx, pPvcName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		if pvc.Status.Phase != corev1.ClaimBound {
			return false, nil
		}

		vpvc, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		if vpvc.Status.Phase != corev1.ClaimBound {
			return false, nil
		}

		return true, nil
	})
}

func (f *Framework) WaitForInitManifestConfigMapCreation(configMapName, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Millisecond*500, PollTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := f.VClusterClient.CoreV1().ConfigMaps(ns).Get(ctx, configMapName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
}

func (f *Framework) WaitForServiceAccount(saName string, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := f.VClusterClient.CoreV1().ServiceAccounts(ns).Get(ctx, saName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func (f *Framework) WaitForService(serviceName string, ns string) error {
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		pServiceName := translate.Default.HostName(nil, serviceName, ns)
		_, err := f.HostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// Some vcluster operations list Service, e.g. pod translation.
// To ensure expected results of such operation we need to wait until newly created Service is in syncer controller cache,
// otherwise syncer will operate on slightly outdated resources, which is not good for test stability.
// This function ensures that Service is actually in controller cache by making an update and checking for it in physical service.
func (f *Framework) WaitForServiceInSyncerCache(serviceName string, ns string) error {
	annotationKey := "e2e-test-bump"
	updated := false
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		vService, err := f.VClusterClient.CoreV1().Services(ns).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if !updated {
			if vService.Annotations == nil {
				vService.Annotations = map[string]string{}
			}
			vService.Annotations[annotationKey] = "arbitrary"
			_, err = f.VClusterClient.CoreV1().Services(ns).Update(ctx, vService, metav1.UpdateOptions{})
			if err != nil {
				if kerrors.IsConflict(err) || kerrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			updated = true
		}

		// Check for annotation
		pServiceName := translate.Default.HostName(nil, serviceName, ns)
		pService, err := f.HostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		_, ok := pService.Annotations[annotationKey]
		return ok, nil
	})
}

func (f *Framework) DeleteTestNamespace(ns string, waitUntilDeleted bool) error {
	err := f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, ns, metav1.DeleteOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !waitUntilDeleted {
		return nil
	}
	return wait.PollUntilContextTimeout(f.Context, time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		_, err = f.VClusterClient.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
}

func (f *Framework) GetDefaultSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser: ptr.To(int64(12345)),
	}
}

func (f *Framework) CreateCurlPod(ns string) (*corev1.Pod, error) {
	return f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "curl"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "curl",
					Image:           "curlimages/curl",
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: f.GetDefaultSecurityContext(),
					Command:         []string{"sleep"},
					Args:            []string{"9999"},
				},
			},
		},
	}, metav1.CreateOptions{})
}

func (f *Framework) CreateNginxPodAndService(ns string) (*corev1.Pod, *corev1.Service, error) {
	podName := "nginx"
	serviceName := "nginx"
	labels := map[string]string{"app": "nginx"}

	pod, err := f.VClusterClient.CoreV1().Pods(ns).Create(f.Context, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            podName,
					Image:           "nginxinc/nginx-unprivileged",
					ImagePullPolicy: corev1.PullIfNotPresent,
					SecurityContext: f.GetDefaultSecurityContext(),
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, err
	}

	service, err := f.VClusterClient.CoreV1().Services(ns).Create(f.Context, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Port: 8080},
			},
		},
	}, metav1.CreateOptions{})

	return pod, service, err
}

func (f *Framework) TestServiceIsEventuallyReachable(curlPod *corev1.Pod, service *corev1.Service) {
	var stdoutBuffer []byte
	var lastError error
	err := wait.PollUntilContextTimeout(f.Context, 10*time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		stdoutBuffer, _, lastError = f.curlService(ctx, curlPod, service)
		if lastError == nil && string(stdoutBuffer) == "200" {
			return true, nil
		}
		return false, nil
	})
	ExpectNoError(err, "Nginx service is expected to be reachable. On the last attempt got %s http code and following error:", string(stdoutBuffer), lastError)
}

func (f *Framework) TestServiceIsEventuallyUnreachable(curlPod *corev1.Pod, service *corev1.Service) {
	var stdoutBuffer, stderrBuffer []byte
	var lastError error
	err := wait.PollUntilContextTimeout(f.Context, 10*time.Second, PollTimeout, true, func(ctx context.Context) (bool, error) {
		stdoutBuffer, stderrBuffer, lastError = f.curlService(ctx, curlPod, service)
		if lastError != nil && strings.Contains(string(stderrBuffer), "timed out") && string(stdoutBuffer) == "000" {
			return true, nil
		}
		return false, nil
	})
	ExpectNoError(err, "Nginx service is expected to be unreachable. On the last attempt got %s http code and following error:", string(stdoutBuffer), lastError)
}

func (f *Framework) curlService(_ context.Context, curlPod *corev1.Pod, service *corev1.Service) ([]byte, []byte, error) {
	url := fmt.Sprintf("http://%s.%s.svc:%d/", service.GetName(), service.GetNamespace(), service.Spec.Ports[0].Port)
	cmd := []string{"curl", "-s", "--show-error", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "2", url}
	return podhelper.ExecBuffered(f.Context, f.VClusterConfig, curlPod.GetNamespace(), curlPod.GetName(), curlPod.Spec.Containers[0].Name, cmd, nil)
}

func (f *Framework) CreateEgressNetworkPolicyForDNS(ctx context.Context, ns string) (*networkingv1.NetworkPolicy, error) {
	UDPProtocol := corev1.ProtocolUDP
	return f.VClusterClient.NetworkingV1().NetworkPolicies(ns).Create(ctx, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "allow-coredns-egress"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 1053},
							Protocol: &UDPProtocol,
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"k8s-app": "kube-dns"}},
							NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
}
