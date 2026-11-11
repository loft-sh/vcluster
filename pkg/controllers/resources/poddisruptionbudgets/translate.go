package poddisruptionbudgets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *pdbSyncer) translate(ctx *synccontext.SyncContext, vObj *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
	newPDB := translate.HostMetadata(vObj, s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	newPDB.Status = policyv1.PodDisruptionBudgetStatus{}
	newPDB.Spec.Selector = translate.HostLabelSelector(newPDB.Spec.Selector)
	if err := s.translateBudget(ctx, newPDB, vObj); err != nil {
		return nil, err
	}
	return newPDB, nil
}

func (s *pdbSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *policyv1.PodDisruptionBudget) error {
	pObj.Spec.Selector = translate.HostLabelSelector(vObj.Spec.Selector)
	return s.translateBudget(ctx, pObj, vObj)
}

func needsBudgetConversion(vObj *policyv1.PodDisruptionBudget) bool {
	if vObj.Spec.MaxUnavailable != nil {
		return true
	}
	if vObj.Spec.MinAvailable != nil && vObj.Spec.MinAvailable.Type == intstr.String {
		return true
	}
	return false
}

func (s *pdbSyncer) translateBudget(ctx *synccontext.SyncContext, pObj, vObj *policyv1.PodDisruptionBudget) error {
	if !needsBudgetConversion(vObj) {
		pObj.Spec.MinAvailable = vObj.Spec.DeepCopy().MinAvailable
		pObj.Spec.MaxUnavailable = nil
		return nil
	}

	expectedCount, err := s.resolveExpectedPodCount(ctx, vObj)
	if err != nil {
		return err
	}

	desiredHealthy := 0
	if vObj.Spec.MaxUnavailable != nil {
		maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(vObj.Spec.MaxUnavailable, expectedCount, true)
		if err != nil {
			return fmt.Errorf("calculate maxUnavailable: %w", err)
		}
		desiredHealthy = expectedCount - maxUnavailable
		if desiredHealthy < 0 {
			desiredHealthy = 0
		}
	} else if vObj.Spec.MinAvailable != nil && vObj.Spec.MinAvailable.Type == intstr.String {
		minAvailable, err := intstr.GetScaledValueFromIntOrPercent(vObj.Spec.MinAvailable, expectedCount, true)
		if err != nil {
			return fmt.Errorf("calculate minAvailable: %w", err)
		}
		desiredHealthy = minAvailable
		if desiredHealthy < 0 {
			desiredHealthy = 0
		}
	}

	pObj.Spec.MinAvailable = &intstr.IntOrString{Type: intstr.Int, IntVal: int32(desiredHealthy)}
	pObj.Spec.MaxUnavailable = nil
	return nil
}

func (s *pdbSyncer) resolveExpectedPodCount(ctx *synccontext.SyncContext, vObj *policyv1.PodDisruptionBudget) (int, error) {
	if vObj.Spec.Selector == nil || (len(vObj.Spec.Selector.MatchLabels) == 0 && len(vObj.Spec.Selector.MatchExpressions) == 0) {
		return 0, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(vObj.Spec.Selector)
	if err != nil {
		return 0, fmt.Errorf("invalid label selector: %w", err)
	}

	podList := &corev1.PodList{}
	if err := ctx.VirtualClient.List(ctx, podList, client.InNamespace(vObj.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return 0, fmt.Errorf("list virtual pods: %w", err)
	}

	var activePods []*corev1.Pod
	controllerScales := map[types.UID]int32{}
	visitedControllers := map[types.UID]struct{}{}

	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed || pod.DeletionTimestamp != nil {
			continue
		}
		activePods = append(activePods, pod)

		ctrlRef := metav1.GetControllerOf(pod)
		if ctrlRef == nil {
			continue
		}

		if _, ok := visitedControllers[ctrlRef.UID]; ok {
			continue
		}
		visitedControllers[ctrlRef.UID] = struct{}{}

		switch ctrlRef.Kind {
		case "ReplicaSet":
			rs := &appsv1.ReplicaSet{}
			if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: ctrlRef.Name}, rs); err == nil {
				deployRef := metav1.GetControllerOf(rs)
				if deployRef != nil && deployRef.Kind == "Deployment" {
					deploy := &appsv1.Deployment{}
					if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: deployRef.Name}, deploy); err == nil {
						controllerScales[deploy.UID] = ptr.Deref(deploy.Spec.Replicas, 1)
						continue
					}
				}
				controllerScales[rs.UID] = ptr.Deref(rs.Spec.Replicas, 1)
			}
		case "Deployment":
			deploy := &appsv1.Deployment{}
			if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: ctrlRef.Name}, deploy); err == nil {
				controllerScales[deploy.UID] = ptr.Deref(deploy.Spec.Replicas, 1)
			}
		case "StatefulSet":
			sts := &appsv1.StatefulSet{}
			if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: ctrlRef.Name}, sts); err == nil {
				controllerScales[sts.UID] = ptr.Deref(sts.Spec.Replicas, 1)
			}
		case "ReplicationController":
			rc := &corev1.ReplicationController{}
			if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: ctrlRef.Name}, rc); err == nil {
				controllerScales[rc.UID] = ptr.Deref(rc.Spec.Replicas, 1)
			}
		default:
			u := &unstructured.Unstructured{}
			u.SetAPIVersion(ctrlRef.APIVersion)
			u.SetKind(ctrlRef.Kind)
			if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Namespace: vObj.Namespace, Name: ctrlRef.Name}, u); err == nil {
				replicas, found, _ := unstructured.NestedInt64(u.Object, "spec", "replicas")
				if found {
					controllerScales[u.GetUID()] = int32(replicas)
				}
			}
		}
	}

	if len(controllerScales) == 0 {
		deployments := &appsv1.DeploymentList{}
		if err := ctx.VirtualClient.List(ctx, deployments, client.InNamespace(vObj.Namespace)); err == nil {
			for _, deploy := range deployments.Items {
				if selector.Matches(labels.Set(deploy.Spec.Template.Labels)) {
					controllerScales[deploy.UID] = ptr.Deref(deploy.Spec.Replicas, 1)
				}
			}
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err := ctx.VirtualClient.List(ctx, statefulSets, client.InNamespace(vObj.Namespace)); err == nil {
			for _, sts := range statefulSets.Items {
				if selector.Matches(labels.Set(sts.Spec.Template.Labels)) {
					controllerScales[sts.UID] = ptr.Deref(sts.Spec.Replicas, 1)
				}
			}
		}

		replicaSets := &appsv1.ReplicaSetList{}
		if err := ctx.VirtualClient.List(ctx, replicaSets, client.InNamespace(vObj.Namespace)); err == nil {
			for _, rs := range replicaSets.Items {
				if metav1.GetControllerOf(&rs) != nil {
					continue
				}
				if selector.Matches(labels.Set(rs.Spec.Template.Labels)) {
					controllerScales[rs.UID] = ptr.Deref(rs.Spec.Replicas, 1)
				}
			}
		}
	}

	expectedCount := 0
	for _, scale := range controllerScales {
		if scale > 0 {
			expectedCount += int(scale)
		}
	}

	if expectedCount == 0 && len(controllerScales) == 0 {
		if vObj.Status.ObservedGeneration >= vObj.Generation && vObj.Status.ExpectedPods > 0 {
			expectedCount = int(vObj.Status.ExpectedPods)
		} else {
			expectedCount = len(activePods)
		}
	}

	return expectedCount, nil
}
