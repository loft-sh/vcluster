package translate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/util/base36"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
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

	// we use base36 to avoid as much conflicts as possible
	digest := sha256.Sum256([]byte(strings.Join([]string{name, "x", namespace, "x", VClusterName}, "-")))
	return base36.EncodeBytes(digest[:])[0:10]
}

func SingleNamespacePhysicalName(name, namespace, suffix string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", suffix)
}

func (s *singleNamespace) PhysicalNameClusterScoped(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) IsManaged(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return false
	} else if metaAccessor.GetNamespace() != "" && !s.IsTargetedNamespace(metaAccessor.GetNamespace()) {
		return false
	} else if metaAccessor.GetLabels()[MarkerLabel] != VClusterName {
		return false
	}

	// vcluster has not synced the object IF:
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err == nil {
		// check if the name annotation is correct
		if metaAccessor.GetAnnotations()[NameAnnotation] == "" ||
			(mappings.Has(gvk) && metaAccessor.GetName() != mappings.VirtualToHostName(context.TODO(), metaAccessor.GetAnnotations()[NameAnnotation], metaAccessor.GetAnnotations()[NamespaceAnnotation], mappings.ByGVK(gvk))) {
			klog.FromContext(context.TODO()).V(1).Info("Host object doesn't match, because name annotations is wrong",
				"object", metaAccessor.GetName(),
				"kind", gvk.String(),
				"existingName", metaAccessor.GetName(),
				"expectedName", mappings.VirtualToHostName(context.TODO(), metaAccessor.GetAnnotations()[NameAnnotation], metaAccessor.GetAnnotations()[NamespaceAnnotation], mappings.ByGVK(gvk)),
				"nameAnnotation", metaAccessor.GetAnnotations()[NamespaceAnnotation]+"/"+metaAccessor.GetAnnotations()[NameAnnotation],
			)
			return false
		}

		// if kind doesn't match vCluster has probably not synced the object
		if metaAccessor.GetAnnotations()[KindAnnotation] != "" && gvk.String() != metaAccessor.GetAnnotations()[KindAnnotation] {
			klog.FromContext(context.TODO()).V(1).Info("Host object doesn't match, because kind annotations is wrong",
				"object", metaAccessor.GetName(),
				"existingKind", gvk.String(),
				"expectedKind", metaAccessor.GetAnnotations()[KindAnnotation],
			)
			return false
		}
	}

	return true
}

func (s *singleNamespace) IsManagedCluster(obj runtime.Object) bool {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
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

func (s *singleNamespace) ApplyMetadata(vObj client.Object, name types.NamespacedName, syncedLabels []string, excludedAnnotations ...string) client.Object {
	pObj, err := s.SetupMetadataWithName(vObj, name)
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
	excluded = append(excluded, NameAnnotation, UIDAnnotation, KindAnnotation, NamespaceAnnotation)
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

	gvk, err := apiutil.GVKForObject(src, scheme.Scheme)
	if err == nil {
		retMap[KindAnnotation] = gvk.String()
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
			if newLabels == nil {
				newLabels = make(map[string]string)
			}
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

func (s *singleNamespace) SetupMetadataWithName(vObj client.Object, name types.NamespacedName) (client.Object, error) {
	target := vObj.DeepCopyObject().(client.Object)
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	// reset metadata & translate name and namespace
	ResetObjectMetadata(m)
	m.SetName(name.Name)
	if vObj.GetNamespace() != "" {
		m.SetNamespace(name.Namespace)

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
