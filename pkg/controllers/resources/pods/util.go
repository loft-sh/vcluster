package pods

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"

	corev1 "k8s.io/api/core/v1"
)

func SecretNamesFromPod(pod *corev1.Pod) []string {
	secrets := []string{}
	for _, c := range pod.Spec.Containers {
		secrets = append(secrets, SecretNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.InitContainers {
		secrets = append(secrets, SecretNamesFromContainer(pod.Namespace, &c)...)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		secrets = append(secrets, SecretNamesFromEphemeralContainer(pod.Namespace, &c)...)
	}
	for i := range pod.Spec.ImagePullSecrets {
		secrets = append(secrets, pod.Namespace+"/"+pod.Spec.ImagePullSecrets[i].Name)
	}
	secrets = append(secrets, SecretNamesFromVolumes(pod)...)
	return translate.UniqueSlice(secrets)
}

func SecretNamesFromVolumes(pod *corev1.Pod) []string {
	secrets := []string{}
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Secret != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].Secret.SecretName)
		}
		if pod.Spec.Volumes[i].Projected != nil {
			for j := range pod.Spec.Volumes[i].Projected.Sources {
				if pod.Spec.Volumes[i].Projected.Sources[j].Secret != nil {
					secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].Projected.Sources[j].Secret.Name)
				}

				// check if projected volume source is a serviceaccount and in such a case
				// we re-write it as a secret too, handle accordingly
				if pod.Spec.Volumes[i].Projected.Sources[j].ServiceAccountToken != nil {
					secrets = append(secrets, pod.Namespace+"/"+podtranslate.SecretNameFromPodName(pod.Name, pod.Namespace))
				}
			}
		}
		if pod.Spec.Volumes[i].ISCSI != nil && pod.Spec.Volumes[i].ISCSI.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].ISCSI.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].RBD != nil && pod.Spec.Volumes[i].RBD.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].RBD.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].FlexVolume != nil && pod.Spec.Volumes[i].FlexVolume.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].FlexVolume.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].Cinder != nil && pod.Spec.Volumes[i].Cinder.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].Cinder.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].CephFS != nil && pod.Spec.Volumes[i].CephFS.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].CephFS.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].AzureFile != nil && pod.Spec.Volumes[i].AzureFile.SecretName != "" {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].AzureFile.SecretName)
		}
		if pod.Spec.Volumes[i].ScaleIO != nil && pod.Spec.Volumes[i].ScaleIO.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].ScaleIO.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].StorageOS != nil && pod.Spec.Volumes[i].StorageOS.SecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].StorageOS.SecretRef.Name)
		}
		if pod.Spec.Volumes[i].CSI != nil && pod.Spec.Volumes[i].CSI.NodePublishSecretRef != nil {
			secrets = append(secrets, pod.Namespace+"/"+pod.Spec.Volumes[i].CSI.NodePublishSecretRef.Name)
		}
	}
	return secrets
}

func SecretNamesFromContainer(namespace string, container *corev1.Container) []string {
	secrets := []string{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			secrets = append(secrets, namespace+"/"+env.ValueFrom.SecretKeyRef.Name)
		}
	}
	for _, from := range container.EnvFrom {
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			secrets = append(secrets, namespace+"/"+from.SecretRef.Name)
		}
	}
	return secrets
}

func SecretNamesFromEphemeralContainer(namespace string, container *corev1.EphemeralContainer) []string {
	secrets := []string{}
	for _, env := range container.Env {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name != "" {
			secrets = append(secrets, namespace+"/"+env.ValueFrom.SecretKeyRef.Name)
		}
	}
	for _, from := range container.EnvFrom {
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			secrets = append(secrets, namespace+"/"+from.SecretRef.Name)
		}
	}
	return secrets
}
