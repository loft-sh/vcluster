package deploy

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KObject struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

func (k *KObject) Equals(obj unstructured.Unstructured) bool {
	if k.APIVersion == obj.GetAPIVersion() &&
		k.Kind == obj.GetKind() &&
		k.Namespace == obj.GetNamespace() &&
		k.Name == obj.GetName() {
		return true
	}

	return false
}

type UnstructuredMap map[KObject]*unstructured.Unstructured

var diffSeparator = regexp.MustCompile(`\n---`)

func ManifestStringToUnstructuredArray(out, defaultNamespace string) ([]*unstructured.Unstructured, error) {
	if strings.TrimSpace(out) == "" || strings.TrimSpace(out) == "---" {
		return nil, nil
	}
	parts := diffSeparator.Split(out, -1)
	var objs []*unstructured.Unstructured
	var firstErr error
	for _, part := range parts {
		var objMap map[string]interface{}
		err := yaml.Unmarshal([]byte(part), &objMap)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to unmarshal manifest: %w", err)
			}
			continue
		}
		if len(objMap) == 0 {
			// handles case where there's no content between `---`
			continue
		}
		var obj unstructured.Unstructured
		err = yaml.Unmarshal([]byte(part), &obj)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to unmarshal manifest: %w", err)
			}
			continue
		}
		if obj.GetNamespace() == "" {
			obj.SetNamespace(defaultNamespace)
		}
		objs = append(objs, &obj)
	}
	return objs, firstErr
}

func UnstructuredToKObject(obj unstructured.Unstructured) KObject {
	return KObject{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Namespace:  obj.GetNamespace(),
		Name:       obj.GetName(),
	}
}
