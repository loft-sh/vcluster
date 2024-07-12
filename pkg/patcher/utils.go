package patcher

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	alwaysIncludedKeys = map[string]bool{
		"kind":       true,
		"apiVersion": true,
		"metadata":   true,
	}

	statusKey = map[string]bool{
		"status": true,
	}
)

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
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
	return &unstructured.Unstructured{Object: rawMap}, nil
}

// unsafeUnstructuredCopy returns a shallow copy of the unstructured object given as input.
// It copies the common fields such as `kind`, `apiVersion`, `metadata` and the patchType specified.
//
// It's not safe to modify any of the keys in the returned unstructured object, the result should be treated as read-only.
func unsafeUnstructuredCopy(obj *unstructured.Unstructured, include, exclude map[string]bool) *unstructured.Unstructured {
	// Create the return focused-unstructured object with a preallocated map.
	res := &unstructured.Unstructured{Object: make(map[string]interface{}, len(obj.Object))}

	// Ranges over the keys of the unstructured object, think of this as the very top level of an object
	// when submitting a yaml to kubectl or a client.
	// These would be keys like `apiVersion`, `kind`, `metadata`, `spec`, `status`, etc.
	for key := range obj.Object {
		value := obj.Object[key]

		// check if key should be always included
		if alwaysIncludedKeys[key] {
			res.Object[key] = value
			continue
		}

		// exclude
		if len(exclude) > 0 && exclude[key] {
			continue
		}

		// include
		if len(include) > 0 && !include[key] {
			continue
		}

		res.Object[key] = value
	}

	return res
}
