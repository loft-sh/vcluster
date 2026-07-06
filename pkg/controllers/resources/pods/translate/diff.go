package translate

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *translator) Diff(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Pod]) error {
	// Sync conditions. We only keep the reconciled host side; the virtual conditions are
	// recomputed below when vPod.Status is overwritten with the host status.
	_, event.Host.Status.Conditions = t.conditionsCopyBidirectional(
		event.VirtualOld.Status.Conditions,
		event.Virtual.Status.Conditions,
		event.HostOld.Status.Conditions,
		event.Host.Status.Conditions,
	)

	// has status changed?
	vPod := event.Virtual
	pPod := event.Host

	// Copy the host status onto the virtual pod, but keep a few virtual values that the host
	// must not overwrite:
	//   - QOSClass: the host treats it as immutable (since K8s 1.32), so it can't be changed.
	//   - ObservedGeneration: a virtual cluster on K8s < 1.34 doesn't store it, so copying the
	//     host value would keep looking like a change and patch the status on every reconcile.
	originalQOSClass := vPod.Status.QOSClass
	originalObservedGeneration := vPod.Status.ObservedGeneration
	vPod.Status = *pPod.Status.DeepCopy()
	vPod.Status.QOSClass = originalQOSClass
	if t.virtualClusterStripsObservedGeneration() {
		vPod.Status.ObservedGeneration = originalObservedGeneration
		vPod.Status.Conditions = stripConditionObservedGenerations(vPod.Status.Conditions)
	}
	err := t.convertResourceClaimStatuses(ctx, vPod, pPod.GetNamespace())
	if err != nil {
		return err
	}

	stripInjectedSidecarContainers(vPod, pPod)

	// get Namespace resource in order to have access to its labels
	vNamespace := &corev1.Namespace{}
	err = t.vClient.Get(ctx, client.ObjectKey{Name: vPod.GetNamespace()}, vNamespace)
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
// - spec.tolerations (can be added not removed)
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

	newTolerations := append([]corev1.Toleration{}, vObj.Spec.Tolerations...)
	for _, hostTol := range pObj.Spec.Tolerations {
		// Carry forward host-only tolerations.
		// If there is a similar tolerations with an different TolerationsSeconds we add the duplicates
		// and let kubernetes decide the one to use.
		if !hasToleration(newTolerations, hostTol) {
			newTolerations = append(newTolerations, hostTol)
		}
	}
	// We add the enforcedTolerations if they are not already present
	for _, toleration := range t.enforcedTolerations {
		if !hasToleration(newTolerations, toleration) {
			newTolerations = append(newTolerations, toleration)
		}
	}
	pObj.Spec.Tolerations = newTolerations
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

func (t *translator) convertResourceClaimStatuses(ctx *synccontext.SyncContext, vPod *corev1.Pod, pNamespace string) error {
	if vPod == nil || !t.resourceClaimEnabled {
		return nil
	}
	for i := range vPod.Status.ResourceClaimStatuses {
		if vPod.Status.ResourceClaimStatuses[i].ResourceClaimName == nil {
			continue
		}
		name := *vPod.Status.ResourceClaimStatuses[i].ResourceClaimName
		nsn := types.NamespacedName{
			Namespace: pNamespace,
			Name:      name,
		}
		resourceClaim := resourcev1.ResourceClaim{}
		err := ctx.HostClient.Get(ctx.Context, nsn, &resourceClaim)
		if err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("can't convert pod resource resourceClaimStatuses to virtual: %w", err)
		}
		translateNsn := mappings.HostToVirtual(
			ctx,
			name,
			resourceClaim.GetNamespace(),
			&resourceClaim,
			mappings.ResourceClaims())
		if translateNsn.Name != "" {
			vPod.Status.ResourceClaimStatuses[i].ResourceClaimName = ptr.To(translateNsn.Name)
		}
	}
	return nil
}

// hasToleration reports whether tol is already present in the slice (full equality).
func hasToleration(tolerations []corev1.Toleration, tol corev1.Toleration) bool {
	for _, t := range tolerations {
		if apiequality.Semantic.DeepEqual(t, tol) {
			return true
		}
	}
	return false
}

// virtualClusterStripsObservedGeneration reports whether the virtual apiserver throws away
// ObservedGeneration when a pod status is written. This happens on K8s < 1.34, where the
// PodObservedGenerationTracking feature gate is off by default.
func (t *translator) virtualClusterStripsObservedGeneration() bool {
	return t.virtualClusterVersion != nil &&
		t.virtualClusterVersion.LessThan(k8sPodObservedGenerationMinVersion)
}

// conditionsCopyBidirectional works like patcher.CopyBidirectional for pod conditions. On
// K8s < 1.34 the virtual cluster does not store ObservedGeneration, so this function ignores that field
// when checking for changes to avoid reacting to a value that was never saved. It does not
// change the conditions, so a real ObservedGeneration from the host is still synced.
func (t *translator) conditionsCopyBidirectional(
	virtualOld, virtual, hostOld, host []corev1.PodCondition,
) (newVirtual, newHost []corev1.PodCondition) {
	if !t.virtualClusterStripsObservedGeneration() {
		return patcher.CopyBidirectional(virtualOld, virtual, hostOld, host)
	}

	return patcher.CopyBidirectionalWithEq(virtualOld, virtual, hostOld, host, func(a, b []corev1.PodCondition) bool {
		return apiequality.Semantic.DeepEqual(
			stripConditionObservedGenerations(a),
			stripConditionObservedGenerations(b),
		)
	})
}

// stripConditionObservedGenerations returns a copy of the conditions with ObservedGeneration
// set to zero on each one. It is only used to compare conditions while ignoring that field.
func stripConditionObservedGenerations(conditions []corev1.PodCondition) []corev1.PodCondition {
	out := slices.Clone(conditions)
	for i := range out {
		out[i].ObservedGeneration = 0
	}
	return out
}
