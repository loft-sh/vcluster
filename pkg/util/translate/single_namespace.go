package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
	"strings"
)

var _ Translator = &singleNamespace{}

func NewSingleNamespaceTranslator(targetNamespace string) Translator {
	return &singleNamespace{
		targetNamespace: targetNamespace,
	}
}

type singleNamespace struct {
	targetNamespace string
}

// PhysicalName returns the physical name of the name / namespace resource
func (s *singleNamespace) PhysicalName(name, namespace string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", Suffix)
}

func (s *singleNamespace) ObjectPhysicalName(obj runtime.Object) string {
	if obj == nil {
		return ""
	}

	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	return s.PhysicalName(metaAccessor.GetName(), metaAccessor.GetNamespace())
}

func (s *singleNamespace) PhysicalNameClusterScoped(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.targetNamespace, "x", Suffix)
}

func (s *singleNamespace) IsManaged(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	} else if metaAccessor.GetNamespace() != "" && metaAccessor.GetNamespace() != s.targetNamespace {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == Suffix
}

func (s *singleNamespace) IsManagedCluster(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == SafeConcatName(s.targetNamespace, "x", Suffix)
}

func (s *singleNamespace) GetOwnerReference(object client.Object) []metav1.OwnerReference {
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

func (s *singleNamespace) convertNamespacedLabelKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(LabelPrefix, s.targetNamespace, "x", Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

func (s *singleNamespace) PhysicalNamespace(vNamespace string) string {
	return s.targetNamespace
}

func (s *singleNamespace) TranslateLabelsCluster(vObj client.Object, pObj client.Object, syncedLabels []string) map[string]string {
	newLabels := map[string]string{}
	if vObj != nil {
		vObjLabels := vObj.GetLabels()
		for k, v := range vObjLabels {
			newLabels[s.convertNamespacedLabelKey(k)] = v
		}
		if vObjLabels != nil {
			for _, k := range syncedLabels {
				if value, ok := vObjLabels[k]; ok {
					newLabels[k] = value
				}
			}
		}
	}
	if pObj != nil {
		pObjLabels := pObj.GetLabels()
		if pObjLabels != nil && pObjLabels[ControllerLabel] != "" {
			newLabels[ControllerLabel] = pObjLabels[ControllerLabel]
		}
	}
	newLabels[MarkerLabel] = SafeConcatName(s.targetNamespace, "x", Suffix)
	return newLabels
}

func (s *singleNamespace) TranslateLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[s.convertNamespacedLabelKey(k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      s.convertNamespacedLabelKey(r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func (s *singleNamespace) LegacyGetTargetNamespace() (string, error) {
	return s.targetNamespace, nil
}

func ConvertLabelKey(key string) string {
	return ConvertLabelKeyWithPrefix(LabelPrefix, key)
}

func ConvertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(prefix, Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

func TranslateLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return TranslateLabelSelectorWithPrefix(LabelPrefix, labelSelector)
}

func TranslateLabelSelectorWithPrefix(labelPrefix string, labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[ConvertLabelKeyWithPrefix(labelPrefix, k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      ConvertLabelKeyWithPrefix(labelPrefix, r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func MergeLabelSelectors(elems ...*metav1.LabelSelector) *metav1.LabelSelector {
	out := &metav1.LabelSelector{}
	for _, selector := range elems {
		if selector == nil {
			continue
		}
		for k, v := range selector.MatchLabels {
			if out.MatchLabels == nil {
				out.MatchLabels = map[string]string{}
			}
			out.MatchLabels[k] = v
		}
		for _, expr := range selector.MatchExpressions {
			if out.MatchExpressions == nil {
				out.MatchExpressions = []metav1.LabelSelectorRequirement{}
			}
			out.MatchExpressions = append(out.MatchExpressions, expr)
		}
	}
	return out
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
