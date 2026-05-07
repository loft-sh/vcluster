package backendtlspolicies

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *backendTLSPolicySyncer) translate(ctx *synccontext.SyncContext, vPolicy *gatewayv1.BackendTLSPolicy) (*gatewayv1.BackendTLSPolicy, error) {
	pPolicy := translate.HostMetadata(vPolicy, s.VirtualToHost(ctx, types.NamespacedName{Name: vPolicy.Name, Namespace: vPolicy.Namespace}, vPolicy))

	spec, err := translateSpecToHost(ctx, vPolicy, true)
	if err != nil {
		return nil, err
	}

	pPolicy.Spec = *spec
	return pPolicy, nil
}

func translateSpecToHost(ctx *synccontext.SyncContext, vPolicy *gatewayv1.BackendTLSPolicy, validateRefs bool) (*gatewayv1.BackendTLSPolicySpec, error) {
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

func translateStatusToVirtual(ctx *synccontext.SyncContext, hostPolicy *gatewayv1.BackendTLSPolicy, virtualPolicyNamespace string, status gatewayv1.PolicyStatus) (gatewayv1.PolicyStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Ancestors {
		err := routetranslate.ParentRefToVirtual(ctx, hostPolicy.Namespace, virtualPolicyNamespace, &retStatus.Ancestors[i].AncestorRef)
		if err != nil {
			return gatewayv1.PolicyStatus{}, fmt.Errorf("translate ancestors[%d].ancestorRef: %w", i, err)
		}
	}

	return retStatus, nil
}
