package translate

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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
			// handle scenarios where path ends with a /
			volumeMount.MountPath = strings.TrimSuffix(volumeMount.MountPath, "/")

			if volumeMount.MountPath == PodLoggingHostPath ||
				volumeMount.MountPath == KubeletPodPath ||
				volumeMount.MountPath == LogHostPath {
				hostToContainer := corev1.MountPropagationHostToContainer
				pPod.Spec.Containers[i].VolumeMounts[j].MountPropagation = &hostToContainer
			}
		}
	}
}

func (t *translator) rewriteHostPaths(ctx *synccontext.SyncContext, pPod *corev1.Pod) error {
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

				if !strings.HasSuffix(volume.Name, PhysicalVolumeNameSuffix) {
					if strings.TrimSuffix(volume.HostPath.Path, "/") == KubeletPodPath {
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
					} else if strings.HasPrefix(volume.HostPath.Path, KubeletPodPath+"/") {
						// Sub-path like /var/lib/kubelet/pods/<virtual-uid>/volumes/...:
						// translate the virtual pod UID to the physical pod UID so the path
						// points to a real directory on the host (no symlinks involved).
						translated, err := t.translateKubeletSubPath(ctx, volume.HostPath.Path)
						if err != nil {
							return fmt.Errorf("failed to translate kubelet sub-path for pPod %s: %w", pPod.Name, err)
						}

						t.log.Debugf("rewriting kubelet sub-path for pPod %s: %s -> %s", pPod.Name, volume.HostPath.Path, translated)
						pPod.Spec.Volumes[i].HostPath.Path = translated
					}
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

	return nil
}

// translateKubeletSubPath takes a hostPath of the form
// /var/lib/kubelet/pods/<virtual-pod-uid>/... and returns the same path with
// the virtual pod UID replaced by the corresponding physical (host) pod UID.
// This ensures the path points to a real directory on the host rather than a
// symlink created by the hostpath-mapper.
// Returns "" if the UID cannot be resolved (path left unchanged by caller).
func (t *translator) translateKubeletSubPath(ctx *synccontext.SyncContext, hostPath string) (string, error) {
	// Extract the UID segment: /var/lib/kubelet/pods/<uid>/rest...
	rest := strings.TrimPrefix(hostPath, KubeletPodPath+"/")
	slashIdx := strings.Index(rest, "/")
	var virtualUID string
	if slashIdx == -1 {
		virtualUID = rest
	} else {
		virtualUID = rest[:slashIdx]
	}
	if virtualUID == "" {
		return "", fmt.Errorf("no virtual UID found in hostPath %s", hostPath)
	}

	// Find the virtual pod with this UID by scanning the cached pod list.
	vPodList := &corev1.PodList{}
	if err := t.vClient.List(ctx, vPodList); err != nil {
		return "", fmt.Errorf("failed to list virtual pods to translate UID %s in hostPath %s: %w", virtualUID, hostPath, err)
	}

	var matchedName, matchedNamespace string
	for i := range vPodList.Items {
		if string(vPodList.Items[i].UID) == virtualUID {
			matchedName = vPodList.Items[i].Name
			matchedNamespace = vPodList.Items[i].Namespace
			break
		}
	}
	if matchedName == "" {
		// UID does not belong to any current virtual pod (e.g. already a physical UID,
		// or the pod was deleted). Leave the path unchanged to avoid aborting the whole pod sync.
		t.log.Debugf("no virtual pod found with UID %s, leaving hostPath %s unchanged", virtualUID, hostPath)
		return hostPath, nil
	}

	// Resolve the physical pod name/namespace via mappings.
	physicalPodRef := mappings.VirtualToHost(ctx, matchedName, matchedNamespace, mappings.Pods())

	// Fetch the physical pod to obtain its UID.
	physicalPod := &corev1.Pod{}
	if err := t.pClient.Get(ctx, physicalPodRef, physicalPod); err != nil {
		return "", fmt.Errorf("failed to get physical pod %s/%s for UID translation: %w", physicalPodRef.Namespace, physicalPodRef.Name, err)
	}

	physicalUID := string(physicalPod.UID)
	if physicalUID == "" {
		t.log.Debugf("physical pod %s/%s has no UID, leaving hostPath %s unchanged", physicalPodRef.Namespace, physicalPodRef.Name, hostPath)
		return hostPath, nil
	}

	return strings.Replace(hostPath, virtualUID, physicalUID, 1), nil
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
