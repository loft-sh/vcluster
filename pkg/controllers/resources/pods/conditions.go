package pods

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/kubernetes"
)

// UpdateConditions adds/updates new/old conditions in the physical Pod
func UpdateConditions(ctx *synccontext.SyncContext, physicalClusterClient kubernetes.Interface, physicalPod *corev1.Pod, virtualPod *corev1.Pod) error {
	// check if the readinessGates are added to vPod
	if len(virtualPod.Spec.ReadinessGates) > 0 {
		newCustomConditions, existingCustomConditions := getCustomConditions(physicalPod, virtualPod)
		// check if newConditions need to be added.
		if len(newCustomConditions) > 0 {
			physicalPod.Status.Conditions = append(physicalPod.Status.Conditions, newCustomConditions...)
		}
		// check if existingConditions need to be updated.
		if len(existingCustomConditions) > 0 {
			for index, condition := range existingCustomConditions {
				physicalPod.Status.Conditions[index] = condition
			}
		}
		// update physical pod
		if len(newCustomConditions) > 0 || len(existingCustomConditions) > 0 {
			err := ctx.PhysicalClient.Status().Update(ctx.Context, physicalPod)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getCustomConditions returns new and existing custom conditions in the given virtual pod.
func getCustomConditions(physicalPod *corev1.Pod, virtualPod *corev1.Pod) ([]corev1.PodCondition, map[int]corev1.PodCondition) {
	// newConditions don't require index so slice will suffice.
	newCustomConditions := []corev1.PodCondition{}
	// existingConditions will need index so, need a map here.
	existingCustomConditions := map[int]corev1.PodCondition{}
	var foundCondition *corev1.PodCondition
	var index int
	for _, vCondition := range virtualPod.Status.Conditions {
		if isCustomCondition(virtualPod, vCondition) {
			for i, pCondition := range physicalPod.Status.Conditions {
				// found condition in pPod with same type, updating foundCondition and index
				if isCustomCondition(physicalPod, pCondition) && vCondition.Type == pCondition.Type {
					foundCondition = &pCondition
					index = i
					break
				}
			}
			if foundCondition != nil {
				// To avoid same status updates, checking status again.
				if !equality.Semantic.DeepEqual(*foundCondition, vCondition) {
					existingCustomConditions[index] = vCondition
				}
			} else {
				// foundCondition is nil means that its a new condition in vPod, needs to be updated to pPod
				newCustomConditions = append(newCustomConditions, vCondition)
			}
			// resetting the helper variables
			index = -1
			foundCondition = nil
		}
	}

	return newCustomConditions, existingCustomConditions
}

// Check for customCondition
func isCustomCondition(pod *corev1.Pod, condition corev1.PodCondition) bool {
	customCondition := condition.Type != corev1.ContainersReady &&
		condition.Type != corev1.PodInitialized &&
		condition.Type != corev1.PodReady &&
		condition.Type != corev1.PodScheduled
	if customCondition {
		for _, readinessGate := range pod.Spec.ReadinessGates {
			if readinessGate.ConditionType == condition.Type {
				return true
			}
		}
	}
	return false
}
