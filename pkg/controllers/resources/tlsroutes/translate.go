package tlsroutes

import (
	"fmt"

	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (s *tlsRouteSyncer) translate(ctx *synccontext.SyncContext, vRoute *gatewayv1alpha2.TLSRoute) (*gatewayv1alpha2.TLSRoute, error) {
	pRoute := translate.HostMetadata(vRoute, s.VirtualToHost(ctx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, vRoute))

	spec, err := specToHost(ctx, vRoute, true)
	if err != nil {
		return nil, err
	}

	pRoute.Spec = *spec
	return pRoute, nil
}

func specToHost(ctx *synccontext.SyncContext, vRoute *gatewayv1alpha2.TLSRoute, validateRefs bool) (*gatewayv1alpha2.TLSRouteSpec, error) {
	if err := routetranslate.ValidateImportedGatewayHostnamePolicy(ctx, "TLSRoute", vRoute.Namespace, vRoute.Spec.ParentRefs, vRoute.Spec.Hostnames); err != nil {
		return nil, err
	}

	retSpec := vRoute.Spec.DeepCopy()
	for i := range retSpec.ParentRefs {
		err := gatewayauthz.TLSRouteAttachment(ctx, vRoute.Namespace, &retSpec.ParentRefs[i])
		if err != nil {
			return nil, fmt.Errorf("authorize parentRefs[%d]: %w", i, err)
		}

		err = routetranslate.ParentRefToHost(ctx, vRoute.Namespace, &retSpec.ParentRefs[i], routetranslate.WithValidateHostObject(validateRefs))
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

func statusToVirtual(ctx *synccontext.SyncContext, hostRoute *gatewayv1alpha2.TLSRoute, virtualRouteNamespace string, status gatewayv1alpha2.TLSRouteStatus) (gatewayv1alpha2.TLSRouteStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Parents {
		hostRouteNamespace := routetranslate.ParentStatusHostNamespace(hostRoute.Namespace, hostRoute.Spec.ParentRefs, retStatus.Parents[i].ParentRef)
		err := routetranslate.ParentRefToVirtual(ctx, hostRouteNamespace, virtualRouteNamespace, &retStatus.Parents[i].ParentRef, hostRoute.Spec.ParentRefs)
		if err != nil {
			return gatewayv1alpha2.TLSRouteStatus{}, fmt.Errorf("translate parents[%d].parentRef: %w", i, err)
		}
	}

	return retStatus, nil
}

func ruleToHost(ctx *synccontext.SyncContext, routeNamespace string, rule *gatewayv1alpha2.TLSRouteRule, translateOpts ...routetranslate.ToHostOption) error {
	for i := range rule.BackendRefs {
		err := gatewayauthz.TLSRouteBackend(ctx, routeNamespace, &rule.BackendRefs[i].BackendObjectReference)
		if err != nil {
			return fmt.Errorf("authorize backendRefs[%d]: %w", i, err)
		}

		err = routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &rule.BackendRefs[i].BackendObjectReference, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate backendRefs[%d]: %w", i, err)
		}
	}

	return nil
}
