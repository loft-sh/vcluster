package networkpolicies

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type syncer struct {
	eventRecoder    record.EventRecorder
	targetNamespace string

	enableIngress bool
	enableEgress  bool

	localClient   client.Client
	virtualClient client.Client
}

func (s *syncer) New() client.Object {
	return &networkingv1.NetworkPolicy{}
}

func (s *syncer) NewList() client.ObjectList {
	return &networkingv1.NetworkPolicyList{}
}

func (s *syncer) translate(vObj client.Object) (*networkingv1.NetworkPolicy, error) {
	newObj, err := translate.SetupMetadata(s.targetNamespace, vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	vNetworkPolicy := vObj.(*networkingv1.NetworkPolicy)
	newNetworkPolicy := newObj.(*networkingv1.NetworkPolicy)

	newPodSelector := translate.TranslateLabelSelector(&vNetworkPolicy.Spec.PodSelector)
	newNetworkPolicy.Spec.PodSelector = *newPodSelector

	newNetworkPolicy.Spec.PolicyTypes = make([]networkingv1.PolicyType, 0)

	newNetworkPolicy.Spec.Ingress = nil
	if s.enableIngress {
		for _, vRule := range vNetworkPolicy.Spec.Ingress {
			newRuleFrom := make([]networkingv1.NetworkPolicyPeer, 0)

			for _, vFrom := range vRule.From {
				newRuleFrom = append(newRuleFrom, s.translatePolicyPeer(vFrom))
			}

			newNetworkPolicy.Spec.Ingress = append(newNetworkPolicy.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				Ports: vRule.Ports,
				From:  newRuleFrom,
			})
		}

		newNetworkPolicy.Spec.PolicyTypes = append(newNetworkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress)
	}

	newNetworkPolicy.Spec.Egress = nil
	if s.enableEgress {
		for _, vRule := range vNetworkPolicy.Spec.Egress {
			newRuleTo := make([]networkingv1.NetworkPolicyPeer, 0)

			for _, vTo := range vRule.To {
				newRuleTo = append(newRuleTo, s.translatePolicyPeer(vTo))
			}

			newNetworkPolicy.Spec.Egress = append(newNetworkPolicy.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				Ports: vRule.Ports,
				To:    newRuleTo,
			})
		}

		newNetworkPolicy.Spec.PolicyTypes = append(newNetworkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress)
	}

	return newNetworkPolicy, nil
}

func (s *syncer) translatePolicyPeer(vPolicyPeer networkingv1.NetworkPolicyPeer) networkingv1.NetworkPolicyPeer {
	newToPodSelector := translate.TranslateLabelSelector(vPolicyPeer.PodSelector)

	if vPolicyPeer.NamespaceSelector != nil {
		if newToPodSelector == nil {
			newToPodSelector = &metav1.LabelSelector{}
		}

		vNamespaceMatchLabels := vPolicyPeer.NamespaceSelector.MatchLabels
		if len(vNamespaceMatchLabels) > 0 {
			_ = make(map[string]string, 0)
			// TODO: convert these to pod selector match labels
			//  how do? do pods get namespace labels on them?
			//  if they don't we need to modify the translate labels to add them somehow

		}

		vNamespaceMatchExpressions := vPolicyPeer.NamespaceSelector.MatchExpressions
		if len(vNamespaceMatchExpressions) > 0 {
			_ = make([]metav1.LabelSelectorRequirement, 0)
			// TODO: convert these to pod selector match labels
			//  how do? do pods get namespace labels on them?
			//  if they don't we need to modify the translate labels to add them somehow
		}
	}

	return networkingv1.NetworkPolicyPeer{
		PodSelector:       newToPodSelector,
		NamespaceSelector: nil,
		IPBlock:           vPolicyPeer.IPBlock,
	}
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.ForwardCreateNeeded(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if createNeeded == false {
		return ctrl.Result{}, nil
	}

	vNetworkPolicy := vObj.(*networkingv1.NetworkPolicy)
	newNetworkPolicy, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical network policy %s/%s", newNetworkPolicy.Namespace, newNetworkPolicy.Name)
	err = s.localClient.Create(ctx, newNetworkPolicy)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vNetworkPolicy.Namespace, vNetworkPolicy.Name, err)
		s.eventRecoder.Eventf(vNetworkPolicy, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardCreateNeeded(vObj client.Object) (bool, error) {
	vNetworkPolicy := vObj.(*networkingv1.NetworkPolicy)

	if s.enableIngress && len(vNetworkPolicy.Spec.Ingress) > 0 {
		return true, nil
	}

	if s.enableEgress && len(vNetworkPolicy.Spec.Egress) > 0 {
		return true, nil
	}

	return false, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pNetworkPolicy := pObj.(*networkingv1.NetworkPolicy)
	vNetworkPolicy := vObj.(*networkingv1.NetworkPolicy)

	updated, err := s.calcNetworkPolicyDiff(pNetworkPolicy, vNetworkPolicy)
	if err != nil {
		return ctrl.Result{}, err
	}

	if updated != nil {
		log.Infof("updating physical network policy %s/%s, because virtual network policy has changed", updated.Namespace, updated.Name)
		err := s.localClient.Update(ctx, updated)
		if err != nil {
			s.eventRecoder.Eventf(vNetworkPolicy, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	updated, err := s.calcNetworkPolicyDiff(pObj.(*networkingv1.NetworkPolicy), vObj.(*networkingv1.NetworkPolicy))
	if err != nil {
		return false, err
	}
	return updated != nil, nil
}

func (s *syncer) calcNetworkPolicyDiff(pObj, vObj *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
	var updated *networkingv1.NetworkPolicy

	// check annotations
	if !equality.Semantic.DeepEqual(vObj.Annotations, pObj.Annotations) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Annotations = vObj.Annotations
	}

	translatedVObj, err := s.translate(vObj)
	if err != nil {
		return nil, err
	}

	// check labels
	if !equality.Semantic.DeepEqual(translatedVObj.Labels, pObj.Labels) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Labels = translatedVObj.Labels
	}

	// check policy types
	if !equality.Semantic.DeepEqual(translatedVObj.Spec.PolicyTypes, pObj.Spec.PolicyTypes) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Spec.PolicyTypes = translatedVObj.Spec.PolicyTypes
	}

	// check pod selector
	if !equality.Semantic.DeepEqual(translatedVObj.Spec.PodSelector, pObj.Spec.PodSelector) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Spec.PodSelector = translatedVObj.Spec.PodSelector
	}

	// check ingress
	if s.enableIngress {
		if !equality.Semantic.DeepEqual(translatedVObj.Spec.Ingress, pObj.Spec.Ingress) {
			if updated == nil {
				updated = pObj.DeepCopy()
			}
			updated.Spec.Ingress = translatedVObj.Spec.Ingress
		}
	}

	// check egress
	if s.enableEgress {
		if !equality.Semantic.DeepEqual(translatedVObj.Spec.Egress, pObj.Spec.Egress) {
			if updated == nil {
				updated = pObj.DeepCopy()
			}
			updated.Spec.Egress = translatedVObj.Spec.Egress
		}
	}

	return updated, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	return false, nil
}
