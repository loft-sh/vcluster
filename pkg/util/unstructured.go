package util

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
)

type KObject struct {
	ApiVersion string
	Kind       string
	Namespace  string
	Name       string
}

func (k *KObject) Equals(obj unstructured.Unstructured) bool {
	if k.ApiVersion == obj.GetAPIVersion() &&
		k.Kind == obj.GetKind() &&
		k.Namespace == obj.GetNamespace() &&
		k.Name == obj.GetName() {
		return true
	}

	return false
}

type UnstructuredMap map[KObject]*unstructured.Unstructured

func ManifestStringToUnstructureArray(rawManifests, defaultNamespace string) ([]*unstructured.Unstructured, error) {
	// check for empty manifest
	if rawManifests == "---" {
		return nil, nil
	}

	manifests := strings.Split(rawManifests, "---")

	var objs []*unstructured.Unstructured
	klog.Infof("got %d raw manifests objects to be converted to unstructured objects", len(manifests))

	for _, manifest := range manifests {
		var obj unstructured.Unstructured

		err := yaml.Unmarshal([]byte(manifest), &obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal manifest: %v", err)
		}

		klog.Infof("successfully converted raw manifest obj to unstructured object: [%s:%s:%s]", obj.GetAPIVersion(), obj.GetKind(), obj.GetName())
		if len(obj.Object) == 0 {
			klog.Infof("object len %d, hence skipping: %v", len(obj.Object), obj)
			continue
		}

		if obj.GetNamespace() == "" {
			klog.Infof("object %s namespace empty setting to default namespace %s", obj.GetName(), defaultNamespace)
			obj.SetNamespace(defaultNamespace)
		}

		objs = append(objs, &obj)
	}

	klog.Infof("returning a total of %d unstructured objects to be applied", len(objs))

	return objs, nil
}

func UnstructuredToKObject(obj unstructured.Unstructured) KObject {
	return KObject{
		ApiVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Namespace:  obj.GetKind(),
		Name:       obj.GetName(),
	}
}
