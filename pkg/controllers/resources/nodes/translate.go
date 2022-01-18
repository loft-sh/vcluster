package nodes

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

func (s *nodeSyncer) translateUpdateBackwards(pNode *corev1.Node, vNode *corev1.Node) *corev1.Node {
	var updated *corev1.Node

	if !equality.Semantic.DeepEqual(vNode.Spec, pNode.Spec) {
		updated = newIfNil(updated, vNode)
		updated.Spec = pNode.Spec
	}

	if !equality.Semantic.DeepEqual(vNode.Annotations, pNode.Annotations) {
		updated = newIfNil(updated, vNode)
		updated.Annotations = pNode.Annotations
	}

	if !equality.Semantic.DeepEqual(vNode.Labels, pNode.Labels) {
		updated = newIfNil(updated, vNode)
		updated.Labels = pNode.Labels
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

	// calculate whats really allocatable
	if translatedStatus.Allocatable != nil {
		cpu := translatedStatus.Allocatable.Cpu().MilliValue()
		memory := translatedStatus.Allocatable.Memory().Value()
		storageEphemeral := translatedStatus.Allocatable.StorageEphemeral().Value()

		podList := &corev1.PodList{}
		err := s.podCache.List(context.TODO(), podList)
		if err != nil {
			klog.Errorf("Error listing pods: %v", err)
		} else {
			for _, pod := range podList.Items {
				if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					continue
				} else if pod.Labels != nil && pod.Labels[translate.MarkerLabel] == translate.Suffix {
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
