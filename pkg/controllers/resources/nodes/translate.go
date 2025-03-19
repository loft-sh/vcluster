package nodes

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog/v2"
	resourceutil "k8s.io/kubectl/pkg/util/resource"
)

var (
	TaintsAnnotation                  = "vcluster.loft.sh/original-taints"
	RancherAgentPodRequestsAnnotation = "management.cattle.io/pod-requests"
	RancherAgentPodLimitsAnnotation   = "management.cattle.io/pod-limits"
)

func (s *nodeSyncer) translateUpdateBackwards(pNode *corev1.Node, vNode *corev1.Node) *corev1.Node {
	var updated *corev1.Node

	// merge labels & taints
	translatedSpec := pNode.Spec.DeepCopy()
	excludeAnnotations := []string{TaintsAnnotation, RancherAgentPodRequestsAnnotation, RancherAgentPodLimitsAnnotation}
	labels, annotations := translate.ApplyMetadata(pNode.Annotations, vNode.Annotations, pNode.Labels, vNode.Labels, excludeAnnotations...)

	// merge taints together
	oldPhysical := []string{}
	if vNode.Annotations != nil && vNode.Annotations[TaintsAnnotation] != "" {
		err := json.Unmarshal([]byte(vNode.Annotations[TaintsAnnotation]), &oldPhysical)
		if err != nil {
			klog.Errorf("error decoding taints: %v", err)
		}
	}

	// convert physical taints
	physical := []string{}
	hasUnready := false
	for _, p := range pNode.Spec.Taints {
		if p.Key == "node.kubernetes.io/not-ready" {
			hasUnready = true
		}

		out, err := json.Marshal(p)
		if err != nil {
			klog.Errorf("error encoding taint: %v", err)
		} else {
			physical = append(physical, string(out))
		}
	}

	// convert virtual taints
	virtual := []string{}
	for _, p := range vNode.Spec.Taints {
		if !hasUnready && p.Key == "node.kubernetes.io/not-ready" {
			continue
		}

		out, err := json.Marshal(p)
		if err != nil {
			klog.Errorf("error encoding taint: %v", err)
		} else {
			virtual = append(virtual, string(out))
		}
	}

	// merge taints
	newTaints := mergeStrings(physical, virtual, oldPhysical)
	newTaintsObjects := []corev1.Taint{}
	for _, t := range newTaints {
		taint := corev1.Taint{}
		err := json.Unmarshal([]byte(t), &taint)
		if err != nil {
			klog.Errorf("error decoding taint: %v", err)
		} else {
			newTaintsObjects = append(newTaintsObjects, taint)
		}
	}

	// set merged taints
	translatedSpec.Taints = newTaintsObjects

	// encode taints
	out, err := json.Marshal(physical)
	if err != nil {
		klog.Errorf("error encoding taints: %v", err)
	} else {
		if annotations == nil {
			annotations = map[string]string{}
		}
		if len(physical) > 0 {
			annotations[TaintsAnnotation] = string(out)
		} else {
			delete(annotations, TaintsAnnotation)
		}
	}

	// Omit those taints for which the vcluster has enforced tolerations defined
	if len(s.enforcedTolerations) > 0 && len(translatedSpec.Taints) > 0 {
		translatedSpec.Taints = s.filterOutTaintsMatchingTolerations(translatedSpec.Taints)
	}

	if !equality.Semantic.DeepEqual(vNode.Spec, *translatedSpec) {
		updated = translator.NewIfNil(updated, vNode)
		updated.Spec = *translatedSpec
	}

	// add annotation to prevent scale down of node by cluster-autoscaler
	// the env var VCLUSTER_NODE_NAME is set when only one replica of vcluster is running
	if nodeName, set := os.LookupEnv("VCLUSTER_NODE_NAME"); set && nodeName == pNode.Name {
		annotations["cluster-autoscaler.kubernetes.io/scale-down-disabled"] = "true"
	}

	if !equality.Semantic.DeepEqual(vNode.Annotations, annotations) {
		updated = translator.NewIfNil(updated, vNode)
		updated.Annotations = annotations
	}

	if !equality.Semantic.DeepEqual(vNode.Labels, labels) {
		updated = translator.NewIfNil(updated, vNode)
		updated.Labels = labels
	}

	return updated
}

func (s *nodeSyncer) translateUpdateStatus(ctx *synccontext.SyncContext, pNode *corev1.Node, vNode *corev1.Node) (*corev1.Node, error) {
	// translate node status first
	translatedStatus := pNode.Status.DeepCopy()
	if s.useFakeKubelets {
		translatedStatus.DaemonEndpoints = corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: constants.KubeletPort,
			},
		}

		// translate addresses
		newAddresses := []corev1.NodeAddress{
			{
				Address: GetNodeHost(vNode.Name),
				Type:    corev1.NodeHostName,
			},
		}

		if s.fakeKubeletIPs {
			// create new service for this node
			nodeIP, err := s.nodeServiceProvider.GetNodeIP(ctx.Context, vNode.Name)
			if err != nil {
				return nil, errors.Wrap(err, "get vNode IP")
			}

			newAddresses = append(newAddresses, corev1.NodeAddress{
				Address: nodeIP,
				Type:    corev1.NodeInternalIP,
			})
		}

		for _, oldAddress := range translatedStatus.Addresses {
			if oldAddress.Type == corev1.NodeInternalIP || oldAddress.Type == corev1.NodeInternalDNS || oldAddress.Type == corev1.NodeHostName {
				continue
			}

			newAddresses = append(newAddresses, oldAddress)
		}
		translatedStatus.Addresses = newAddresses
	}

	// if scheduler is enabled we allow custom capacity and allocatable
	if s.enableScheduler {
		// calculate what's really allocatable
		if translatedStatus.Allocatable != nil {
			var (
				nonVClusterPods int64
				allocatable     = map[corev1.ResourceName]int64{
					corev1.ResourceCPU:              translatedStatus.Allocatable.Cpu().MilliValue(),
					corev1.ResourceMemory:           translatedStatus.Allocatable.Memory().Value(),
					corev1.ResourceEphemeralStorage: translatedStatus.Allocatable.StorageEphemeral().Value(),
					corev1.ResourcePods:             translatedStatus.Allocatable.Pods().Value(),
				}
			)

			unmanagedPods := &corev1.PodList{}
			err := s.unmanagedPodCache.List(ctx.Context, unmanagedPods, client.MatchingFields{constants.IndexRunningNonVClusterPodsByNode: pNode.Name})
			if err != nil {
				return nil, fmt.Errorf("error listing unmanaged pods: %w", err)
			}

			for _, pod := range unmanagedPods.Items {
				if !translate.Default.IsManaged(&pod) {
					// count pods that are not synced by this vcluster
					nonVClusterPods++
				}

				reqs, _ := resourceutil.PodRequestsAndLimits(&pod)
				updateAllocatable(reqs, allocatable)
			}

			managedPods := &corev1.PodList{}
			err = s.managedPodCache.List(ctx.Context, managedPods, client.MatchingFields{constants.IndexByAssigned: pNode.Name})
			if err != nil {
				return nil, fmt.Errorf("error listing managed pods: %w", err)
			}

			for _, pod := range managedPods.Items {
				// skip single container pods
				if len(pod.Spec.InitContainers)+len(pod.Spec.Containers) == 1 {
					continue
				}

				reqs, err := s.ManagedPodRequestsAndLimits(ctx, &pod)
				if err != nil {
					return nil, err
				}

				updateAllocatable(reqs, allocatable)
			}

			reservedReqs := corev1.ResourceList{}
			if s.reservedResourceCPU != "" {
				cpu, err := resource.ParseQuantity(s.reservedResourceCPU)
				if err != nil {
					klog.Errorf("error parsing reservedResource cpu: %v", err)
				} else {
					reservedReqs[corev1.ResourceCPU] = cpu
				}
			}
			if s.reservedResourceMemory != "" {
				memory, err := resource.ParseQuantity(s.reservedResourceMemory)
				if err != nil {
					klog.Errorf("error parsing reservedResource memory: %v", err)
				} else {
					reservedReqs[corev1.ResourceMemory] = memory
				}
			}
			if s.reservedResourceEphemeralStorage != "" {
				ephemeralStorage, err := resource.ParseQuantity(s.reservedResourceEphemeralStorage)
				if err != nil {
					klog.Errorf("error parsing reservedResource ephemeral storage: %v", err)
				} else {
					reservedReqs[corev1.ResourceEphemeralStorage] = ephemeralStorage
				}
			}

			updateAllocatable(reservedReqs, allocatable)

			allocatable[corev1.ResourcePods] -= nonVClusterPods
			if allocatable[corev1.ResourcePods] > 0 {
				translatedStatus.Allocatable[corev1.ResourcePods] = *resource.NewQuantity(allocatable[corev1.ResourcePods], resource.DecimalSI)
			}
			if allocatable[corev1.ResourceCPU] > 0 {
				translatedStatus.Allocatable[corev1.ResourceCPU] = *resource.NewMilliQuantity(allocatable[corev1.ResourceCPU], resource.DecimalSI)
			}
			if allocatable[corev1.ResourceMemory] > 0 {
				translatedStatus.Allocatable[corev1.ResourceMemory] = *resource.NewQuantity(allocatable[corev1.ResourceMemory], resource.BinarySI)
			}
			if allocatable[corev1.ResourceEphemeralStorage] > 0 {
				translatedStatus.Allocatable[corev1.ResourceEphemeralStorage] = *resource.NewQuantity(allocatable[corev1.ResourceEphemeralStorage], resource.BinarySI)
			}
		}

		// calculate what's in capacity & allocatable
		capacity := mergeResources(vNode.Status.Capacity, translatedStatus.Capacity)
		if len(capacity) > 0 {
			translatedStatus.Capacity = capacity
		}

		// allocatable
		allocatable := mergeResources(vNode.Status.Allocatable, translatedStatus.Allocatable)
		if len(allocatable) > 0 {
			translatedStatus.Allocatable = allocatable
		}

		if translatedStatus.Capacity == nil {
			translatedStatus.Capacity = corev1.ResourceList{}
		}
		if translatedStatus.Allocatable == nil {
			translatedStatus.Allocatable = corev1.ResourceList{}
		}
		for k := range translatedStatus.Allocatable {
			_, found := translatedStatus.Capacity[k]
			if !found {
				delete(translatedStatus.Allocatable, k)
			}
		}
		for k := range translatedStatus.Capacity {
			_, found := translatedStatus.Allocatable[k]
			if !found {
				translatedStatus.Allocatable[k] = translatedStatus.Capacity[k]
			}
		}
	}

	if s.clearImages {
		translatedStatus.Images = make([]corev1.ContainerImage, 0)
	}

	// check if the status has changed
	if !equality.Semantic.DeepEqual(vNode.Status, *translatedStatus) {
		newNode := vNode.DeepCopy()
		newNode.Status = *translatedStatus
		return newNode, nil
	}

	return nil, nil
}

func mergeStrings(physical []string, virtual []string, oldPhysical []string) []string {
	merged := []string{}
	merged = append(merged, physical...)
	merged = append(merged, virtual...)
	merged = stringutil.RemoveDuplicates(merged)
	newMerged := []string{}
	for _, o := range merged {
		if stringutil.Contains(oldPhysical, o) && !stringutil.Contains(physical, o) {
			continue
		}

		newMerged = append(newMerged, o)
	}

	return newMerged
}

func mergeResources(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
	merged := corev1.ResourceList{}
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func (s *nodeSyncer) filterOutTaintsMatchingTolerations(taints []corev1.Taint) []corev1.Taint {
	var filtered []corev1.Taint

nextTaint:
	for _, taint := range taints {
		for _, tol := range s.enforcedTolerations {
			// Special case
			// An empty key with operator Exists matches all keys,
			// values and effects which means this will tolerate everything.
			if tol.Key == "" && tol.Operator == corev1.TolerationOpExists {
				return nil
			}

			if taint.Key == tol.Key && (taint.Effect == tol.Effect || tol.Effect == corev1.TaintEffect("")) {
				if tol.Operator == corev1.TolerationOpExists ||
					(tol.Operator == corev1.TolerationOpEqual && taint.Value == tol.Value) {
					// taint matched the current toleration, skip to next taint
					continue nextTaint
				}
			}
		}

		// taint did not match any enforced toleration, keep it
		filtered = append(filtered, taint)
	}

	return filtered
}

func (s *nodeSyncer) ManagedPodRequestsAndLimits(ctx *synccontext.SyncContext, pod *corev1.Pod) (corev1.ResourceList, error) {
	reqs := corev1.ResourceList{}

	virtualPod := &corev1.Pod{}
	err := ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{Namespace: pod.DeepCopy().Annotations[translate.NamespaceAnnotation], Name: pod.DeepCopy().Annotations[translate.NameAnnotation]}, virtualPod)
	if err != nil {
		return reqs, err
	}

	virtualPodContainerStatuses := map[string]struct{}{}
	for _, cs := range [][]corev1.ContainerStatus{virtualPod.Status.ContainerStatuses, virtualPod.Status.InitContainerStatuses} {
		for i := range cs {
			virtualPodContainerStatuses[cs[i].Name] = struct{}{}
		}
	}

	podContainerStatuses := map[string]corev1.ContainerStatus{}
	for _, cs := range [][]corev1.ContainerStatus{pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses} {
		for i := range cs {
			podContainerStatuses[cs[i].Name] = cs[i]
		}
	}

	for _, containers := range [][]corev1.Container{pod.Spec.Containers, pod.Spec.InitContainers} {
		for _, container := range containers {
			containerReqs := container.Resources.Requests
			_, found := virtualPodContainerStatuses[container.Name]
			if found {
				continue
			}

			if cs, ok := podContainerStatuses[container.Name]; ok {
				if cs.State.Running == nil {
					continue
				}

				if pod.Status.Resize == corev1.PodResizeStatusInfeasible {
					containerReqs = cs.AllocatedResources.DeepCopy()
				} else {
					containerReqs = maxOf(container.Resources.Requests, cs.AllocatedResources)
				}
			}

			addResourceList(reqs, containerReqs)
		}
	}

	return reqs, nil
}

// addResourceList adds the resources in newList to list
func addResourceList(list, newList corev1.ResourceList) {
	for name, quantity := range newList {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}
}

// maxOf returns the result of maxOf(a, b) for each named resource and is only used if we can't
// accumulate into an existing resource list
func maxOf(a corev1.ResourceList, b corev1.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	for key, value := range a {
		if other, found := b[key]; found {
			if value.Cmp(other) <= 0 {
				result[key] = other.DeepCopy()
				continue
			}
		}
		result[key] = value.DeepCopy()
	}
	for key, value := range b {
		if _, found := result[key]; !found {
			result[key] = value.DeepCopy()
		}
	}
	return result
}

func updateAllocatable(reqs corev1.ResourceList, allocatable map[corev1.ResourceName]int64) {
	for _, resName := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage} {
		if req, ok := reqs[resName]; ok {
			if resName == corev1.ResourceCPU {
				allocatable[resName] -= req.MilliValue()
			} else {
				allocatable[resName] -= req.Value()
			}
		}
	}
}
