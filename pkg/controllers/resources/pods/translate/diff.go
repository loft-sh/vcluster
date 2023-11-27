package translate

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *translator) Diff(ctx context.Context, vPod, pPod *corev1.Pod) (*corev1.Pod, error) {
	// get Namespace resource in order to have access to its labels
	vNamespace := &corev1.Namespace{}
	err := t.vClient.Get(ctx, client.ObjectKey{Name: vPod.ObjectMeta.GetNamespace()}, vNamespace)
	if err != nil {
		return nil, err
	}

	var updatedPod *corev1.Pod
	updatedPodSpec := t.calcSpecDiff(pPod, vPod)
	if updatedPodSpec != nil {
		updatedPod = pPod.DeepCopy()
		updatedPod.Spec = *updatedPodSpec
	}

	// check annotations
	_, updatedAnnotations, updatedLabels := translate.Default.ApplyMetadataUpdate(vPod, pPod, t.syncedLabels, getExcludedAnnotations(pPod)...)
	if updatedAnnotations == nil {
		updatedAnnotations = map[string]string{}
	}
	if updatedLabels == nil {
		updatedLabels = map[string]string{}
	}

	// set owner references
	updatedAnnotations[VClusterLabelsAnnotation] = LabelsAnnotation(vPod)
	if len(vPod.OwnerReferences) > 0 {
		ownerReferencesData, _ := json.Marshal(vPod.OwnerReferences)
		updatedAnnotations[OwnerReferences] = string(ownerReferencesData)
		for _, ownerReference := range vPod.OwnerReferences {
			if ownerReference.APIVersion == appsv1.SchemeGroupVersion.String() && canAnnotateOwnerSetKind(ownerReference.Kind) {
				updatedAnnotations[OwnerSetKind] = ownerReference.Kind
				break
			}
		}
	} else {
		delete(updatedAnnotations, OwnerReferences)
		delete(updatedAnnotations, OwnerSetKind)
	}

	if !equality.Semantic.DeepEqual(updatedAnnotations, pPod.Annotations) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Annotations = updatedAnnotations
	}

	// check pod and namespace labels
	for k, v := range vNamespace.GetLabels() {
		updatedLabels[translate.ConvertLabelKeyWithPrefix(NamespaceLabelPrefix, k)] = v
	}
	if !equality.Semantic.DeepEqual(updatedLabels, pPod.Labels) {
		if updatedPod == nil {
			updatedPod = pPod.DeepCopy()
		}
		updatedPod.Labels = updatedLabels
	}

	return updatedPod, nil
}

func getExcludedAnnotations(pPod *corev1.Pod) []string {
	annotations := []string{ClusterAutoScalerAnnotation, OwnerReferences, OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation, VClusterLabelsAnnotation}
	if pPod != nil {
		for _, v := range pPod.Spec.Volumes {
			if v.Projected != nil {
				for _, source := range v.Projected.Sources {
					if source.DownwardAPI != nil {
						for _, item := range source.DownwardAPI.Items {
							if item.FieldRef != nil {
								// check if its a label we have to rewrite
								annotationsMatch := FieldPathAnnotationRegEx.FindStringSubmatch(item.FieldRef.FieldPath)
								if len(annotationsMatch) == 2 {
									if strings.HasPrefix(annotationsMatch[1], ServiceAccountTokenAnnotation) {
										annotations = append(annotations, annotationsMatch[1])
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return annotations
}

// Changeable fields within the pod:
// - spec.containers[*].image
// - spec.initContainers[*].image
// - spec.activeDeadlineSeconds
//
// TODO: check for ephemereal containers
func (t *translator) calcSpecDiff(pObj, vObj *corev1.Pod) *corev1.PodSpec {
	var updatedPodSpec *corev1.PodSpec

	// active deadlines different?
	val, equal := isInt64Different(pObj.Spec.ActiveDeadlineSeconds, vObj.Spec.ActiveDeadlineSeconds)
	if !equal {
		updatedPodSpec = pObj.Spec.DeepCopy()
		updatedPodSpec.ActiveDeadlineSeconds = val
	}

	// is image different?
	updatedContainer := calcContainerImageDiff(pObj.Spec.Containers, vObj.Spec.Containers, t.imageTranslator, nil)
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

	updatedContainer = calcContainerImageDiff(pObj.Spec.InitContainers, vObj.Spec.InitContainers, t.imageTranslator, skipContainers)
	if len(updatedContainer) != 0 {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.InitContainers = updatedContainer
	}

	isEqual := isPodSpecSchedulingGatesDiff(pObj.Spec.SchedulingGates, vObj.Spec.SchedulingGates)
	if !isEqual {
		if updatedPodSpec == nil {
			updatedPodSpec = pObj.Spec.DeepCopy()
		}
		updatedPodSpec.SchedulingGates = vObj.Spec.SchedulingGates
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

	if !changed {
		return nil
	}
	return newContainers
}

func isInt64Different(i1, i2 *int64) (*int64, bool) {
	if i1 == nil && i2 == nil {
		return nil, true
	} else if i1 != nil && i2 != nil {
		return ptr.To(*i2), *i1 == *i2
	}

	var updated *int64
	if i2 != nil {
		updated = ptr.To(*i2)
	}

	return updated, false
}

func isPodSpecSchedulingGatesDiff(pGates, vGates []corev1.PodSchedulingGate) bool {
	if len(vGates) != len(pGates) {
		return false
	}
	for i, v := range vGates {
		if v.Name != pGates[i].Name {
			return false
		}
	}
	return true
}
