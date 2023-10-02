package pods

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
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

// UpdateConditions adds/updates new/old conditions in the physical Pod
func UpdateConditions(ctx *synccontext.SyncContext, physicalPod *corev1.Pod, virtualPod *corev1.Pod) (*corev1.Pod, error) {
	// check if the readinessGates are added to vPod
	physicalPod = physicalPod.DeepCopy()
	updated := false

	// check if newConditions need to be added.
	for _, vCondition := range virtualPod.Status.Conditions {
		if isCustomCondition(vCondition) {
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
		ctx.Log.Infof("update physical pod %s/%s, because custom pod conditions have changed", physicalPod.Namespace, physicalPod.Name)
		err := ctx.PhysicalClient.Status().Update(ctx.Context, physicalPod)
		if err != nil {
			return nil, err
		}
	}

	// don't sync custom conditions up
	newConditions := []corev1.PodCondition{}
	for _, pCondition := range physicalPod.Status.Conditions {
		if isCustomCondition(pCondition) {
			found := false
			for _, vCondition := range virtualPod.Status.Conditions {
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
	physicalPod.Status.Conditions = newConditions
	return physicalPod, nil
}

// Check for custom condition
func isCustomCondition(condition corev1.PodCondition) bool {
	// if not a default condition, we assume it's a custom condition
	return !coreConditions[string(condition.Type)]
}
