package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/base36"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (s *singleNamespace) HostName(name, namespace string) string {
	return SingleNamespaceHostName(name, namespace, VClusterName)
}

func (s *singleNamespace) HostNameShort(name, namespace string) string {
	if name == "" {
		return ""
	}

	// we use base36 to avoid as much conflicts as possible
	digest := sha256.Sum256([]byte(strings.Join([]string{name, "x", namespace, "x", VClusterName}, "-")))
	return base36.EncodeBytes(digest[:])[0:10]
}

func SingleNamespaceHostName(name, namespace, suffix string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName(name, "x", namespace, "x", suffix)
}

func (s *singleNamespace) HostNameCluster(name string) string {
	if name == "" {
		return ""
	}
	return SafeConcatName("vcluster", name, "x", s.targetNamespace, "x", VClusterName)
}

func (s *singleNamespace) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) bool {
	// check if cluster scoped object
	if pObj.GetNamespace() == "" {
		return pObj.GetLabels()[MarkerLabel] == SafeConcatName(s.targetNamespace, "x", VClusterName)
	}

	// is object not in our target namespace?
	if !s.IsTargetedNamespace(pObj.GetNamespace()) {
		return false
	} else if pObj.GetLabels()[MarkerLabel] != VClusterName {
		return false
	}

	// vcluster has not synced the object IF:
	// If object-name annotation is not set OR
	// If object-name annotation is different from actual name
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err == nil {
		// check if the name annotation is correct
		if pObj.GetAnnotations()[NameAnnotation] == "" ||
			(ctx.Mappings.Has(gvk) && pObj.GetName() != mappings.VirtualToHostName(ctx, pObj.GetAnnotations()[NameAnnotation], pObj.GetAnnotations()[NamespaceAnnotation], gvk)) {
			klog.FromContext(ctx).V(1).Info("Host object doesn't match, because name annotations is wrong",
				"object", pObj.GetName(),
				"kind", gvk.String(),
				"existingName", pObj.GetName(),
				"expectedName", mappings.VirtualToHostName(ctx, pObj.GetAnnotations()[NameAnnotation], pObj.GetAnnotations()[NamespaceAnnotation], gvk),
				"nameAnnotation", pObj.GetAnnotations()[NamespaceAnnotation]+"/"+pObj.GetAnnotations()[NameAnnotation],
			)
			return false
		}

		// if kind doesn't match vCluster has probably not synced the object
		if pObj.GetAnnotations()[KindAnnotation] != "" && gvk.String() != pObj.GetAnnotations()[KindAnnotation] {
			klog.FromContext(ctx).V(1).Info("Host object doesn't match, because kind annotations is wrong",
				"object", pObj.GetName(),
				"existingKind", gvk.String(),
				"expectedKind", pObj.GetAnnotations()[KindAnnotation],
			)
			return false
		}
	}

	return true
}

func (s *singleNamespace) IsTargetedNamespace(ns string) bool {
	return ns == s.targetNamespace
}

func (s *singleNamespace) HostNamespace(_ string) string {
	return s.targetNamespace
}

func (s *singleNamespace) HostLabelsCluster(vLabels, pLabels map[string]string, syncedLabels []string) map[string]string {
	return hostLabelsCluster(vLabels, pLabels, s.targetNamespace, syncedLabels)
}

func hostLabelsCluster(vLabels, pLabels map[string]string, vClusterNamespace string, syncedLabels []string) map[string]string {
	newLabels := map[string]string{}
	for k, v := range vLabels {
		newLabels[hostLabelCluster(k, vClusterNamespace)] = v
	}
	if len(vLabels) > 0 {
		for _, k := range syncedLabels {
			if strings.HasSuffix(k, "/*") {
				r, _ := regexp.Compile(strings.ReplaceAll(k, "/*", "/.*"))

				for key, val := range vLabels {
					if r.MatchString(key) {
						newLabels[key] = val
					}
				}
			} else {
				if value, ok := vLabels[k]; ok {
					newLabels[k] = value
				}
			}
		}
	}
	if pLabels[ControllerLabel] != "" {
		newLabels[ControllerLabel] = pLabels[ControllerLabel]
	}
	newLabels[MarkerLabel] = SafeConcatName(vClusterNamespace, "x", VClusterName)
	return newLabels
}

func (s *singleNamespace) HostLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return hostLabelSelectorCluster(labelSelector, s.targetNamespace)
}

func hostLabelSelectorCluster(labelSelector *metav1.LabelSelector, vClusterNamespace string) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[hostLabelCluster(k, vClusterNamespace)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      hostLabelCluster(r.Key, vClusterNamespace),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
}

func (s *singleNamespace) HostLabels(vLabels, pLabels map[string]string, vNamespace string, syncedLabels []string) map[string]string {
	if vLabels == nil {
		return nil
	}

	newLabels := map[string]string{}
	for k, v := range vLabels {
		newLabels[s.HostLabel(k)] = v
	}
	for _, k := range syncedLabels {
		if strings.HasSuffix(k, "/*") {
			r, _ := regexp.Compile(strings.ReplaceAll(k, "/*", "/.*"))

			for key, val := range vLabels {
				if r.MatchString(key) {
					newLabels[key] = val
				}
			}
		} else {
			if value, ok := vLabels[k]; ok {
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

	// set controller label
	if pLabels[ControllerLabel] != "" {
		newLabels[ControllerLabel] = pLabels[ControllerLabel]
	}

	return newLabels
}

func (s *singleNamespace) HostLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
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

func (s *singleNamespace) HostLabel(key string) string {
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
