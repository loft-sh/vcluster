package tlsroutes

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *tlsRouteSyncer) translate(ctx *synccontext.SyncContext, vRoute *gatewayv1.TLSRoute) (*gatewayv1.TLSRoute, error) {
	pRoute := translate.HostMetadata(vRoute, s.VirtualToHost(ctx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, vRoute))

	spec, err := specToHost(ctx, vRoute, true)
	if err != nil {
		return nil, err
	}

	pRoute.Spec = *spec
	return pRoute, nil
}

func specToHost(ctx *synccontext.SyncContext, vRoute *gatewayv1.TLSRoute, validateRefs bool) (*gatewayv1.TLSRouteSpec, error) {
	retSpec := vRoute.Spec.DeepCopy()
	for i := range retSpec.ParentRefs {
		err := routetranslate.ParentRefToHost(ctx, vRoute.Namespace, &retSpec.ParentRefs[i], routetranslate.WithValidateHostObject(validateRefs))
		if err != nil {
			return nil, fmt.Errorf("translate parentRefs[%d]: %w", i, err)
		}
	}

	for i := range retSpec.Rules {
		err := ruleToHost(ctx, vRoute.Namespace, &retSpec.Rules[i], routetranslate.WithValidateHostObject(validateRefs))
		if err != nil {
			return nil, fmt.Errorf("translate rules[%d]: %w", i, err)
		}
	}

	return retSpec, nil
}

func statusToVirtual(ctx *synccontext.SyncContext, hostRoute *gatewayv1.TLSRoute, virtualRouteNamespace string, status gatewayv1.TLSRouteStatus) (gatewayv1.TLSRouteStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Parents {
		hostRouteNamespace := routetranslate.ParentStatusHostNamespace(hostRoute.Namespace, hostRoute.Spec.ParentRefs, retStatus.Parents[i].ParentRef)
		err := routetranslate.ParentRefToVirtual(ctx, hostRouteNamespace, virtualRouteNamespace, &retStatus.Parents[i].ParentRef)
		if err != nil {
			return gatewayv1.TLSRouteStatus{}, fmt.Errorf("translate parents[%d].parentRef: %w", i, err)
		}
	}

	return retStatus, nil
}

func ruleToHost(ctx *synccontext.SyncContext, routeNamespace string, rule *gatewayv1.TLSRouteRule, translateOpts ...routetranslate.ToHostOption) error {
	for i := range rule.BackendRefs {
		err := routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &rule.BackendRefs[i].BackendObjectReference, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate backendRefs[%d]: %w", i, err)
		}
	}

	return nil
}
