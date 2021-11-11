package priorityclasses

import (
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *syncer) translate(vObj client.Object) (*schedulingv1.PriorityClass, error) {
	target, err := s.translator.Translate(vObj)
	if err != nil {
		return nil, err
	}

	// translate the priority class
	priorityClass := target.(*schedulingv1.PriorityClass)
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass, nil
}

func (s *syncer) translateUpdate(pObj, vObj *schedulingv1.PriorityClass) *schedulingv1.PriorityClass {
	var updated *schedulingv1.PriorityClass

	// check subsets
	if !equality.Semantic.DeepEqual(vObj.PreemptionPolicy, pObj.PreemptionPolicy) {
		updated = newIfNil(updated, pObj)
		updated.PreemptionPolicy = vObj.PreemptionPolicy
	}

	// check annotations
	updatedAnnotations := s.translator.TranslateAnnotations(vObj, pObj)
	if !equality.Semantic.DeepEqual(updatedAnnotations, pObj.Annotations) {
		updated = newIfNil(updated, pObj)
		updated.Annotations = vObj.Annotations
	}

	// check labels
	updatedLabels := s.translator.TranslateLabels(vObj)
	if !equality.Semantic.DeepEqual(updatedLabels, pObj.Labels) {
		updated = newIfNil(updated, pObj)
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
