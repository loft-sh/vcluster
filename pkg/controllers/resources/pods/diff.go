package pods

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func calcPodDiff(pPod, vPod *corev1.Pod, translateImages ImageTranslator) *corev1.Pod {
	var updatedPod *corev1.Pod
	updatedPodSpec := calcSpecDiff(pPod, vPod, translateImages)
	if updatedPodSpec != nil {
		updatedPod = pPod.DeepCopy()
		updatedPod.Spec = *updatedPodSpec
	}

	// check annotations
	if !translate.EqualExcept(pPod.Annotations, vPod.Annotations, OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Annotations = translate.SetExcept(vPod.Annotations, pPod.Annotations, OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation)
	}

	return updatedPod
}

// Changeable fields within the pod:
// - spec.containers[*].image
// - spec.initContainers[*].image
// - spec.activeDeadlineSeconds
//
// TODO: check for ephemereal containers
func calcSpecDiff(pObj, vObj *corev1.Pod, translateImages ImageTranslator) *corev1.PodSpec {
	var updatedPodSpec *corev1.PodSpec

	// active deadlines different?
	val, equal := isInt64Different(pObj.Spec.ActiveDeadlineSeconds, vObj.Spec.ActiveDeadlineSeconds)
	if !equal {
		updatedPodSpec = pObj.Spec.DeepCopy()
		updatedPodSpec.ActiveDeadlineSeconds = val
	}

	// is image different?
	updatedContainer := calcContainerImageDiff(pObj.Spec.Containers, vObj.Spec.Containers, translateImages, nil)
	if len(updatedContainer) != 0 {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.Containers = updatedContainer
	}

	// we have to skip some init images that are injected by us to change the /etc/hosts file
	var skipContainers map[string]bool
	if pObj.Annotations != nil && pObj.Annotations[HostsRewrittenAnnotation] == "true" {
		skipContainers = map[string]bool{
			HostsRewriteContainerName: true,
		}
	}

	updatedContainer = calcContainerImageDiff(pObj.Spec.InitContainers, vObj.Spec.InitContainers, translateImages, skipContainers)
	if len(updatedContainer) != 0 {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.InitContainers = updatedContainer
	}

	return updatedPodSpec
}

func calcContainerImageDiff(pContainers, vContainers []corev1.Container, translateImages ImageTranslator, skipContainers map[string]bool) []corev1.Container {
	newContainers := []corev1.Container{}
	changed := false
	for _, p := range pContainers {
		if skipContainers != nil && skipContainers[p.Name] {
			newContainers = append(newContainers, p)
			continue
		}

		for _, v := range vContainers {
			if p.Name == v.Name {
				if p.Image != translateImages.Translate(v.Image) {
					newContainer := *p.DeepCopy()
					newContainer.Image = translateImages.Translate(v.Image)
					newContainers = append(newContainers, newContainer)
					changed = true
				} else {
					newContainers = append(newContainers, p)
				}

				break
			}
		}
	}

	if changed == false {
		return nil
	}
	return newContainers
}

func isInt64Different(i1, i2 *int64) (*int64, bool) {
	if i1 == nil && i2 == nil {
		return nil, true
	} else if i1 != nil && i2 != nil {
		return pointer.Int64Ptr(*i2), *i1 == *i2
	}

	var updated *int64
	if i2 != nil {
		updated = pointer.Int64Ptr(*i2)
	}

	return updated, false
}
