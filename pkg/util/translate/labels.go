package translate

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsTranslatedLabel(label string) (string, bool) {
	for k := range Default.LabelsToTranslate() {
		if convertLabelKeyWithPrefix(LabelPrefix, k) == label {
			return k, true
		}
	}
	return "", false
}

func HostLabel(vLabel string) string {
	if Default.LabelsToTranslate()[vLabel] {
		return convertLabelKeyWithPrefix(LabelPrefix, vLabel)
	}

	return vLabel
}

func VirtualLabel(pLabel string) (string, bool) {
	if Default.LabelsToTranslate()[pLabel] {
		return "", false
	}

	if originalLabel, ok := IsTranslatedLabel(pLabel); ok {
		return originalLabel, true
	}

	return pLabel, true
}

func HostLabelsMap(vLabels, pLabels map[string]string, vNamespace string, isMetadata bool) map[string]string {
	if vLabels == nil {
		return nil
	}

	newLabels := map[string]string{}
	for k, v := range vLabels {
		if _, ok := IsTranslatedLabel(k); ok {
			continue
		}

		newLabels[HostLabel(k)] = v
	}

	// check if we should add namespace and marker label
	if isMetadata || pLabels == nil || pLabels[MarkerLabel] != "" {
		if vNamespace == "" {
			newLabels[MarkerLabel] = Default.MarkerLabelCluster()
		} else if Default.SingleNamespaceTarget() {
			newLabels[MarkerLabel] = VClusterName
			newLabels[NamespaceLabel] = vNamespace
		}
	}

	// set controller label
	if pLabels[ControllerLabel] != "" {
		newLabels[ControllerLabel] = pLabels[ControllerLabel]
	}

	return newLabels
}

func VirtualLabelsMap(pLabels, vLabels map[string]string, excluded ...string) map[string]string {
	if pLabels == nil {
		return nil
	}

	excluded = append(excluded, MarkerLabel, NamespaceLabel, ControllerLabel)
	retLabels := copyMaps(pLabels, vLabels, func(key string) bool {
		return exists(excluded, key) || strings.HasPrefix(key, NamespaceLabelPrefix)
	})

	// try to translate back
	for key, value := range retLabels {
		// if the original key was on vLabels we want to preserve it
		vValue, ok := vLabels[key]
		if !ok {
			delete(retLabels, key)
		} else {
			retLabels[key] = vValue
		}

		// if the virtual label can be converted we will add it back
		vKey, ok := VirtualLabel(key)
		if ok {
			retLabels[vKey] = value
		}
	}

	return retLabels
}

func VirtualLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return virtualLabelSelector(labelSelector, func(key string) (string, bool) {
		return VirtualLabel(key)
	})
}

type vLabelFunc func(key string) (string, bool)

func virtualLabelSelector(labelSelector *metav1.LabelSelector, labelFunc vLabelFunc) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			pLabel, ok := labelFunc(k)
			if !ok {
				pLabel = k
			}

			newLabelSelector.MatchLabels[pLabel] = v
		}
	}
	for _, r := range labelSelector.MatchExpressions {
		pLabel, ok := labelFunc(r.Key)
		if !ok {
			pLabel = r.Key
		}

		newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
			Key:      pLabel,
			Operator: r.Operator,
			Values:   r.Values,
		})
	}

	return newLabelSelector
}

func HostLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector {
	return hostLabelSelector(labelSelector, func(key string) string {
		return HostLabel(key)
	})
}

type labelFunc func(key string) string

func hostLabelSelector(labelSelector *metav1.LabelSelector, labelFunc labelFunc) *metav1.LabelSelector {
	if labelSelector == nil {
		return nil
	}

	newLabelSelector := &metav1.LabelSelector{}
	if labelSelector.MatchLabels != nil {
		newLabelSelector.MatchLabels = map[string]string{}
		for k, v := range labelSelector.MatchLabels {
			newLabelSelector.MatchLabels[labelFunc(k)] = v
		}
	}
	for _, r := range labelSelector.MatchExpressions {
		newLabelSelector.MatchExpressions = append(newLabelSelector.MatchExpressions, metav1.LabelSelectorRequirement{
			Key:      labelFunc(r.Key),
			Operator: r.Operator,
			Values:   r.Values,
		})
	}

	return newLabelSelector
}

func VirtualLabels(pObj, vObj client.Object) map[string]string {
	pLabels := pObj.GetLabels()
	if pLabels == nil {
		pLabels = map[string]string{}
	}
	var vLabels map[string]string
	if vObj != nil {
		vLabels = vObj.GetLabels()
	}
	retLabels := VirtualLabelsMap(pLabels, vLabels)
	if len(retLabels) == 0 {
		return nil
	}
	return retLabels
}

func HostLabels(vObj, pObj client.Object) map[string]string {
	vLabels := vObj.GetLabels()
	if vLabels == nil {
		vLabels = map[string]string{}
	}
	var pLabels map[string]string
	if pObj != nil {
		pLabels = pObj.GetLabels()
	}
	retLabels := HostLabelsMap(vLabels, pLabels, vObj.GetNamespace(), true)
	if len(retLabels) == 0 {
		return nil
	}
	return retLabels
}

func MergeLabelSelectors(elems ...*metav1.LabelSelector) *metav1.LabelSelector {
	out := &metav1.LabelSelector{}
	for _, selector := range elems {
		if selector == nil {
			continue
		}
		if len(selector.MatchLabels) > 0 {
			if out.MatchLabels == nil {
				out.MatchLabels = make(map[string]string, len(selector.MatchLabels))
			}
			for k, v := range selector.MatchLabels {
				out.MatchLabels[k] = v
			}
		}
		out.MatchExpressions = append(out.MatchExpressions, selector.MatchExpressions...)
	}
	return out
}
