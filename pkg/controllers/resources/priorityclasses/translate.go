package priorityclasses

import (
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *priorityClassSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *schedulingv1.PriorityClass {
	// translate the priority class
	priorityClass := translate.HostMetadata(vObj.(*schedulingv1.PriorityClass), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateFromHost(ctx *synccontext.SyncContext, pObj client.Object) *schedulingv1.PriorityClass {
	// translate the priority class
	priorityClass := translate.VirtualMetadata(pObj.(*schedulingv1.PriorityClass), s.HostToVirtual(ctx, types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, pObj))
	if priorityClass.Name == "" {
		priorityClass.Name = pObj.GetName()
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateUpdate(event *synccontext.SyncEvent[*schedulingv1.PriorityClass]) {
	if s.fromHost {
		event.Virtual.PreemptionPolicy = event.Host.PreemptionPolicy
		event.Virtual.Description = event.Host.Description
		event.Virtual.Annotations = event.Host.Annotations
		event.Virtual.Labels = event.Host.Labels
	} else if s.toHost {
		// bi-directional
		event.Virtual.PreemptionPolicy, event.Host.PreemptionPolicy = patcher.CopyBidirectional(
			event.VirtualOld.PreemptionPolicy,
			event.Virtual.PreemptionPolicy,
			event.HostOld.PreemptionPolicy,
			event.Host.PreemptionPolicy,
		)
		event.Virtual.Description, event.Host.Description = patcher.CopyBidirectional(
			event.VirtualOld.Description,
			event.Virtual.Description,
			event.HostOld.Description,
			event.Host.Description,
		)
		event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
		event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

		// copy from virtual -> host
		translatedValue := event.Virtual.Value
		if translatedValue > 1000000000 {
			translatedValue = 1000000000
		}
		event.Host.Value = translatedValue
	}
}
