package networkpolicies

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *networkPolicySyncer) translate(ctx *synccontext.SyncContext, vNetworkPolicy *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	newNetworkPolicy := translate.HostMetadata(ctx, vNetworkPolicy, s.VirtualToHost(ctx, types.NamespacedName{Name: vNetworkPolicy.GetName(), Namespace: vNetworkPolicy.GetNamespace()}, vNetworkPolicy))
	if spec := translateSpec(ctx, &vNetworkPolicy.Spec, vNetworkPolicy.GetNamespace()); spec != nil {
		newNetworkPolicy.Spec = *spec
	}
	return newNetworkPolicy
}

func (s *networkPolicySyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *networkingv1.NetworkPolicy) {
	if translatedSpec := translateSpec(ctx, &vObj.Spec, vObj.GetNamespace()); translatedSpec != nil {
		pObj.Spec = *translatedSpec
	}

	pObj.Annotations = translate.HostAnnotations(vObj, pObj)
	pObj.Labels = translate.HostLabels(ctx, vObj, pObj)
}

func translateSpec(ctx *synccontext.SyncContext, spec *networkingv1.NetworkPolicySpec, namespace string) *networkingv1.NetworkPolicySpec {
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
			To:    translateNetworkPolicyPeers(ctx, er.To, namespace),
		})
	}
	for _, ir := range spec.Ingress {
		if outSpec.Ingress == nil {
			outSpec.Ingress = []networkingv1.NetworkPolicyIngressRule{}
		}
		outSpec.Ingress = append(outSpec.Ingress, networkingv1.NetworkPolicyIngressRule{
			Ports: ir.Ports,
			From:  translateNetworkPolicyPeers(ctx, ir.From, namespace),
		})
	}

	// TODO(Multi-Namespace): add support for multi-namespace translation
	if !translate.Default.SingleNamespaceTarget() {
		panic("Multi-Namespace Mode not supported for network policies yet!")
	}

	if translatedLabelSelector := translate.HostLabelSelector(ctx, &spec.PodSelector, namespace); translatedLabelSelector != nil {
		outSpec.PodSelector = *translatedLabelSelector
		if outSpec.PodSelector.MatchLabels == nil {
			outSpec.PodSelector.MatchLabels = map[string]string{}
		}
		// add selector for namespace as NetworkPolicy podSelector applies to pods within it's namespace
		outSpec.PodSelector.MatchLabels[translate.NamespaceLabel] = namespace
		// add selector for the marker label to select only from pods belonging this vcluster instance
		outSpec.PodSelector.MatchLabels[translate.MarkerLabel] = translate.VClusterName
	}

	outSpec.PolicyTypes = spec.PolicyTypes
	return outSpec
}

func translateNetworkPolicyPeers(ctx *synccontext.SyncContext, peers []networkingv1.NetworkPolicyPeer, namespace string) []networkingv1.NetworkPolicyPeer {
	if peers == nil {
		return nil
	}
	out := []networkingv1.NetworkPolicyPeer{}
	for _, peer := range peers {
		newPeer := networkingv1.NetworkPolicyPeer{
			PodSelector:       translate.HostLabelSelector(ctx, peer.PodSelector, namespace),
			NamespaceSelector: nil, // must be set to nil as all vcluster pods are in the same host namespace as the NetworkPolicy
		}
		if peer.IPBlock == nil {
			translatedNamespaceSelectors := translate.HostLabelSelectorNamespace(ctx, peer.NamespaceSelector)
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
