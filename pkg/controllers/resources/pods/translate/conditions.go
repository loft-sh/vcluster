package translate

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

var coreConditions = map[string]bool{
	string(corev1.PodReady):        true,
	string(corev1.ContainersReady): true,
	string(corev1.PodInitialized):  true,
	string(corev1.PodScheduled):    true,
	"PodReadyToStartContainers":    true,
}

// updateConditions adds/updates new/old conditions in the physical Pod
func updateConditions(pPod, vPod *corev1.Pod, oldVPodStatus *corev1.PodStatus) {
	// check if newConditions need to be added.
	for _, vCondition := range oldVPodStatus.Conditions {
		if isCustomCondition(vCondition) {
			found := false
			for index, pCondition := range pPod.Status.Conditions {
				// found condition in pPod with same type, updating foundCondition
				if vCondition.Type == pCondition.Type {
					found = true
					if !equality.Semantic.DeepEqual(pCondition, vCondition) {
						pPod.Status.Conditions[index] = vCondition
					}
					break
				}
			}
			if !found {
				pPod.Status.Conditions = append(pPod.Status.Conditions, vCondition)
			}
		}
	}

	// don't sync custom conditions up
	newConditions := []corev1.PodCondition{}
	for _, pCondition := range pPod.Status.Conditions {
		if isCustomCondition(pCondition) {
			found := false
			for _, vCondition := range oldVPodStatus.Conditions {
				if pCondition.Type == vCondition.Type {
					found = true
					break
				}
			}
			if !found {
				// don't sync custom conditions we don't have on the virtual pod
				continue
			}
		}

		newConditions = append(newConditions, pCondition)
	}

	vPod.Status.Conditions = newConditions
}

// Check for custom condition
func isCustomCondition(condition corev1.PodCondition) bool {
	// if not a default condition, we assume it's a custom condition
	return !coreConditions[string(condition.Type)]
}
