package translator

import (
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func PrintChanges(oldObject, newObject client.Object, log loghelper.Logger) {
	if os.Getenv("DEBUG") == "true" {
		rawPatch, err := client.MergeFrom(oldObject).Data(newObject)
		if err == nil {
			log.Debugf("Updating object with: %v", string(rawPatch))
		}
	}
}

func TranslateLabelSelectorCluster(physicalNamespace string, labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[convertNamespacedLabelKey(physicalNamespace, k)] = v
		}
	}
	if len(labelSelector.MatchExpressions) > 0 {
		newLabelSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
		for _, r := range labelSelector.MatchExpressions {
			newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      convertNamespacedLabelKey(physicalNamespace, r.Key),
				Operator: r.Operator,
				Values:   r.Values,
			})
		}
	}

	return newLabelSelector
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

// ObjectPhysicalName returns the translated physical name of this object
func ObjectPhysicalName(obj runtime.Object) string {
	metaAccessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}

	return translate.PhysicalName(metaAccessor.GetName(), metaAccessor.GetNamespace())
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
