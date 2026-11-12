package nodes

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	resourceutil "k8s.io/component-helpers/resource"
	"k8s.io/klog/v2"
)

var (
	TaintsAnnotation                  = "vcluster.loft.sh/original-taints"
	RancherAgentPodRequestsAnnotation = "management.cattle.io/pod-requests"
	RancherAgentPodLimitsAnnotation   = "management.cattle.io/pod-limits"
)

func (s *nodeSyncer) translateUpdateBackwards(pNode *corev1.Node, vNode *corev1.Node) {
	// merge labels & taints
	translatedSpec := pNode.Spec.DeepCopy()
	excludeAnnotations := []string{TaintsAnnotation, RancherAgentPodRequestsAnnotation, RancherAgentPodLimitsAnnotation}
	labels, annotations := translate.ApplyMetadata(pNode.Annotations, vNode.Annotations, pNode.Labels, vNode.Labels, excludeAnnotations...)

	// Merge by key and effect. The host copy wins, including value and timeAdded.
	// translatedSpec is a deep copy, so the kept host taints do not alias pNode.
	translatedSpec.Taints = mergeTaints(translatedSpec.Taints, vNode.Spec.Taints, originalTaints(vNode.Annotations))

	// Store the current host taints as a JSON array of taint JSON strings.
	physical := []string{}
	for _, p := range pNode.Spec.Taints {
		out, err := json.Marshal(p)
		if err != nil {
			klog.Errorf("error encoding taint: %v", err)
		} else {
			physical = append(physical, string(out))
		}
	}

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

	// add annotation to prevent scale down of node by cluster-autoscaler
	// the env var NODE_NAME is set when only one replica of vcluster is running
	if nodeName, set := os.LookupEnv("NODE_NAME"); set && nodeName == pNode.Name {
		annotations["cluster-autoscaler.kubernetes.io/scale-down-disabled"] = "true"
	}

	// set annotations, spec & labels
	vNode.Spec = *translatedSpec
	vNode.Annotations = annotations
	vNode.Labels = labels
}

// translateUpdateStatus translates the node's status.
// Returns a Node object, a boolean indicating whether it changed or an error.
func (s *nodeSyncer) translateUpdateStatus(ctx *synccontext.SyncContext, pNode *corev1.Node, vNode *corev1.Node) error {
	// translate node status first
	translatedStatus := pNode.Status.DeepCopy()
	if s.useFakeKubelets {
		translatedStatus.DaemonEndpoints = corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: constants.KubeletPort,
			},
		}

		// translate addresses
		newAddresses := []corev1.NodeAddress{}

		if s.fakeKubeletHostnames {
			newAddresses = append(newAddresses, corev1.NodeAddress{
				Address: GetNodeHost(vNode.Name),
				Type:    corev1.NodeHostName,
			})
		}

		if s.fakeKubeletIPs {
			// create new service for this node
			nodeIP, err := s.nodeServiceProvider.GetNodeIP(ctx, vNode.Name)
			if err != nil {
				return fmt.Errorf("get vNode IP: %w", err)
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
			allocatable := map[corev1.ResourceName]int64{
				corev1.ResourceCPU:              translatedStatus.Allocatable.Cpu().MilliValue(),
				corev1.ResourceMemory:           translatedStatus.Allocatable.Memory().Value(),
				corev1.ResourceEphemeralStorage: translatedStatus.Allocatable.StorageEphemeral().Value(),
				corev1.ResourcePods:             translatedStatus.Allocatable.Pods().Value(),
			}

			var nonVClusterPods int64
			podList := &corev1.PodList{}
			err := s.unmanagedPodCache.List(ctx, podList, client.MatchingFields{constants.IndexRunningNonVClusterPodsByNode: pNode.Name})
			if err != nil {
				klog.Errorf("Error listing pods: %v", err)
			} else {
				for _, pod := range podList.Items {
					isManaged, err := s.IsManaged(ctx, &pod)
					if err != nil {
						klog.FromContext(ctx).Error(err, "is pod managed")
					}

					// check if managed
					if !isManaged {
						// count pods that are not synced by this vcluster
						nonVClusterPods++
					}

					reqs := resourceutil.PodRequests(&pod, resourceutil.PodResourcesOptions{
						UseStatusResources: true,
						InPlacePodLevelResourcesVerticalScalingEnabled: true, // Feature gate is in beta and enabled by default starting with k8s 1.36: https://github.com/kubernetes/enhancements/issues/5419
						SkipPodLevelResources:                          false,
					})

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
			}

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

	vNode.Status = *translatedStatus
	return nil
}

// taintID is the identity Kubernetes uses for a taint. timeAdded and value are not part of it.
type taintID struct {
	key    string
	effect corev1.TaintEffect
}

// mergeTaints keeps every host taint, then virtual taints whose key and effect are still new.
// Virtual not-ready and unreachable taints are dropped when the host has no taint with that key.
// A virtual taint that was recorded from the host and is gone from the host is dropped too.
func mergeTaints(physical, virtual, oldPhysical []corev1.Taint) []corev1.Taint {
	merged := []corev1.Taint{}
	seen := make(map[taintID]struct{}, len(physical)+len(virtual))
	physicalKeys := make(map[string]struct{}, len(physical))
	for _, taint := range physical {
		id := taintID{key: taint.Key, effect: taint.Effect}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		physicalKeys[taint.Key] = struct{}{}
		merged = append(merged, taint)
	}

	old := make(map[taintID]struct{}, len(oldPhysical))
	for _, taint := range oldPhysical {
		old[taintID{key: taint.Key, effect: taint.Effect}] = struct{}{}
	}

	for _, taint := range virtual {
		id := taintID{key: taint.Key, effect: taint.Effect}
		if _, ok := seen[id]; ok {
			continue
		}
		if dropVirtualLifecycleTaint(taint.Key, physicalKeys) {
			continue
		}
		// The host used to have this taint and has since removed it.
		if _, ok := old[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, taint)
	}
	return merged
}

// dropVirtualLifecycleTaint reports whether a virtual node lifecycle taint should be
// omitted because the host node does not currently have that key.
func dropVirtualLifecycleTaint(key string, physicalKeys map[string]struct{}) bool {
	switch key {
	case corev1.TaintNodeNotReady, corev1.TaintNodeUnreachable:
		_, present := physicalKeys[key]
		return !present
	default:
		return false
	}
}

// originalTaints reads vcluster.loft.sh/original-taints. The value is a JSON array of
// taint JSON strings. Identity checks use key and effect, not the raw string.
func originalTaints(annotations map[string]string) []corev1.Taint {
	if annotations == nil || annotations[TaintsAnnotation] == "" {
		return nil
	}

	encoded := []string{}
	err := json.Unmarshal([]byte(annotations[TaintsAnnotation]), &encoded)
	if err != nil {
		klog.Errorf("error decoding taints: %v", err)
		return nil
	}

	taints := []corev1.Taint{}
	for _, raw := range encoded {
		taint := corev1.Taint{}
		err = json.Unmarshal([]byte(raw), &taint)
		if err != nil {
			klog.Errorf("error decoding taint: %v", err)
			continue
		}
		taints = append(taints, taint)
	}
	return taints
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
