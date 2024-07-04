package nodes

import (
	"encoding/json"
	"os"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/util/stringutil"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/klog/v2"
)

var (
	TaintsAnnotation = "vcluster.loft.sh/original-taints"
)

func (s *nodeSyncer) translateUpdateBackwards(pNode *corev1.Node, vNode *corev1.Node) *corev1.Node {
	var updated *corev1.Node

	// merge labels & taints
	translatedSpec := pNode.Spec.DeepCopy()
	labels, annotations := translate.ApplyMetadata(pNode.Annotations, vNode.Annotations, pNode.Labels, vNode.Labels, TaintsAnnotation)

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
	// the env var NODE_NAME is set when only one replica of vcluster is running
	if nodeName, set := os.LookupEnv("NODE_NAME"); set && nodeName == pNode.Name {
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
		newAddresses := []corev1.NodeAddress{}

		if s.fakeKubeletHostnames {
			newAddresses = append(newAddresses, corev1.NodeAddress{
				Address: GetNodeHost(vNode.Name),
				Type:    corev1.NodeHostName,
			})
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
			cpu := translatedStatus.Allocatable.Cpu().MilliValue()
			memory := translatedStatus.Allocatable.Memory().Value()
			storageEphemeral := translatedStatus.Allocatable.StorageEphemeral().Value()
			pods := translatedStatus.Allocatable.Pods().Value()

			var nonVClusterPods int64
			podList := &corev1.PodList{}
			err := s.unmanagedPodCache.List(ctx.Context, podList, client.MatchingFields{constants.IndexRunningNonVClusterPodsByNode: pNode.Name})
			if err != nil {
				klog.Errorf("Error listing pods: %v", err)
			} else {
				for _, pod := range podList.Items {
					if !translate.Default.IsManaged(&pod, translate.Default.PhysicalName) {
						// count pods that are not synced by this vcluster
						nonVClusterPods++
					}
					for _, container := range pod.Spec.InitContainers {
						cpu -= container.Resources.Requests.Cpu().MilliValue()
						memory -= container.Resources.Requests.Memory().Value()
						storageEphemeral -= container.Resources.Requests.StorageEphemeral().Value()
					}
					for _, container := range pod.Spec.Containers {
						cpu -= container.Resources.Requests.Cpu().MilliValue()
						memory -= container.Resources.Requests.Memory().Value()
						storageEphemeral -= container.Resources.Requests.StorageEphemeral().Value()
					}
				}
			}

			pods -= nonVClusterPods
			if pods > 0 {
				translatedStatus.Allocatable[corev1.ResourcePods] = *resource.NewQuantity(pods, resource.DecimalSI)
			}
			if cpu > 0 {
				translatedStatus.Allocatable[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpu, resource.DecimalSI)
			}
			if memory > 0 {
				translatedStatus.Allocatable[corev1.ResourceMemory] = *resource.NewQuantity(memory, resource.BinarySI)
			}
			if storageEphemeral > 0 {
				translatedStatus.Allocatable[corev1.ResourceEphemeralStorage] = *resource.NewQuantity(storageEphemeral, resource.BinarySI)
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
