package util

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtractClientObjectFromApplyConfiguration converts ApplyConfiguration to client.Object for efficient metadata access.
// It returns an unstructured.Unstructured which implements client.Object
func ExtractClientObjectFromApplyConfiguration(obj runtime.ApplyConfiguration) (client.Object, error) {
	// Use DefaultUnstructuredConverter to convert the object to a map
	// This handles all ApplyConfiguration types, including unstructured ones
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	unstructuredObj := &unstructured.Unstructured{Object: content}
	if unstructuredObj.GroupVersionKind().Empty() {
		return nil, fmt.Errorf("could not extract GVK from ApplyConfiguration")
	}
	return unstructuredObj, nil
}

// MergeClientObjectIntoApplyConfiguration writes the state of the mutated client object
// back into the given ApplyConfiguration so that an Apply call sends the mutated content.
// Use this after MutateObject so that plugin hook mutations (e.g. labels/annotations)
// are applied to the server instead of being discarded.
func MergeClientObjectIntoApplyConfiguration(clientObj client.Object, applyConfig runtime.ApplyConfiguration) error {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clientObj)
	if err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(content, applyConfig)
}
