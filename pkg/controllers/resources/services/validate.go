package services

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ValidateServiceBeforeSync checks whether service labels match with the label selector provided in the vcluster config.
// These matchers provided in the config decide whether the services will be synced to host or not.
func ValidateServiceBeforeSync(ctx *synccontext.SyncContext, serviceLabels map[string]string) (bool, error) {
	// fetch the selector provided in the config.
	configLabelSelector := ctx.Config.Sync.ToHost.Services.Selector
	var selector labels.Selector
	var err error
	if configLabelSelector != nil {
		// form metav1.LabelSelector object from selector provided in the config.
		labelSelector := &metav1.LabelSelector{
			MatchLabels:      configLabelSelector.MatchLabels,
			MatchExpressions: configLabelSelector.MatchExpressions,
		}
		selector, err = metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return false, fmt.Errorf("invalid label selector: %v", err)
		}
	}

	if selector != nil && !selector.Matches(labels.Set(serviceLabels)) {
		return false, nil
	}
	return true, nil
}
