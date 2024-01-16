package networkpolicies

import (
	"context"

	podstranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *networkPolicySyncer) translate(ctx context.Context, vNetworkPolicy *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	newNetworkPolicy := s.TranslateMetadata(ctx, vNetworkPolicy).(*networkingv1.NetworkPolicy)
	newNetworkPolicy.Spec = *translateSpec(&vNetworkPolicy.Spec, vNetworkPolicy.GetNamespace())
	return newNetworkPolicy
}

func (s *networkPolicySyncer) translateUpdate(ctx context.Context, pObj, vObj *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	var updated *networkingv1.NetworkPolicy

	translatedSpec := *translateSpec(&vObj.Spec, vObj.GetNamespace())
	if !equality.Semantic.DeepEqual(translatedSpec, pObj.Spec) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Spec = translatedSpec
	}

	changed, translatedAnnotations, translatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Labels = translatedLabels
		updated.Annotations = translatedAnnotations
	}

	return updated
}

func translateSpec(spec *networkingv1.NetworkPolicySpec, namespace string) *networkingv1.NetworkPolicySpec {
	if spec == nil {
		return nil
	}

	outSpec := &networkingv1.NetworkPolicySpec{}
	for _, er := range spec.Egress {
		if outSpec.Egress == nil {
			outSpec.Egress = []networkingv1.NetworkPolicyEgressRule{}
		}
		outSpec.Egress = append(outSpec.Egress, networkingv1.NetworkPolicyEgressRule{
			Ports: er.Ports,
			To:    translateNetworkPolicyPeers(er.To, namespace),
		})
	}
	for _, ir := range spec.Ingress {
		if outSpec.Ingress == nil {
			outSpec.Ingress = []networkingv1.NetworkPolicyIngressRule{}
		}
		outSpec.Ingress = append(outSpec.Ingress, networkingv1.NetworkPolicyIngressRule{
			Ports: ir.Ports,
			From:  translateNetworkPolicyPeers(ir.From, namespace),
		})
	}

	// TODO(Multi-Namespace): add support for multi-namespace translation
	if !translate.Default.SingleNamespaceTarget() {
		panic("Multi-Namespace Mode not supported for network policies yet!")
	}

	outSpec.PodSelector = *translate.Default.TranslateLabelSelector(&spec.PodSelector)
	if outSpec.PodSelector.MatchLabels == nil {
		outSpec.PodSelector.MatchLabels = map[string]string{}
	}
	// add selector for namespace as NetworkPolicy podSelector applies to pods within it's namespace
	outSpec.PodSelector.MatchLabels[translate.NamespaceLabel] = namespace
	// add selector for the marker label to select only from pods belonging this vcluster instance
	outSpec.PodSelector.MatchLabels[translate.MarkerLabel] = translate.VClusterName

	outSpec.PolicyTypes = spec.PolicyTypes
	return outSpec
}

func translateNetworkPolicyPeers(peers []networkingv1.NetworkPolicyPeer, namespace string) []networkingv1.NetworkPolicyPeer {
	if peers == nil {
		return nil
	}
	out := []networkingv1.NetworkPolicyPeer{}
	for _, peer := range peers {
		newPeer := networkingv1.NetworkPolicyPeer{
			PodSelector:       translate.Default.TranslateLabelSelector(peer.PodSelector),
			NamespaceSelector: nil, // must be set to nil as all vcluster pods are in the same host namespace as the NetworkPolicy
		}
		if peer.IPBlock == nil {
			translatedNamespaceSelectors := translate.LabelSelectorWithPrefix(podstranslate.NamespaceLabelPrefix, peer.NamespaceSelector)
			newPeer.PodSelector = translate.MergeLabelSelectors(newPeer.PodSelector, translatedNamespaceSelectors)

			if newPeer.PodSelector.MatchLabels == nil {
				newPeer.PodSelector.MatchLabels = map[string]string{}
			}
			if peer.NamespaceSelector == nil {
				newPeer.PodSelector.MatchLabels[translate.NamespaceLabel] = namespace
			}
			// add selector for the marker label to select only from pods belonging this vcluster instance
			newPeer.PodSelector.MatchLabels[translate.MarkerLabel] = translate.VClusterName
		} else {
			newPeer.IPBlock = peer.IPBlock.DeepCopy()
		}
		out = append(out, newPeer)
	}
	return out
}
