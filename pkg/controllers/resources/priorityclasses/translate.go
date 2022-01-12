package priorityclasses

import (
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *priorityClassSyncer) translate(vObj client.Object) *schedulingv1.PriorityClass {
	// translate the priority class
	priorityClass := s.TranslateMetadata(vObj).(*schedulingv1.PriorityClass)
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateUpdate(pObj, vObj *schedulingv1.PriorityClass) *schedulingv1.PriorityClass {
	var updated *schedulingv1.PriorityClass

	// check subsets
	if !equality.Semantic.DeepEqual(vObj.PreemptionPolicy, pObj.PreemptionPolicy) {
		updated = newIfNil(updated, pObj)
		updated.PreemptionPolicy = vObj.PreemptionPolicy
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check description
	if vObj.Description != pObj.Description {
		updated = newIfNil(updated, pObj)
		updated.Description = vObj.Description
	}

	// check value
	translatedValue := vObj.Value
	if translatedValue > 1000000000 {
		translatedValue = 1000000000
	}
	if translatedValue != pObj.Value {
		updated = newIfNil(updated, pObj)
		updated.Value = translatedValue
	}

	return updated
}

func newIfNil(updated *schedulingv1.PriorityClass, obj *schedulingv1.PriorityClass) *schedulingv1.PriorityClass {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
