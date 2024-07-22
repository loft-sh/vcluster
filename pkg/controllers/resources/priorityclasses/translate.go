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
	priorityClass := translate.HostMetadata(ctx, vObj.(*schedulingv1.PriorityClass), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj, sourceObject, targetObject *schedulingv1.PriorityClass) {
	targetObject.PreemptionPolicy = sourceObject.PreemptionPolicy
	targetObject.Description = sourceObject.Description

	// check metadata
	pObj.Annotations = translate.HostAnnotations(vObj, pObj)
	pObj.Labels = translate.HostLabels(ctx, vObj, pObj)

	// check value
	translatedValue := vObj.Value
	if translatedValue > 1000000000 {
		translatedValue = 1000000000
	}
	pObj.Value = translatedValue
}
