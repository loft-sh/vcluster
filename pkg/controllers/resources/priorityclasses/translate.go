package priorityclasses

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *priorityClassSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *schedulingv1.PriorityClass {
	// translate the priority class
	priorityClass := s.TranslateMetadata(ctx, vObj).(*schedulingv1.PriorityClass)
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj, sourceObject, targetObject *schedulingv1.PriorityClass) {
	// check subsets
	if !equality.Semantic.DeepEqual(vObj.PreemptionPolicy, pObj.PreemptionPolicy) {
		targetObject.PreemptionPolicy = sourceObject.PreemptionPolicy
	}

	// check description
	if vObj.Description != pObj.Description {
		targetObject.Description = sourceObject.Description
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		pObj.Annotations = updatedAnnotations
		pObj.Labels = updatedLabels
	}

	// check value
	translatedValue := vObj.Value
	if translatedValue > 1000000000 {
		translatedValue = 1000000000
	}
	if translatedValue != pObj.Value {
		pObj.Value = translatedValue
	}
}
