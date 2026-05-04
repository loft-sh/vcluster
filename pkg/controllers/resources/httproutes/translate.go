package httproutes

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *httpRouteSyncer) translate(ctx *synccontext.SyncContext, vRoute *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRoute, error) {
	pRoute := translate.HostMetadata(vRoute, s.VirtualToHost(ctx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, vRoute))

	spec, err := specToHost(ctx, vRoute, true)
	if err != nil {
		return nil, err
	}

	pRoute.Spec = *spec
	return pRoute, nil
}

func specToHost(ctx *synccontext.SyncContext, vRoute *gatewayv1.HTTPRoute, validateRefs bool) (*gatewayv1.HTTPRouteSpec, error) {
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

func statusToVirtual(ctx *synccontext.SyncContext, hostRoute *gatewayv1.HTTPRoute, virtualRouteNamespace string, status gatewayv1.HTTPRouteStatus) (gatewayv1.HTTPRouteStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Parents {
		hostRouteNamespace := routetranslate.ParentStatusHostNamespace(hostRoute.Namespace, hostRoute.Spec.ParentRefs, retStatus.Parents[i].ParentRef)
		err := routetranslate.ParentRefToVirtual(ctx, hostRouteNamespace, virtualRouteNamespace, &retStatus.Parents[i].ParentRef, hostRoute.Spec.ParentRefs)
		if err != nil {
			return gatewayv1.HTTPRouteStatus{}, fmt.Errorf("translate parents[%d].parentRef: %w", i, err)
		}
	}

	return retStatus, nil
}

func ruleToHost(ctx *synccontext.SyncContext, routeNamespace string, rule *gatewayv1.HTTPRouteRule, translateOpts ...routetranslate.ToHostOption) error {
	for i := range rule.BackendRefs {
		err := httpBackendRefToHost(ctx, routeNamespace, &rule.BackendRefs[i], translateOpts...)
		if err != nil {
			return fmt.Errorf("translate backendRefs[%d]: %w", i, err)
		}
	}

	for i := range rule.Filters {
		err := filterToHost(ctx, routeNamespace, &rule.Filters[i], translateOpts...)
		if err != nil {
			return fmt.Errorf("translate filters[%d]: %w", i, err)
		}
	}

	return nil
}

func httpBackendRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.HTTPBackendRef, translateOpts ...routetranslate.ToHostOption) error {
	err := routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &ref.BackendObjectReference, translateOpts...)
	if err != nil {
		return err
	}

	for i := range ref.Filters {
		err := filterToHost(ctx, routeNamespace, &ref.Filters[i], translateOpts...)
		if err != nil {
			return fmt.Errorf("translate filters[%d]: %w", i, err)
		}
	}

	return nil
}

func filterToHost(ctx *synccontext.SyncContext, routeNamespace string, filter *gatewayv1.HTTPRouteFilter, translateOpts ...routetranslate.ToHostOption) error {
	if filter.RequestMirror != nil {
		err := routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &filter.RequestMirror.BackendRef, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate requestMirror.backendRef: %w", err)
		}
	}

	if filter.ExternalAuth != nil {
		err := routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &filter.ExternalAuth.BackendRef, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate externalAuth.backendRef: %w", err)
		}
	}

	return nil
}
