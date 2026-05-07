package tlsroutes

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *tlsRouteSyncer) translate(ctx *synccontext.SyncContext, vRoute *gatewayv1.TLSRoute) (*gatewayv1.TLSRoute, error) {
	pRoute := translate.HostMetadata(vRoute, s.VirtualToHost(ctx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, vRoute))

	spec, err := translateSpecToHost(ctx, vRoute, true)
	if err != nil {
		return nil, err
	}

	pRoute.Spec = *spec
	return pRoute, nil
}

func translateSpecToHost(ctx *synccontext.SyncContext, vRoute *gatewayv1.TLSRoute, validateRefs bool) (*gatewayv1.TLSRouteSpec, error) {
	retSpec := vRoute.Spec.DeepCopy()

	for i := range retSpec.ParentRefs {
		err := translateParentRefToHost(ctx, vRoute.Namespace, &retSpec.ParentRefs[i], validateRefs)
		if err != nil {
			return nil, fmt.Errorf("translate parentRefs[%d]: %w", i, err)
		}
	}

	for i := range retSpec.Rules {
		err := translateRuleToHost(ctx, vRoute.Namespace, &retSpec.Rules[i], validateRefs)
		if err != nil {
			return nil, fmt.Errorf("translate rules[%d]: %w", i, err)
		}
	}

	return retSpec, nil
}

func translateStatusToVirtual(ctx *synccontext.SyncContext, hostRoute *gatewayv1.TLSRoute, virtualRouteNamespace string, status gatewayv1.TLSRouteStatus) (gatewayv1.TLSRouteStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Parents {
		hostRouteNamespace := gatewayroutes.ParentStatusHostNamespace(hostRoute.Namespace, hostRoute.Spec.ParentRefs, retStatus.Parents[i].ParentRef)
		err := gatewayroutes.TranslateParentRefToVirtual(ctx, hostRouteNamespace, virtualRouteNamespace, &retStatus.Parents[i].ParentRef)
		if err != nil {
			return gatewayv1.TLSRouteStatus{}, fmt.Errorf("translate parents[%d].parentRef: %w", i, err)
		}
	}

	return retStatus, nil
}

func translateRuleToHost(ctx *synccontext.SyncContext, routeNamespace string, rule *gatewayv1.TLSRouteRule, validateRefs bool) error {
	for i := range rule.BackendRefs {
		err := translateBackendObjectRefToHost(ctx, routeNamespace, &rule.BackendRefs[i].BackendObjectReference, validateRefs)
		if err != nil {
			return fmt.Errorf("translate backendRefs[%d]: %w", i, err)
		}
	}

	return nil
}

func translateParentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference, validateRef bool) error {
	if validateRef {
		return gatewayroutes.TranslateParentRefToHost(ctx, routeNamespace, ref)
	}

	return gatewayroutes.TranslateParentRefToHostWithoutValidation(ctx, routeNamespace, ref)
}

func translateBackendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference, validateRef bool) error {
	if validateRef {
		return gatewayroutes.TranslateBackendObjectRefToHost(ctx, routeNamespace, ref)
	}

	return gatewayroutes.TranslateBackendObjectRefToHostWithoutValidation(ctx, routeNamespace, ref)
}
