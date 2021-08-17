package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	NamespaceLabel = "vcluster.loft.sh/namespace"
	MarkerLabel    = "vcluster.loft.sh/managed-by"
	Suffix         = "suffix"
)

func Split(s, sep string) (string, string) {
	parts := strings.SplitN(s, sep, 2)
	return strings.TrimSpace(parts[0]), strings.TrimSpace(safeIndex(parts, 1))
}

func safeIndex(parts []string, idx int) string {
	if len(parts) <= idx {
		return ""
	}
	return parts[idx]
}

func SafeConcatGenerateName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 53 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.Replace(fullPath[0:42]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-", -1)
	}
	return fullPath
}

func SafeConcatName(name ...string) string {
	fullPath := strings.Join(name, "-")
	if len(fullPath) > 63 {
		digest := sha256.Sum256([]byte(fullPath))
		return strings.Replace(fullPath[0:52]+"-"+hex.EncodeToString(digest[0:])[0:10], ".-", "-", -1)
	}
	return fullPath
}

func SetExcept(from map[string]string, to map[string]string, except ...string) map[string]string {
	retMap := map[string]string{}
	if from != nil {
		for k, v := range from {
			if exists(except, k) {
				continue
			}

			retMap[k] = v
		}
	}

	if to != nil {
		for _, k := range except {
			if to[k] != "" {
				retMap[k] = to[k]
			}
		}
	}

	if len(retMap) == 0 {
		return nil
	}

	return retMap
}

func UniqueSlice(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if entry == "" {
			continue
		}
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func LabelsEqual(virtualNamespace string, virtualLabels map[string]string, physicalLabels map[string]string) bool {
	physicalLabelsToCompare := TranslateLabels(virtualNamespace, virtualLabels)
	return EqualExcept(physicalLabelsToCompare, physicalLabels)
}

func LabelsClusterEqual(physicalNamespace string, virtualLabels map[string]string, physicalLabels map[string]string) bool {
	physicalLabelsToCompare := TranslateLabelsCluster(physicalNamespace, virtualLabels)
	return EqualExcept(physicalLabelsToCompare, physicalLabels)
}

func TranslateLabelSelectorCluster(physicalNamespace string, labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[ConvertNamespacedLabelKey(physicalNamespace, k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      ConvertNamespacedLabelKey(physicalNamespace, r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func TranslateLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[ConvertLabelKey(k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      ConvertLabelKey(r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func EqualExcept(a map[string]string, b map[string]string, except ...string) bool {
	for k, v := range a {
		if exists(except, k) {
			continue
		}

		if b == nil || b[k] != v {
			return false
		}
	}

	for k, v := range b {
		if exists(except, k) {
			continue
		}

		if a == nil || a[k] != v {
			return false
		}
	}

	return true
}

func exists(a []string, k string) bool {
	for _, i := range a {
		if i == k {
			return true
		}
	}

	return false
}

func IsManaged(obj runtime.Object) bool {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if meta.GetLabels() == nil {
		return false
	}

	return meta.GetLabels()[MarkerLabel] == Suffix
}

func IsManagedCluster(physicalNamespace string, obj runtime.Object) bool {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if meta.GetLabels() == nil {
		return false
	}

	return meta.GetLabels()[MarkerLabel] == SafeConcatName(physicalNamespace, "x", Suffix)
}

// PhysicalName returns the physical name of the name / namespace resource
func PhysicalName(name, namespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", Suffix)
}

// PhysicalNameClusterScoped returns the physical name of a cluster scoped object in the host cluster
func PhysicalNameClusterScoped(name, physicalNamespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", physicalNamespace, "x", Suffix)
}

// ObjectPhysicalName returns the translated physical name of this object
func ObjectPhysicalName(obj runtime.Object) string {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	return PhysicalName(metaAccessor.GetName(), metaAccessor.GetNamespace())
}

func SetupMetadata(targetNamespace string, obj runtime.Object) (runtime.Object, error) {
	target := obj.DeepCopyObject()
	if err := initMetadata(targetNamespace, target); err != nil {
		return nil, err
	}

	return target, nil
}

type PhysicalNameTranslator interface {
	PhysicalName(vName string, vObj runtime.Object) string
}

func SetupMetadataCluster(targetNamespace string, vObj runtime.Object, translator PhysicalNameTranslator) (runtime.Object, error) {
	target := vObj.DeepCopyObject()
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(translator.PhysicalName(m.GetName(), vObj))
	// set marker label
	m.SetLabels(TranslateLabelsCluster(targetNamespace, m.GetLabels()))
	return target, nil
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

// TranslateLabels transforms the virtual labels into physical ones
func TranslateLabels(virtualNamespace string, labels map[string]string) map[string]string {
	newLabels := map[string]string{}
	for k, v := range labels {
		if k == NamespaceLabel {
			newLabels[k] = v
			continue
		}

		newLabels[ConvertLabelKey(k)] = v
	}
	newLabels[MarkerLabel] = Suffix
	if virtualNamespace != "" && newLabels[NamespaceLabel] == "" {
		newLabels[NamespaceLabel] = NamespaceLabelValue(virtualNamespace)
	}

	return newLabels
}

// TranslateLabelsCluster transforms the virtual labels into physical ones
func TranslateLabelsCluster(physicalNamespace string, labels map[string]string) map[string]string {
	newLabels := map[string]string{}
	for k, v := range labels {
		newLabels[ConvertNamespacedLabelKey(physicalNamespace, k)] = v
	}
	newLabels[MarkerLabel] = SafeConcatName(physicalNamespace, "x", Suffix)
	return newLabels
}

func NamespaceLabelValue(virtualNamespace string) string {
	return SafeConcatName(virtualNamespace, "x", Suffix)
}

func ConvertNamespacedLabelKey(physicalNamespace, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName("vcluster.loft.sh/label", physicalNamespace, "x", Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

func ConvertLabelKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName("vcluster.loft.sh/label", Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

var OwningStatefulSet *appsv1.StatefulSet

func initMetadata(targetNamespace string, target runtime.Object) error {
	m, err := meta.Accessor(target)
	if err != nil {
		return err
	}

	// reset metadata & translate name and namespace
	name, namespace := m.GetName(), m.GetNamespace()
	ResetObjectMetadata(m)
	m.SetName(PhysicalName(name, namespace))
	m.SetNamespace(targetNamespace)
	m.SetLabels(TranslateLabels(namespace, m.GetLabels()))

	// set owning stateful set if defined
	if OwningStatefulSet != nil {
		m.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
				Name:       OwningStatefulSet.Name,
				UID:        OwningStatefulSet.UID,
			},
		})
	}

	return nil
}
