package misconfig

import (
	"context"
	"fmt"
	"slices"

	utilportforward "github.com/loft-sh/vcluster/pkg/util/portforward"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

const (
	appName  = "mounts-app"
	appImage = "nginxinc/nginx-unprivileged:stable-alpine3.20-slim"

	rootCAConfigMap = "kube-root-ca.crt"

	StatefulSet ControlPlaneKind = "StatefulSet"
	Deployment  ControlPlaneKind = "Deployment"
)

// ControlPlaneKind is the workload kind the chart renders for the control plane.
type ControlPlaneKind string

// controlPlaneReplicas returns spec.replicas of the control plane workload.
func controlPlaneReplicas(ctx context.Context, c kubernetes.Interface, kind ControlPlaneKind, namespace, name string) (int32, error) {
	switch kind {
	case StatefulSet:
		scale, err := c.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}
		return scale.Spec.Replicas, nil
	case Deployment:
		scale, err := c.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}
		return scale.Spec.Replicas, nil
	}
	return 0, fmt.Errorf("unknown control plane kind %q", kind)
}

// scaleControlPlane sets the replicas through the scale subresource, like `kubectl scale`.
func scaleControlPlane(ctx context.Context, c kubernetes.Interface, kind ControlPlaneKind, namespace, name string, replicas int32) error {
	switch kind {
	case StatefulSet:
		scale, err := c.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		scale.Spec.Replicas = replicas
		_, err = c.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	case Deployment:
		scale, err := c.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		scale.Spec.Replicas = replicas
		_, err = c.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	}
	return fmt.Errorf("unknown control plane kind %q", kind)
}

func controlPlanePods(ctx context.Context, c kubernetes.Interface, namespace, name string) ([]corev1.Pod, error) {
	pods, err := c.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=vcluster,release=" + name,
	})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func isPodReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// servingPod returns the only Ready control plane pod that isn't terminating.
func servingPod(ctx context.Context, c kubernetes.Interface, namespace, name string) (corev1.Pod, error) {
	pods, err := controlPlanePods(ctx, c, namespace, name)
	if err != nil {
		return corev1.Pod{}, err
	}

	var serving []corev1.Pod
	for _, pod := range pods {
		if pod.DeletionTimestamp == nil && isPodReady(pod) {
			serving = append(serving, pod)
		}
	}
	if len(serving) != 1 {
		return corev1.Pod{}, fmt.Errorf("want exactly 1 Ready control plane pod in %s, got %d", namespace, len(serving))
	}
	return serving[0], nil
}

// readyServiceEndpoints counts the ready addresses behind the control plane Service.
func readyServiceEndpoints(ctx context.Context, c kubernetes.Interface, namespace, service string) (int, error) {
	endpointSlices, err := c.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + service,
	})
	if err != nil {
		return 0, err
	}

	ready := 0
	for _, slice := range endpointSlices.Items {
		for _, ep := range slice.Endpoints {
			if ptr.Deref(ep.Conditions.Ready, false) {
				ready++
			}
		}
	}
	return ready, nil
}

// configMapNames lists ConfigMap names in a namespace, sorted, without the injected root CA.
func configMapNames(ctx context.Context, c kubernetes.Interface, namespace string) ([]string, error) {
	list, err := c.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, cm := range list.Items {
		if cm.Name != rootCAConfigMap {
			names = append(names, cm.Name)
		}
	}
	slices.Sort(names)
	return names, nil
}

func secretNames(ctx context.Context, c kubernetes.Interface, namespace string) ([]string, error) {
	list, err := c.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, secret := range list.Items {
		names = append(names, secret.Name)
	}
	slices.Sort(names)
	return names, nil
}

// mountingDeployment builds a Deployment whose pod mounts every given ConfigMap and Secret.
func mountingDeployment(namespace string, configMaps, secrets []string) *appsv1.Deployment {
	labels := map[string]string{
		"app": appName,
	}

	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	for _, name := range configMaps {
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      name,
			MountPath: "/mnt/" + name,
		})
	}
	for _, name := range secrets {
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: name,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      name,
			MountPath: "/mnt/" + name,
		})
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         appName,
							Image:        appImage,
							VolumeMounts: mounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

// podVClusterClient port-forwards to the syncer port of one control plane pod and returns a
// vCluster client for it. Call stop when done.
func podVClusterClient(ctx context.Context, hostConfig *rest.Config, hostClient kubernetes.Interface, vClusterConfig *rest.Config, namespace, pod string) (kubernetes.Interface, func(), error) {
	localPort, stop, err := utilportforward.ForwardPod(ctx, hostConfig, hostClient, namespace, pod, 8443)
	if err != nil {
		return nil, nil, err
	}
	podConfig := rest.CopyConfig(vClusterConfig)
	podConfig.Host = fmt.Sprintf("https://127.0.0.1:%d", localPort)
	client, err := kubernetes.NewForConfig(podConfig)
	if err != nil {
		stop()
		return nil, nil, err
	}
	return client, stop, nil
}
