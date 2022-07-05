package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	NamespaceLabel  = "vcluster.loft.sh/namespace"
	MarkerLabel     = "vcluster.loft.sh/managed-by"
	ControllerLabel = "vcluster.loft.sh/controlled-by"
	Suffix          = "suffix"

	ManagedAnnotationsAnnotation = "vcluster.loft.sh/managed-annotations"
	ManagedLabelsAnnotation      = "vcluster.loft.sh/managed-labels"
)

var Owner client.Object

func SafeConcatGenerateName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 53 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:42]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-")
	}
	return fullPath
}

func SafeConcatName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 63 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.ReplaceAll(fullPath[0:52]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-")
	}
	return fullPath
}

func GetOwnerReference(object client.Object) []metav1.OwnerReference {
	if Owner == nil || Owner.GetName() == "" || Owner.GetUID() == "" {
		return nil
	}

	typeAccessor, err := meta.TypeAccessor(Owner)
	if err != nil || typeAccessor.GetAPIVersion() == "" || typeAccessor.GetKind() == "" {
		return nil
	}

	isController := false
	if object != nil {
		ctrl := metav1.GetControllerOf(object)
		isController = ctrl != nil
	}
	return []metav1.OwnerReference{
		{
			APIVersion: typeAccessor.GetAPIVersion(),
			Kind:       typeAccessor.GetKind(),
			Name:       Owner.GetName(),
			UID:        Owner.GetUID(),
			Controller: &isController,
		},
	}
}

func IsManaged(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == Suffix
}

func IsManagedCluster(physicalNamespace string, obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == SafeConcatName(physicalNamespace, "x", Suffix)
}

// PhysicalName returns the physical name of the name / namespace resource
func PhysicalName(name, namespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", Suffix)
}

func ObjectPhysicalName(obj client.Object) string {
	return PhysicalName(obj.GetName(), obj.GetNamespace())
}

// PhysicalNameClusterScoped returns the physical name of a cluster scoped object in the host cluster
func PhysicalNameClusterScoped(name, physicalNamespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", physicalNamespace, "x", Suffix)
}

func ApplyMetadata(fromAnnotations map[string]string, toAnnotations map[string]string, fromLabels map[string]string, toLabels map[string]string, excludeAnnotations ...string) (labels map[string]string, annotations map[string]string) {
	mergedAnnotations := ApplyAnnotations(fromAnnotations, toAnnotations, excludeAnnotations...)
	return ApplyLabels(fromLabels, toLabels, mergedAnnotations)
}

func ApplyAnnotations(fromAnnotations map[string]string, toAnnotations map[string]string, excludeAnnotations ...string) map[string]string {
	if toAnnotations == nil {
		toAnnotations = map[string]string{}
	}

	excludedKeys := []string{ManagedAnnotationsAnnotation, ManagedLabelsAnnotation}
	excludedKeys = append(excludedKeys, excludeAnnotations...)
	mergedAnnotations, managedKeys := ApplyMaps(fromAnnotations, toAnnotations, ApplyMapsOptions{
		ManagedKeys: strings.Split(toAnnotations[ManagedAnnotationsAnnotation], "\n"),
		ExcludeKeys: excludedKeys,
	})
	if managedKeys == "" {
		delete(mergedAnnotations, ManagedAnnotationsAnnotation)
	} else {
		mergedAnnotations[ManagedAnnotationsAnnotation] = managedKeys
	}

	return mergedAnnotations
}

func ApplyLabels(fromLabels map[string]string, toLabels map[string]string, toAnnotations map[string]string) (labels map[string]string, annotations map[string]string) {
	if toAnnotations == nil {
		toAnnotations = map[string]string{}
	}

	mergedLabels, managedKeys := ApplyMaps(fromLabels, toLabels, ApplyMapsOptions{
		ManagedKeys: strings.Split(toAnnotations[ManagedLabelsAnnotation], "\n"),
		ExcludeKeys: []string{ManagedAnnotationsAnnotation, ManagedLabelsAnnotation},
	})
	mergedAnnotations := map[string]string{}
	for k, v := range toAnnotations {
		mergedAnnotations[k] = v
	}
	if managedKeys == "" {
		delete(mergedAnnotations, ManagedLabelsAnnotation)
	} else {
		mergedAnnotations[ManagedLabelsAnnotation] = managedKeys
	}

	return mergedLabels, mergedAnnotations
}

type ApplyMapsOptions struct {
	ManagedKeys []string
	ExcludeKeys []string
}

func ApplyMaps(fromMap map[string]string, toMap map[string]string, opts ApplyMapsOptions) (map[string]string, string) {
	retMap := map[string]string{}
	managedKeys := []string{}
	for k, v := range fromMap {
		if Exists(opts.ExcludeKeys, k) {
			continue
		}

		retMap[k] = v
		managedKeys = append(managedKeys, k)
	}

	for key, value := range toMap {
		if Exists(opts.ExcludeKeys, key) {
			if value != "" {
				retMap[key] = value
			}
			continue
		} else if Exists(managedKeys, key) || Exists(opts.ManagedKeys, key) {
			continue
		}

		retMap[key] = value
	}

	sort.Strings(managedKeys)
	managedKeysStr := strings.Join(managedKeys, "\n")
	return retMap, managedKeysStr
}

func Exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
}
