package configmaps

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

func configNamesFromPod(pod *corev1.Pod) []string {
	configMaps := []string{}
	for _, c := range pod.Spec.Containers {
		configMaps = append(configMaps, configNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.InitContainers {
		configMaps = append(configMaps, configNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		configMaps = append(configMaps, configNamesFromEphemeralContainer(pod.Namespace, &c)...)
	}
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].ConfigMap != nil {
			configMaps = append(configMaps, pod.Namespace+"/"+pod.Spec.Volumes[i].ConfigMap.Name)
		}
		if pod.Spec.Volumes[i].Projected != nil {
			for j := range pod.Spec.Volumes[i].Projected.Sources {
				if pod.Spec.Volumes[i].Projected.Sources[j].ConfigMap != nil {
					configMaps = append(configMaps, pod.Namespace+"/"+pod.Spec.Volumes[i].Projected.Sources[j].ConfigMap.Name)
				}
			}
		}
	}
	return translate.UniqueSlice(configMaps)
}

func configNamesFromContainer(namespace string, container *corev1.Container) []string {
	configNames := []string{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			configNames = append(configNames, namespace+"/"+env.ValueFrom.ConfigMapKeyRef.Name)
		}
	}
	for _, from := range container.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			configNames = append(configNames, namespace+"/"+from.ConfigMapRef.Name)
		}
	}
	return configNames
}

func configNamesFromEphemeralContainer(namespace string, container *corev1.EphemeralContainer) []string {
	configNames := []string{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name != "" {
			configNames = append(configNames, namespace+"/"+env.ValueFrom.ConfigMapKeyRef.Name)
		}
	}
	for _, from := range container.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			configNames = append(configNames, namespace+"/"+from.ConfigMapRef.Name)
		}
	}
	return configNames
}
