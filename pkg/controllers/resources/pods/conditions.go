package pods

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// UpdateConditions adds/updates new/old conditions in the physical Pod
func UpdateConditions(ctx *synccontext.SyncContext, physicalPod *corev1.Pod, virtualPod *corev1.Pod) (bool, error) {
	// check if the readinessGates are added to vPod
	updated := false
	if len(virtualPod.Spec.ReadinessGates) > 0 {
		// check if newConditions need to be added.
		for _, vCondition := range virtualPod.Status.Conditions {
			if isCustomCondition(virtualPod, vCondition) {
				found := false
				for index, pCondition := range physicalPod.Status.Conditions {
					// found condition in pPod with same type, updating foundCondition
					if vCondition.Type == pCondition.Type {
						found = true
						if !equality.Semantic.DeepEqual(pCondition, vCondition) {
							updated = true
							physicalPod.Status.Conditions[index] = vCondition
						}
						break
					}
				}
				if !found {
					physicalPod.Status.Conditions = append(physicalPod.Status.Conditions, vCondition)
					updated = true
				}
			}
		}

		// update physical pod
		if updated {
			ctx.Log.Infof("update physical pod %s/%s, because readiness gate condition status has changed", physicalPod.Namespace, physicalPod.Name)
			err := ctx.PhysicalClient.Status().Update(ctx.Context, physicalPod)
			if err != nil {
				return false, err
			}
		}
	}

	return updated, nil
}

// Check for custom condition
func isCustomCondition(pod *corev1.Pod, condition corev1.PodCondition) bool {
	for _, readinessGate := range pod.Spec.ReadinessGates {
		if readinessGate.ConditionType == condition.Type {
			return true
		}
	}
	return false
}
