package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Translator = &multiNamespace{}

func NewMultiNamespaceTranslator(currentNamespace string) Translator {
	return &multiNamespace{
		currentNamespace: currentNamespace,
	}
}

type multiNamespace struct {
	currentNamespace string
}

func (s *multiNamespace) SingleNamespaceTarget() bool {
	return false
}

// PhysicalName returns the physical name of the name / namespace resource
func (s *multiNamespace) PhysicalName(name, _ string) string {
	return name
}

// PhysicalNameShort returns the short physical name of the name / namespace resource
func (s *multiNamespace) PhysicalNameShort(name, _ string) string {
	return name
}

func (s *multiNamespace) objectPhysicalName(obj runtime.Object) string {
	if obj == nil {
		return ""
	}

	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	return s.PhysicalName(metaAccessor.GetName(), metaAccessor.GetNamespace())
}

func (s *multiNamespace) PhysicalNameClusterScoped(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.currentNamespace, "x", VClusterName)
}

func (s *multiNamespace) IsManaged(obj runtime.Object, _ PhysicalNameFunc) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	// vcluster has not synced the object IF:
	// If obj is not in the synced namespace OR
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	if !s.IsTargetedNamespace(metaAccessor.GetNamespace()) || metaAccessor.GetAnnotations() == nil || metaAccessor.GetAnnotations()[NameAnnotation] == "" {
		return false
	}

	_, isCM := obj.(*corev1.ConfigMap)
	if isCM && metaAccessor.GetName() == "kube-root-ca.crt" {
		return false
	}

	return true
}

func (s *multiNamespace) IsManagedCluster(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetLabels() == nil {
		return false
	}

	return metaAccessor.GetLabels()[MarkerLabel] == SafeConcatName(s.currentNamespace, "x", VClusterName)
}

func (s *multiNamespace) IsTargetedNamespace(ns string) bool {
	return strings.HasPrefix(ns, s.getNamespacePrefix()) && strings.HasSuffix(ns, getNamespaceSuffix(s.currentNamespace, VClusterName))
}

func (s *multiNamespace) convertLabelKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return SafeConcatName(LabelPrefix, s.currentNamespace, "x", VClusterName, "x", hex.EncodeToString(digest[0:])[0:10])
}

func (s *multiNamespace) getNamespacePrefix() string {
	return "vcluster"
}

func (s *multiNamespace) PhysicalNamespace(vNamespace string) string {
	return PhysicalNamespace(s.currentNamespace, vNamespace, s.getNamespacePrefix(), VClusterName)
}

func PhysicalNamespace(currentNamespace, vNamespace, prefix, suffix string) string {
	sha := sha256.Sum256([]byte(vNamespace))
	return fmt.Sprintf("%s-%s-%s", prefix, hex.EncodeToString(sha[0:])[0:8], getNamespaceSuffix(currentNamespace, suffix))
}

func getNamespaceSuffix(currentNamespace, suffix string) string {
	sha := sha256.Sum256([]byte(currentNamespace + "x" + suffix))
	return hex.EncodeToString(sha[0:])[0:8]
}

func (s *multiNamespace) TranslateLabelsCluster(vObj client.Object, pObj client.Object, syncedLabels []string) map[string]string {
	newLabels := map[string]string{}
	if vObj != nil {
		vObjLabels := vObj.GetLabels()
		for k, v := range vObjLabels {
			newLabels[s.convertLabelKey(k)] = v
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
	newLabels[MarkerLabel] = SafeConcatName(s.currentNamespace, "x", VClusterName)
	return newLabels
}

func (s *multiNamespace) TranslateLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[s.convertLabelKey(k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      s.convertLabelKey(r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func (s *multiNamespace) LegacyGetTargetNamespace() (string, error) {
	return "", fmt.Errorf("unsupported feature in multi-namespace mode")
}

func (s *multiNamespace) ApplyMetadata(vObj client.Object, syncedLabels []string, excludedAnnotations ...string) client.Object {
	pObj, err := s.SetupMetadataWithName(vObj, func(_ string, vObj client.Object) string {
		return s.objectPhysicalName(vObj)
	})
	if err != nil {
		return nil
	}
	pObj.SetAnnotations(s.ApplyAnnotations(vObj, nil, excludedAnnotations))
	pObj.SetLabels(s.TranslateLabels(vObj.GetLabels(), vObj.GetNamespace(), syncedLabels))
	return pObj
}

func (s *multiNamespace) ApplyMetadataUpdate(vObj client.Object, pObj client.Object, syncedLabels []string, excludedAnnotations ...string) (bool, map[string]string, map[string]string) {
	updatedAnnotations := s.ApplyAnnotations(vObj, pObj, excludedAnnotations)
	updatedLabels := s.TranslateLabels(vObj.GetLabels(), vObj.GetNamespace(), syncedLabels)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (s *multiNamespace) ApplyAnnotations(src client.Object, to client.Object, excluded []string) map[string]string {
	excluded = append(excluded, NameAnnotation, NamespaceAnnotation)
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

func (s *multiNamespace) ApplyLabels(src client.Object, _ client.Object, syncedLabels []string) map[string]string {
	fromLabels := src.GetLabels()
	if fromLabels == nil {
		fromLabels = map[string]string{}
	}
	return s.TranslateLabels(fromLabels, src.GetNamespace(), syncedLabels)
}

func (s *multiNamespace) TranslateLabels(fromLabels map[string]string, _ string, _ []string) map[string]string {
	return fromLabels
}

func (s *multiNamespace) SetupMetadataWithName(vObj client.Object, translator PhysicalNameTranslator) (client.Object, error) {
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
	}

	return target, nil
}

func (s *multiNamespace) TranslateLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return labelSelector
}

func (s *multiNamespace) ConvertLabelKey(key string) string {
	return key
}
