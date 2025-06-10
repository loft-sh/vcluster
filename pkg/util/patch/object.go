package patch

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConvertPatchToObject(fromMap map[string]interface{}, toObj client.Object) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(fromMap, toObj)
}

func ConvertObjectToPatch(obj runtime.Object) (Patch, error) {
	// If the incoming object is already unstructured, perform a deep copy first
	// otherwise DefaultUnstructuredConverter ends up returning the inner map without
	// making a copy.
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject()
	}
	rawMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return rawMap, nil
}

func CalculateMergePatch(beforeObject, afterObject client.Object) (Patch, error) {
	patchBytes, err := client.MergeFrom(beforeObject).Data(afterObject)
	if err != nil {
		return nil, fmt.Errorf("calculate patch: %w", err)
	}

	// Unmarshal patch data into a local map.
	patchDiff := map[string]interface{}{}
	if err := json.Unmarshal(patchBytes, &patchDiff); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patch data into a map: %w", err)
	}

	patchObject := Patch(patchDiff)
	patchObject.DeleteAllExcept("metadata", "annotations", "labels", "finalizers")
	return patchObject, nil
}
