package backendtlspolicies

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
)

func (s *backendTLSPolicySyncer) translate(ctx *synccontext.SyncContext, vPolicy *gatewayv1alpha3.BackendTLSPolicy) (*gatewayv1alpha3.BackendTLSPolicy, error) {
	pPolicy := translate.HostMetadata(vPolicy, s.VirtualToHost(ctx, types.NamespacedName{Name: vPolicy.Name, Namespace: vPolicy.Namespace}, vPolicy))

	spec, err := specToHost(ctx, vPolicy, true)
	if err != nil {
		return nil, err
	}

	pPolicy.Spec = *spec
	return pPolicy, nil
}

func specToHost(ctx *synccontext.SyncContext, vPolicy *gatewayv1alpha3.BackendTLSPolicy, validateRefs bool) (*gatewayv1.BackendTLSPolicySpec, error) {
	retSpec := vPolicy.Spec.DeepCopy()
	for i := range retSpec.TargetRefs {
		err := routetranslate.PolicyTargetRefToHost(ctx, vPolicy.Namespace, &retSpec.TargetRefs[i], routetranslate.WithValidateHostObject(validateRefs))
		if err != nil {
			return nil, fmt.Errorf("translate targetRefs[%d]: %w", i, err)
		}
	}

	for i := range retSpec.Validation.CACertificateRefs {
		err := routetranslate.LocalObjectRefToHost(ctx, vPolicy.Namespace, &retSpec.Validation.CACertificateRefs[i], routetranslate.WithValidateHostObject(validateRefs))
		if err != nil {
			return nil, fmt.Errorf("translate validation.caCertificateRefs[%d]: %w", i, err)
		}
	}

	return retSpec, nil
}

func statusToVirtual(ctx *synccontext.SyncContext, hostPolicy *gatewayv1alpha3.BackendTLSPolicy, virtualPolicyNamespace string, status gatewayv1.PolicyStatus) (gatewayv1.PolicyStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Ancestors {
		// BackendTLSPolicy.Spec has no ParentRefs analog (only TargetRefs that point at
		// Services), so there is no spec-side reference whose explicit-namespace choice
		// we could mirror onto the status. Status namespaces collapse to nil whenever
		// the resolved virtual namespace matches the policy's namespace.
		err := routetranslate.ParentRefToVirtual(ctx, hostPolicy.Namespace, virtualPolicyNamespace, &retStatus.Ancestors[i].AncestorRef, nil)
		if err != nil {
			return gatewayv1.PolicyStatus{}, fmt.Errorf("translate ancestors[%d].ancestorRef: %w", i, err)
		}
	}

	return retStatus, nil
}
