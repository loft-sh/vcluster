package selector

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/config"
)

func IsLabelSelectorEmpty(labelSelector config.StandardLabelSelector) bool {
	ls := v1.LabelSelector(labelSelector)
	selector, err := v1.LabelSelectorAsSelector(&ls)
	if err != nil {
		return false
	}
	return selector.Empty()
}

func StandardLabelSelectorMatches(obj client.Object, labelSelector config.StandardLabelSelector) bool {
	ls := v1.LabelSelector(labelSelector)
	selector, err := v1.LabelSelectorAsSelector(&ls)
	if err != nil {
		return false
	}
	return selector.Matches(labels.Set(obj.GetLabels()))
}
