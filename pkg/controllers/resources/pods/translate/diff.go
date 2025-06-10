package translate

import (
	"encoding/json"
	"strings"

	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *translator) Diff(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Pod]) error {
	// sync conditions
	event.Virtual.Status.Conditions, event.Host.Status.Conditions = patcher.CopyBidirectional(
		event.VirtualOld.Status.Conditions,
		event.Virtual.Status.Conditions,
		event.HostOld.Status.Conditions,
		event.Host.Status.Conditions,
	)

	// has status changed?
	vPod := event.Virtual
	pPod := event.Host
	vPod.Status = *pPod.Status.DeepCopy()
	stripInjectedSidecarContainers(vPod, pPod)

	// get Namespace resource in order to have access to its labels
	vNamespace := &corev1.Namespace{}
	err := t.vClient.Get(ctx, client.ObjectKey{Name: vPod.GetNamespace()}, vNamespace)
	if err != nil {
		return err
	}

	// spec diff
	t.calcSpecDiff(pPod, vPod)

	// bi-directionally sync labels & annotations
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(
		event,
		GetExcludedAnnotations(pPod)...,
	)

	// exclude namespace labels
	excludeLabelsFn := func(key string, value interface{}) (string, interface{}) {
		if strings.HasPrefix(key, translate.NamespaceLabelPrefix) {
			return "", nil
		}

		return key, value
	}
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdateFunction(
		event,
		excludeLabelsFn,
		excludeLabelsFn,
	)

	// update namespace labels
	for key := range event.Host.Labels {
		if strings.HasPrefix(key, translate.NamespaceLabelPrefix) {
			delete(event.Host.Labels, key)
		}
	}
	for k, v := range vNamespace.GetLabels() {
		event.Host.Labels[translate.HostLabelNamespace(k)] = v
	}

	// update pod annotations
	event.Host.Annotations[VClusterLabelsAnnotation] = LabelsAnnotation(vPod)
	if len(vPod.OwnerReferences) > 0 {
		ownerReferencesData, _ := json.Marshal(vPod.OwnerReferences)
		event.Host.Annotations[OwnerReferences] = string(ownerReferencesData)
		for _, ownerReference := range vPod.OwnerReferences {
			if ownerReference.APIVersion == appsv1.SchemeGroupVersion.String() && canAnnotateOwnerSetKind(ownerReference.Kind) {
				event.Host.Annotations[OwnerSetKind] = ownerReference.Kind
				break
			}
		}
	} else {
		delete(event.Host.Annotations, OwnerReferences)
		delete(event.Host.Annotations, OwnerSetKind)
	}

	return nil
}

func GetExcludedAnnotations(pPod *corev1.Pod) []string {
	annotations := []string{ClusterAutoScalerAnnotation, OwnerReferences, OwnerSetKind, NamespaceAnnotation, NameAnnotation, UIDAnnotation, ServiceAccountNameAnnotation, HostsRewrittenAnnotation, VClusterLabelsAnnotation}
	if pPod != nil {
		for _, v := range pPod.Spec.Volumes {
			if v.Projected != nil {
				for _, source := range v.Projected.Sources {
					if source.DownwardAPI != nil {
						for _, item := range source.DownwardAPI.Items {
							if item.FieldRef != nil {
								// check if it's a label we have to rewrite
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
func (t *translator) calcSpecDiff(pObj, vObj *corev1.Pod) {
	// active deadlines different?
	pObj.Spec.ActiveDeadlineSeconds = vObj.Spec.ActiveDeadlineSeconds

	// is image different?
	updatedContainer := calcContainerImageDiff(pObj.Spec.Containers, vObj.Spec.Containers, t.imageTranslator, nil)
	if len(updatedContainer) != 0 {
		pObj.Spec.Containers = updatedContainer
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
		pObj.Spec.InitContainers = updatedContainer
	}

	pObj.Spec.SchedulingGates = vObj.Spec.SchedulingGates
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

func stripInjectedSidecarContainers(vPod, pPod *corev1.Pod) {
	vInitContainersMap := make(map[string]bool)
	vContainersMap := make(map[string]bool)

	for _, vInitContainer := range vPod.Spec.InitContainers {
		vInitContainersMap[vInitContainer.Name] = true
	}

	for _, vContainer := range vPod.Spec.Containers {
		vContainersMap[vContainer.Name] = true
	}

	vPod.Status.InitContainerStatuses = []corev1.ContainerStatus{}
	for _, initContainerStatus := range pPod.Status.InitContainerStatuses {
		if _, ok := vInitContainersMap[initContainerStatus.Name]; ok {
			vPod.Status.InitContainerStatuses = append(vPod.Status.InitContainerStatuses, initContainerStatus)
		}
	}

	vPod.Status.ContainerStatuses = []corev1.ContainerStatus{}
	for _, containerStatus := range pPod.Status.ContainerStatuses {
		if _, ok := vContainersMap[containerStatus.Name]; ok {
			vPod.Status.ContainerStatuses = append(vPod.Status.ContainerStatuses, containerStatus)
		}
	}
}
