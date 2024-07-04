package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (s *singleNamespace) SingleNamespaceTarget() bool {
	return true
}

// PhysicalName returns the physical name of the name / namespace resource
func (s *singleNamespace) PhysicalName(name, namespace string) string {
	return SingleNamespacePhysicalName(name, namespace, VClusterName)
}

// PhysicalNameShort returns the short physical name of the name / namespace resource
func (s *singleNamespace) PhysicalNameShort(name, namespace string) string {
	if name == "" {
		return ""
	}

	digest := sha256.Sum256([]byte(strings.Join([]string{name, "x", namespace, "x", VClusterName}, "-")))
	return hex.EncodeToString(digest[0:])[0:8]
}

func SingleNamespacePhysicalName(name, namespace, suffix string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", suffix)
}

func (s *singleNamespace) objectPhysicalName(obj runtime.Object) string {
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
	return SafeConcatName("vcluster", name, "x", s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) IsManaged(obj runtime.Object, physicalName PhysicalNameFunc) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	} else if metaAccessor.GetNamespace() != "" && !s.IsTargetedNamespace(metaAccessor.GetNamespace()) {
		return false
	}

	// vcluster has not synced the object IF:
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	if metaAccessor.GetAnnotations() == nil ||
		metaAccessor.GetAnnotations()[NameAnnotation] == "" ||
		metaAccessor.GetName() != physicalName(metaAccessor.GetAnnotations()[NameAnnotation], metaAccessor.GetAnnotations()[NamespaceAnnotation]) {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == VClusterName
}

func (s *singleNamespace) IsManagedCluster(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == SafeConcatName(s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) IsTargetedNamespace(ns string) bool {
	return ns == s.targetNamespace
}

func (s *singleNamespace) convertNamespacedLabelKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(LabelPrefix, s.targetNamespace, "x", VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
}

func (s *singleNamespace) PhysicalNamespace(_ string) string {
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
				if strings.HasSuffix(k, "/*") {
					r, _ := regexp.Compile(strings.ReplaceAll(k, "/*", "/.*"))

					for key, val := range vObjLabels {
						if r.MatchString(key) {
							newLabels[key] = val
						}
					}
				} else {
					if value, ok := vObjLabels[k]; ok {
						newLabels[k] = value
					}
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
	newLabels[MarkerLabel] = SafeConcatName(s.targetNamespace, "x", VClusterName)
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

func (s *singleNamespace) ApplyMetadata(vObj client.Object, syncedLabels []string, excludedAnnotations ...string) client.Object {
	pObj, err := s.SetupMetadataWithName(vObj, func(_ string, vObj client.Object) string {
		return s.objectPhysicalName(vObj)
	})
	if err != nil {
		return nil
	}
	pObj.SetAnnotations(s.ApplyAnnotations(vObj, nil, excludedAnnotations))
	pObj.SetLabels(s.ApplyLabels(vObj, nil, syncedLabels))
	return pObj
}

func (s *singleNamespace) ApplyMetadataUpdate(vObj client.Object, pObj client.Object, syncedLabels []string, excludedAnnotations ...string) (bool, map[string]string, map[string]string) {
	updatedAnnotations := s.ApplyAnnotations(vObj, pObj, excludedAnnotations)
	updatedLabels := s.ApplyLabels(vObj, pObj, syncedLabels)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (s *singleNamespace) ApplyAnnotations(src client.Object, to client.Object, excluded []string) map[string]string {
	excluded = append(excluded, NameAnnotation, UIDAnnotation, NamespaceAnnotation)
	toAnnotations := map[string]string{}
	if to != nil {
		toAnnotations = to.GetAnnotations()
	}

	retMap := applyAnnotations(src.GetAnnotations(), toAnnotations, excluded...)
	retMap[NameAnnotation] = src.GetName()
	retMap[UIDAnnotation] = string(src.GetUID())
	if src.GetNamespace() == "" {
		delete(retMap, NamespaceAnnotation)
	} else {
		retMap[NamespaceAnnotation] = src.GetNamespace()
	}

	return retMap
}

func (s *singleNamespace) ApplyLabels(src client.Object, dest client.Object, syncedLabels []string) map[string]string {
	fromLabels := src.GetLabels()
	if fromLabels == nil {
		fromLabels = map[string]string{}
	}

	newLabels := s.TranslateLabels(fromLabels, src.GetNamespace(), syncedLabels)
	if dest != nil {
		pObjLabels := dest.GetLabels()
		if pObjLabels != nil && pObjLabels[ControllerLabel] != "" {
			newLabels[ControllerLabel] = pObjLabels[ControllerLabel]
		}
	}

	return newLabels
}

func (s *singleNamespace) TranslateLabels(fromLabels map[string]string, vNamespace string, syncedLabels []string) map[string]string {
	if fromLabels == nil {
		return nil
	}

	newLabels := map[string]string{}
	for k, v := range fromLabels {
		newLabels[s.ConvertLabelKey(k)] = v
	}
	for _, k := range syncedLabels {
		if strings.HasSuffix(k, "/*") {
			r, _ := regexp.Compile(strings.ReplaceAll(k, "/*", "/.*"))

			for key, val := range fromLabels {
				if r.MatchString(key) {
					newLabels[key] = val
				}
			}
		} else {
			if value, ok := fromLabels[k]; ok {
				newLabels[k] = value
			}
		}
	}

	newLabels[MarkerLabel] = VClusterName
	if vNamespace != "" {
		newLabels[NamespaceLabel] = vNamespace
	} else {
		delete(newLabels, NamespaceLabel)
	}

	return newLabels
}

func (s *singleNamespace) SetupMetadataWithName(vObj client.Object, translator PhysicalNameTranslator) (client.Object, error) {
	target := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(translator(m.GetName(), vObj))
	if vObj.GetNamespace() != "" {
		m.SetNamespace(s.PhysicalNamespace(vObj.GetNamespace()))

		// set owning stateful set if defined
		if Owner != nil {
			m.SetOwnerReferences(GetOwnerReference(vObj))
		}
	}

	return target, nil
}

func (s *singleNamespace) TranslateLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return LabelSelectorWithPrefix(LabelPrefix, labelSelector)
}

func LabelSelectorWithPrefix(labelPrefix string, labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
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

func (s *singleNamespace) ConvertLabelKey(key string) string {
	return ConvertLabelKeyWithPrefix(LabelPrefix, key)
}

func ConvertLabelKeyWithPrefix(prefix, key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(prefix, VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
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
