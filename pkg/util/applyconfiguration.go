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
