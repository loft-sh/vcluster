package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ManagedAnnotationsAnnotation = "vcluster.loft.sh/managed-annotations"
	NamespaceAnnotation          = "vcluster.loft.sh/object-namespace"
	NameAnnotation               = "vcluster.loft.sh/object-name"
	LabelPrefix                  = "vcluster.loft.sh/label"
)

type Translator interface {
	Translate(vObj client.Object) (runtime.Object, error)
	TranslateLabels(vObj client.Object) map[string]string
	TranslateAnnotations(vObj client.Object, pObj client.Object) map[string]string
}

type PhysicalNameTranslator func(vName string, vObj client.Object) string

func DefaultPhysicalName(vName string, vObj client.Object) string {
	name, namespace := vObj.GetName(), vObj.GetNamespace()
	return PhysicalName(name, namespace)
}

func NewDefaultTranslator(physicalNamespace string, excludedAnnotations ...string) Translator {
	return &defaultTranslator{
		physicalNamespace:   physicalNamespace,
		excludedAnnotations: excludedAnnotations,
	}
}

type defaultTranslator struct {
	physicalNamespace   string
	excludedAnnotations []string
}

func (d *defaultTranslator) Translate(vObj client.Object) (runtime.Object, error) {
	pObj, err := setupMetadataWithName(d.physicalNamespace, vObj, DefaultPhysicalName)
	if err != nil {
		return nil, err
	}

	pObj.SetLabels(d.TranslateLabels(vObj))
	pObj.SetAnnotations(d.TranslateAnnotations(vObj, nil))
	return pObj, nil
}

func (d *defaultTranslator) TranslateAnnotations(vObj client.Object, pObj client.Object) map[string]string {
	return translateAnnotations(vObj, pObj, d.excludedAnnotations)
}

func translateAnnotations(vObj client.Object, pObj client.Object, excluded []string) map[string]string {
	excluded = append(excluded, ManagedAnnotationsAnnotation, NameAnnotation, NamespaceAnnotation)

	retMap := map[string]string{}
	managedAnnotations := []string{}
	if vObj != nil {
		for k, v := range vObj.GetAnnotations() {
			if exists(excluded, k) {
				continue
			}

			retMap[k] = v
			managedAnnotations = append(managedAnnotations, k)
		}
	}

	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil {
			oldManagedAnnotationsStr := pAnnotations[ManagedAnnotationsAnnotation]
			oldManagedAnnotations := strings.Split(oldManagedAnnotationsStr, "\n")

			for key, value := range pAnnotations {
				if exists(excluded, key) {
					if value != "" {
						retMap[key] = value
					}
					continue
				} else if exists(managedAnnotations, key) || (exists(oldManagedAnnotations, key) && !exists(managedAnnotations, key)) {
					continue
				}

				retMap[key] = value
			}
		}
	}

	sort.Strings(managedAnnotations)
	retMap[NameAnnotation] = vObj.GetName()
	if vObj.GetNamespace() == "" {
		delete(retMap, NamespaceAnnotation)
	} else {
		retMap[NamespaceAnnotation] = vObj.GetNamespace()
	}

	managedAnnotationsStr := strings.Join(managedAnnotations, "\n")
	if managedAnnotationsStr == "" {
		delete(retMap, ManagedAnnotationsAnnotation)
	} else {
		retMap[ManagedAnnotationsAnnotation] = managedAnnotationsStr
	}
	return retMap
}

func (d *defaultTranslator) TranslateLabels(vObj client.Object) map[string]string {
	newLabels := map[string]string{}
	for k, v := range vObj.GetLabels() {
		if k == NamespaceLabel {
			newLabels[k] = v
			continue
		}

		newLabels[ConvertLabelKey(k)] = v
	}

	newLabels[MarkerLabel] = Suffix
	if vObj.GetNamespace() != "" {
		newLabels[NamespaceLabel] = vObj.GetNamespace()
	}
	return newLabels
}

func setupMetadataWithName(targetNamespace string, vObj client.Object, translator PhysicalNameTranslator) (client.Object, error) {
	target := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(translator(m.GetName(), vObj))
	if vObj.GetNamespace() != "" {
		m.SetNamespace(targetNamespace)

		// set owning stateful set if defined
		if Owner != nil {
			m.SetOwnerReferences(GetOwnerReference())
		}
	}

	return target, nil
}

func ConvertLabelKey(key string) string {
	return ConvertLabelKeyWithPrefix(LabelPrefix, key)
}

func ConvertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(prefix, Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

func exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
}

// ResetObjectMetadata resets the objects metadata except name, namespace and annotations
func ResetObjectMetadata(obj metav1.Object) {
	obj.SetGenerateName("")
	obj.SetSelfLink("")
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetDeletionTimestamp(nil)
	obj.SetDeletionGracePeriodSeconds(nil)
	obj.SetOwnerReferences(nil)
	obj.SetFinalizers(nil)
	obj.SetClusterName("")
	obj.SetManagedFields(nil)
}
