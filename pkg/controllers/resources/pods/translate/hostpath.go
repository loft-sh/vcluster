package translate

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	VirtualPathTemplate = "/tmp/vcluster/%s/%s"

	PodLoggingHostPath = "/var/log/pods"
	LogHostPath        = "/var/log"

	KubeletPodPath = "/var/lib/kubelet/pods"

	PhysicalVolumeNameSuffix = "vcluster-physical"

	PhysicalLogVolumeMountPath     = "/var/vcluster/physical/log"
	PhysicalPodLogVolumeMountPath  = "/var/vcluster/physical/log/pods"
	PhysicalKubeletVolumeMountPath = "/var/vcluster/physical/kubelet/pods"
)

func (t *translator) ensureMountPropagation(pPod *corev1.Pod) {
	for i, container := range pPod.Spec.Containers {
		for j, volumeMount := range container.VolumeMounts {
			if volumeMount.MountPath == PodLoggingHostPath ||
				volumeMount.MountPath == KubeletPodPath ||
				volumeMount.MountPath == LogHostPath {
				hostToContainer := corev1.MountPropagationHostToContainer
				pPod.Spec.Containers[i].VolumeMounts[j].MountPropagation = &hostToContainer
			}
		}
	}
}

func (t *translator) rewriteHostPaths(pPod *corev1.Pod) {
	if len(pPod.Spec.Volumes) > 0 {
		t.log.Debugf("checking for host path volumes")

		// Keep track of the containers that already have a physical
		// mount added to them before, so that we don't append the same
		// mount path more than once to the same container. This is to
		// tackle edge cases like in kubevirt where we had
		// VolumeMounts
		//   - mountPath: /pods
		//     name: kubelet-pods-shortened
		//   - mountPath: /var/lib/kubelet/pods
		//     mountPropagation: Bidirectional
		//     name: kubelet-pods
		//
		// and Volumes
		//   - hostPath:
		//     path: /var/lib/kubelet/pods
		//     type: ""
		//     name: kubelet-pods-shortened
		//   - hostPath:
		//     path: /var/lib/kubelet/pods
		//     type: ""
		//     name: kubelet-pods
		//
		// causing the physical physical path to be mounted twice, one for each above
		// virtual volumeMounts
		// ---
		//   - name: kubelet-pods-shortened-vcluster-physical
		//     mountPath: "/var/vcluster/physical/kubelet/pods"
		//   - name: kubelet-pods-vcluster-physical
		//     mountPath: "/var/vcluster/physical/kubelet/pods"
		//     mountPropagation: Bidirectional
		kubeletMountPath := make(map[string]bool)
		podLogMountPath := make(map[string]bool)
		logMountPath := make(map[string]bool)

		for i, volume := range pPod.Spec.Volumes {
			if volume.HostPath != nil {
				if strings.TrimSuffix(volume.HostPath.Path, "/") == PodLoggingHostPath &&
					// avoid recursive rewriting of HostPaths across reconciles
					!strings.HasSuffix(volume.Name, PhysicalVolumeNameSuffix) {
					// we can't just mount the new hostpath to the virtual log path
					// we also need the actual 'physical' hostpath to be mounted
					// at a separate location and added to the correct containers as
					// only then the symlink targets created by hostpath-mapper would be
					// able to point to the actual log files to be traced.
					// Also we need to make sure this physical log path is not a
					// path used by the scraping agent - which should only see the
					// virtual log path
					t.log.Debugf("rewriting hostPath for pPod %s", pPod.Name)
					pPod.Spec.Volumes[i].HostPath.Path = t.virtualPodLogsPath

					t.log.Debugf("adding original hostPath to relevant containers")
					pPod = t.addPhysicalPathToVolumesAndCorrectContainers(
						volume.Name,
						volume.HostPath.Type,
						PodLoggingHostPath,
						PhysicalPodLogVolumeMountPath,
						podLogMountPath,
						pPod,
					)
				}

				if strings.TrimSuffix(volume.HostPath.Path, "/") == KubeletPodPath &&
					!strings.HasSuffix(volume.Name, PhysicalVolumeNameSuffix) {
					t.log.Debugf("rewriting hostPath for kubelet pods %s", pPod.Name)
					pPod.Spec.Volumes[i].HostPath.Path = t.virtualKubeletPodPath
					t.log.Debugf("adding original hostPath to relevant containers")
					pPod = t.addPhysicalPathToVolumesAndCorrectContainers(
						volume.Name,
						volume.HostPath.Type,
						KubeletPodPath,
						PhysicalKubeletVolumeMountPath,
						kubeletMountPath,
						pPod,
					)
				}

				if strings.TrimSuffix(volume.HostPath.Path, "/") == LogHostPath {
					pPod.Spec.Volumes[i].HostPath.Path = t.virtualLogsPath
					pPod = t.addPhysicalPathToVolumesAndCorrectContainers(
						volume.Name,
						volume.HostPath.Type,
						LogHostPath,
						PhysicalLogVolumeMountPath,
						logMountPath,
						pPod,
					)
				}
			}
		}

		t.ensureMountPropagation(pPod)
	}
}

// addPhysicalPathToVolumesAndCorrectContainers is only needed if deploying
// along side the vcluster-hostpath-mapper component
// see github.com/loft-sh/vcluster-hostpath-mapper
func (t *translator) addPhysicalPathToVolumesAndCorrectContainers(
	volName string,
	hostPathType *corev1.HostPathType,
	hostPath,
	physicalVolumeMount string,
	registerToCheck map[string]bool,
	pPod *corev1.Pod,
) *corev1.Pod {
	if !t.mountPhysicalHostPaths {
		// return without mounting extra physical mount
		return pPod
	}

	// add another volume with the correct suffix
	pPod.Spec.Volumes = append(pPod.Spec.Volumes, corev1.Volume{
		Name: fmt.Sprintf("%s-%s", volName, PhysicalVolumeNameSuffix),
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: hostPath,
				Type: hostPathType,
			},
		},
	})

	// find containers using the original volume and mount this new volume
	// under /var/vcluster/physical/ - this will be used by the hostPathMapper
	// as the target directory for the symlinks
	for i, container := range pPod.Spec.Containers {
		if len(container.VolumeMounts) > 0 {
			if _, ok := registerToCheck[container.Name]; ok {
				// physical path already added to the container
				// volume mounts hence skip
				continue
			}

			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == volName {
					// this container uses the original volume
					// keeping it as it is, we mount the physical volume
					// at the above specified mount point
					pVolMount := volumeMount.DeepCopy()
					pVolMount.Name = fmt.Sprintf("%s-%s", volName, PhysicalVolumeNameSuffix)
					pVolMount.MountPath = physicalVolumeMount

					pPod.Spec.Containers[i].VolumeMounts = append(pPod.Spec.Containers[i].VolumeMounts, *pVolMount)

					// add this to the container volume mount mapping
					registerToCheck[container.Name] = true
				}
			}
		}
	}

	return pPod
}
