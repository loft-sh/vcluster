package poddisruptionbudgets

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *pdbSyncer) translate(ctx *synccontext.SyncContext, vObj *policyv1.PodDisruptionBudget) *policyv1.PodDisruptionBudget {
	newPDB := translate.HostMetadata(vObj, s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	newPDB.Spec.Selector = hostSelector(vObj.Spec.Selector, vObj.Namespace)
	return newPDB
}

func (s *pdbSyncer) translateUpdate(pObj, vObj *policyv1.PodDisruptionBudget) {
	pObj.Spec.MaxUnavailable = vObj.Spec.MaxUnavailable
	pObj.Spec.MinAvailable = vObj.Spec.MinAvailable
	pObj.Spec.UnhealthyPodEvictionPolicy = vObj.Spec.UnhealthyPodEvictionPolicy
	pObj.Spec.Selector = hostSelector(vObj.Spec.Selector, vObj.Namespace)
}

// hostSelector translates the selector and limits it to the pods of the virtual
// namespace, since every virtual namespace shares the host namespace with the
// other virtual namespaces, the control plane and any other host pod.
func hostSelector(selector *metav1.LabelSelector, vNamespace string) *metav1.LabelSelector {
	translated := translate.HostLabelSelector(selector)
	if translated == nil {
		return nil
	}

	if translated.MatchLabels == nil {
		translated.MatchLabels = map[string]string{}
	}
	translated.MatchLabels[translate.NamespaceLabel] = vNamespace
	translated.MatchLabels[translate.MarkerLabel] = translate.VClusterName
	return translated
}
