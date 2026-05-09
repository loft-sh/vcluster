package httproutes

import (
	"fmt"
	"reflect"

	"github.com/loft-sh/vcluster/pkg/constants"
	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
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
		err := gatewayauthz.HTTPRouteAttachment(ctx, vRoute.Namespace, &retSpec.ParentRefs[i])
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
	err := gatewayauthz.HTTPRouteBackend(ctx, routeNamespace, &ref.BackendObjectReference)
	if err != nil {
		return err
	}

	err = routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &ref.BackendObjectReference, translateOpts...)
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
		err := gatewayauthz.HTTPRouteBackend(ctx, routeNamespace, &filter.RequestMirror.BackendRef)
		if err != nil {
			return fmt.Errorf("authorize requestMirror.backendRef: %w", err)
		}

		err = routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &filter.RequestMirror.BackendRef, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate requestMirror.backendRef: %w", err)
		}
	}

	if filter.ExternalAuth != nil {
		err := gatewayauthz.HTTPRouteBackend(ctx, routeNamespace, &filter.ExternalAuth.BackendRef)
		if err != nil {
			return fmt.Errorf("authorize externalAuth.backendRef: %w", err)
		}

		err = routetranslate.BackendObjectRefToHost(ctx, routeNamespace, &filter.ExternalAuth.BackendRef, translateOpts...)
		if err != nil {
			return fmt.Errorf("translate externalAuth.backendRef: %w", err)
		}
	}

	return nil
}

// preserveHostRule re-prepends a named host rule (identified by
// constants.PreserveHostRuleAnnotation) onto the desired host spec so it survives
// vCluster's re-derivation of spec from virtual. The annotation's value is the rule's
// HTTPRouteRule.Name. This is the extension hook for external host-side controllers
// that need to inject a high-priority managed rule which is not visible to the tenant.
func preserveHostRule(hostSpec gatewayv1.HTTPRouteSpec, desiredSpec *gatewayv1.HTTPRouteSpec, annotations map[string]string) {
	if desiredSpec == nil {
		return
	}

	ruleName := annotations[constants.PreserveHostRuleAnnotation]
	if ruleName == "" {
		return
	}
	targetName := gatewayv1.SectionName(ruleName)

	var hostRule *gatewayv1.HTTPRouteRule
	for i := range hostSpec.Rules {
		if hostSpec.Rules[i].Name != nil && *hostSpec.Rules[i].Name == targetName {
			hostRule = &hostSpec.Rules[i]
			break
		}
	}
	if hostRule == nil {
		return
	}

	for i := range desiredSpec.Rules {
		if desiredSpec.Rules[i].Name != nil && *desiredSpec.Rules[i].Name == targetName {
			return
		}
	}

	preserved := hostRule.DeepCopy()
	desiredSpec.Rules = append([]gatewayv1.HTTPRouteRule{*preserved}, desiredSpec.Rules...)
}

func preserveRequestMirrorFilters(hostSpec gatewayv1.HTTPRouteSpec, desiredSpec *gatewayv1.HTTPRouteSpec, annotations map[string]string) {
	if desiredSpec == nil || annotations[constants.PreserveRequestMirrorFiltersAnnotation] != "true" {
		return
	}

	hostRulesByName := make(map[gatewayv1.SectionName]*gatewayv1.HTTPRouteRule, len(hostSpec.Rules))
	for i := range hostSpec.Rules {
		if hostSpec.Rules[i].Name != nil {
			hostRulesByName[*hostSpec.Rules[i].Name] = &hostSpec.Rules[i]
		}
	}

	for i := range desiredSpec.Rules {
		hostRule := matchingHostRule(hostSpec, i, desiredSpec.Rules[i].Name, hostRulesByName)
		if hostRule == nil {
			continue
		}

		for _, filter := range hostRule.Filters {
			if !isRequestMirrorFilter(filter) || hasRequestMirrorFilter(desiredSpec.Rules[i].Filters, filter) {
				continue
			}

			desiredSpec.Rules[i].Filters = append(desiredSpec.Rules[i].Filters, filter)
		}
	}
}

// matchingHostRule returns the host-side rule whose mirror filters should be preserved onto the
// desired rule at index i. It prefers name-based correlation (Gateway API v1 HTTPRouteRule.Name)
// when both sides have named rules, and falls back to positional correlation otherwise. This keeps
// the original index-based behavior intact for unnamed routes while preventing misalignment when a
// host controller injects, removes, or reorders rules.
func matchingHostRule(
	hostSpec gatewayv1.HTTPRouteSpec,
	desiredIndex int,
	desiredName *gatewayv1.SectionName,
	hostRulesByName map[gatewayv1.SectionName]*gatewayv1.HTTPRouteRule,
) *gatewayv1.HTTPRouteRule {
	if desiredName != nil {
		if rule, ok := hostRulesByName[*desiredName]; ok {
			return rule
		}
		// Desired rule is named but host has no rule with that name — do not silently fall back
		// to positional matching, which would attach the host's mirror filter to a semantically
		// different rule.
		return nil
	}
	if desiredIndex >= len(hostSpec.Rules) {
		return nil
	}
	return &hostSpec.Rules[desiredIndex]
}

func hasRequestMirrorFilter(filters []gatewayv1.HTTPRouteFilter, searchFilter gatewayv1.HTTPRouteFilter) bool {
	for _, filter := range filters {
		if isRequestMirrorFilter(filter) && reflect.DeepEqual(filter.RequestMirror.BackendRef, searchFilter.RequestMirror.BackendRef) {
			return true
		}
	}

	return false
}

func isRequestMirrorFilter(filter gatewayv1.HTTPRouteFilter) bool {
	return filter.Type == gatewayv1.HTTPRouteFilterRequestMirror && filter.RequestMirror != nil
}
