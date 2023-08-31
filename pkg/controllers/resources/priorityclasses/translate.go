package priorityclasses

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *priorityClassSyncer) translate(ctx context.Context, vObj client.Object) *schedulingv1.PriorityClass {
	// translate the priority class
	priorityClass := s.TranslateMetadata(ctx, vObj).(*schedulingv1.PriorityClass)
	priorityClass.GlobalDefault = false
	if priorityClass.Value > 1000000000 {
		priorityClass.Value = 1000000000
	}
	return priorityClass
}

func (s *priorityClassSyncer) translateUpdate(ctx context.Context, pObj, vObj *schedulingv1.PriorityClass) *schedulingv1.PriorityClass {
	var updated *schedulingv1.PriorityClass

	// check subsets
	if !equality.Semantic.DeepEqual(vObj.PreemptionPolicy, pObj.PreemptionPolicy) {
		updated = translator.NewIfNil(updated, pObj)
		updated.PreemptionPolicy = vObj.PreemptionPolicy
	}

	// check annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	// check description
	if vObj.Description != pObj.Description {
		updated = translator.NewIfNil(updated, pObj)
		updated.Description = vObj.Description
	}

	// check value
	translatedValue := vObj.Value
	if translatedValue > 1000000000 {
		translatedValue = 1000000000
	}
	if translatedValue != pObj.Value {
		updated = translator.NewIfNil(updated, pObj)
		updated.Value = translatedValue
	}

	return updated
}
