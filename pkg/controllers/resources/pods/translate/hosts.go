package translate

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/coredns"
	corev1 "k8s.io/api/core/v1"
)

const (
	DisableSubdomainRewriteAnnotation = "vcluster.loft.sh/disable-subdomain-rewrite"
	HostsRewrittenAnnotation          = "vcluster.loft.sh/hosts-rewritten"
	HostsVolumeName                   = "vcluster-rewrite-hosts"
	HostsRewriteContainerName         = "vcluster-rewrite-hosts"
)

var (
	nonRoot             = true
	privilegeEscalation = false
	capabilities        = corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	}
	seccompProfile = corev1.SeccompProfile{
		Type: corev1.SeccompProfileTypeRuntimeDefault,
	}
)

func (t *translator) rewritePodHostnameFQDN(pPod *corev1.Pod, fromHost, toHostname, toHostnameFQDN string) {
	if pPod.Annotations == nil || pPod.Annotations[DisableSubdomainRewriteAnnotation] != "true" || pPod.Annotations[HostsRewrittenAnnotation] != "true" {
		image := t.overrideHostsImage
		if t.defaultImageRegistry != "" {
			image = strings.TrimSuffix(t.defaultImageRegistry, "/") + "/" + image
		}

		userID := coredns.GetUserID()
		groupID := coredns.GetGroupID()
		initContainer := corev1.Container{
			Name:    HostsRewriteContainerName,
			Image:   image,
			Command: []string{"sh"},
			Args:    []string{"-c", "sed -E -e 's/^(\\d+.\\d+.\\d+.\\d+\\s+)" + fromHost + "$/\\1 " + toHostnameFQDN + " " + toHostname + "/' /etc/hosts > /hosts/hosts"},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:                &userID,
				RunAsGroup:               &groupID,
				RunAsNonRoot:             &nonRoot,
				Capabilities:             &capabilities,
				AllowPrivilegeEscalation: &privilegeEscalation,
				SeccompProfile:           &seccompProfile,
			},
			Resources: t.overrideHostsResources,
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/hosts",
					Name:      HostsVolumeName,
				},
			},
		}

		// Add volume
		if pPod.Spec.Volumes == nil {
			pPod.Spec.Volumes = []corev1.Volume{}
		}
		pPod.Spec.Volumes = append(pPod.Spec.Volumes, corev1.Volume{
			Name: HostsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		// Add init container
		newContainers := []corev1.Container{initContainer}
		newContainers = append(newContainers, pPod.Spec.InitContainers...)
		pPod.Spec.InitContainers = newContainers

		if pPod.Annotations == nil {
			pPod.Annotations = map[string]string{}
		}
		pPod.Annotations[HostsRewrittenAnnotation] = "true"

		// translate containers
		for i := range pPod.Spec.Containers {
			if pPod.Spec.Containers[i].VolumeMounts == nil {
				pPod.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{}
			}
			pPod.Spec.Containers[i].VolumeMounts = append(pPod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				MountPath: "/etc/hosts",
				Name:      HostsVolumeName,
				SubPath:   "hosts",
			})
		}

		// translate init containers
		for i := range pPod.Spec.InitContainers {
			if pPod.Spec.InitContainers[i].Name != HostsRewriteContainerName {
				if pPod.Spec.InitContainers[i].VolumeMounts == nil {
					pPod.Spec.InitContainers[i].VolumeMounts = []corev1.VolumeMount{}
				}
				pPod.Spec.InitContainers[i].VolumeMounts = append(pPod.Spec.InitContainers[i].VolumeMounts, corev1.VolumeMount{
					MountPath: "/etc/hosts",
					Name:      HostsVolumeName,
					SubPath:   "hosts",
				})
			}
		}

		// translate ephemeral containers
		for i := range pPod.Spec.EphemeralContainers {
			if pPod.Spec.EphemeralContainers[i].VolumeMounts == nil {
				pPod.Spec.EphemeralContainers[i].VolumeMounts = []corev1.VolumeMount{}
			}
			pPod.Spec.EphemeralContainers[i].VolumeMounts = append(pPod.Spec.EphemeralContainers[i].VolumeMounts, corev1.VolumeMount{
				MountPath: "/etc/hosts",
				Name:      HostsVolumeName,
				SubPath:   "hosts",
			})
		}
	}
}
