package nodes

import (
	"context"
	"encoding/json"
	"github.com/loft-sh/vcluster/pkg/util/stringutil"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

var (
	TaintsAnnotation = "vcluster.loft.sh/original-taints"
)

func (s *nodeSyncer) translateUpdateBackwards(pNode *corev1.Node, vNode *corev1.Node) *corev1.Node {
	var updated *corev1.Node

	var (
		annotations    map[string]string
		labels         map[string]string
		translatedSpec = pNode.Spec.DeepCopy()
	)
	if s.enableScheduler {
		annotations = mergeStringMap(vNode.Annotations, pNode.Annotations)
		labels = mergeStringMap(vNode.Labels, pNode.Labels)

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
		for _, p := range pNode.Spec.Taints {
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
		translatedSpec.Taints = newTaintsObjects

		// encode taints
		out, err := json.Marshal(physical)
		if err != nil {
			klog.Errorf("error encoding taints: %v", err)
		} else {
			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations[TaintsAnnotation] = string(out)
		}
	} else {
		annotations = pNode.Annotations
		labels = pNode.Labels
	}

	if !equality.Semantic.DeepEqual(vNode.Spec, *translatedSpec) {
		updated = newIfNil(updated, vNode)
		updated.Spec = *translatedSpec
	}

	if !equality.Semantic.DeepEqual(vNode.Annotations, annotations) {
		updated = newIfNil(updated, vNode)
		updated.Annotations = annotations
	}

	if !equality.Semantic.DeepEqual(vNode.Labels, labels) {
		updated = newIfNil(updated, vNode)
		updated.Labels = labels
	}

	return updated
}

func (s *nodeSyncer) translateUpdateStatus(ctx *synccontext.SyncContext, pNode *corev1.Node, vNode *corev1.Node) (*corev1.Node, error) {
	// translate node status first
	translatedStatus := pNode.Status.DeepCopy()
	if s.useFakeKubelets {
		s.nodeServiceProvider.Lock()
		defer s.nodeServiceProvider.Unlock()
		translatedStatus.DaemonEndpoints = corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: nodeservice.KubeletPort,
			},
		}

		// translate addresses
		// create a new service for this node
		nodeIP, err := s.nodeServiceProvider.GetNodeIP(ctx.Context, types.NamespacedName{Name: vNode.Name})
		if err != nil {
			return nil, errors.Wrap(err, "get vNode IP")
		}
		newAddresses := []corev1.NodeAddress{
			{
				Address: nodeIP,
				Type:    corev1.NodeInternalIP,
			},
		}
		for _, oldAddress := range translatedStatus.Addresses {
			if oldAddress.Type == corev1.NodeInternalIP || oldAddress.Type == corev1.NodeInternalDNS || oldAddress.Type == corev1.NodeHostName {
				continue
			}

			newAddresses = append(newAddresses, oldAddress)
		}
		translatedStatus.Addresses = newAddresses
	}

	// calculate what's really allocatable
	if translatedStatus.Allocatable != nil && translatedStatus.Capacity != nil {
		cpu := translatedStatus.Capacity.Cpu().MilliValue()
		memory := translatedStatus.Capacity.Memory().Value()
		storageEphemeral := translatedStatus.Capacity.StorageEphemeral().Value()

		podList := &corev1.PodList{}
		err := s.podCache.List(context.TODO(), podList)
		if err != nil {
			klog.Errorf("Error listing pods: %v", err)
		} else {
			for _, pod := range podList.Items {
				if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					continue
				} else if pod.Spec.NodeName != pNode.Name {
					continue
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

	// if scheduler is enabled we allow custom capacity and allocatable
	if s.enableScheduler {
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

	// check if the status has changed
	if !equality.Semantic.DeepEqual(vNode.Status, *translatedStatus) {
		newNode := vNode.DeepCopy()
		newNode.Status = *translatedStatus
		return newNode, nil
	}

	return nil, nil
}

func newIfNil(updated *corev1.Node, pObj *corev1.Node) *corev1.Node {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
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

func mergeStringMap(a map[string]string, b map[string]string) map[string]string {
	merged := map[string]string{}
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
