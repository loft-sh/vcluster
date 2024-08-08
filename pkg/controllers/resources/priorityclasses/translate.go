package priorityclasses

import (
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
	targetObject := event.TargetObject()
	sourceObject := event.SourceObject()
	pObj := event.Host
	vObj := event.Virtual

	targetObject.PreemptionPolicy = sourceObject.PreemptionPolicy
	targetObject.Description = sourceObject.Description

	switch event.Source {
	case synccontext.SyncEventSourceVirtual:
		// check metadata
		pObj.Annotations = translate.HostAnnotations(vObj, pObj)
		pObj.Labels = translate.HostLabels(vObj, pObj)
		translatedValue := vObj.Value
		if translatedValue > 1000000000 {
			translatedValue = 1000000000
		}
		pObj.Value = translatedValue
	case synccontext.SyncEventSourceHost:
		vObj.Annotations = translate.VirtualAnnotations(pObj, vObj)
		vObj.Labels = translate.VirtualLabels(pObj, vObj)
	}
}
