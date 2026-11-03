package translate

import (
	"fmt"
	"strings"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ValidateImportedGatewayHostnamePolicy enforces allowed hostnames configured for imported Gateways.
func ValidateImportedGatewayHostnamePolicy(ctx *synccontext.SyncContext, routeKind, routeNamespace string, parentRefs []gatewayv1.ParentReference, hostnames []gatewayv1.Hostname) error {
	if ctx == nil || ctx.Config == nil {
		return nil
	}

	for _, parent := range parentRefs {
		imp := importedGatewayForParent(ctx, routeNamespace, parent)
		if imp == nil || len(imp.AllowedHostnames) == 0 {
			continue
		}
		if len(hostnames) == 0 {
			return fmt.Errorf("%s with no hostnames is not permitted by imported Gateway %q hostname policy", routeKind, parent.Name)
		}
		for _, hostname := range hostnames {
			if !hostnameAllowed(string(hostname), imp.AllowedHostnames) {
				return fmt.Errorf("hostname %q is not permitted by imported Gateway %q hostname policy", hostname, parent.Name)
			}
		}
	}
	return nil
}

func importedGatewayForParent(ctx *synccontext.SyncContext, routeNamespace string, parent gatewayv1.ParentReference) *rootconfig.GatewayAllowedRoutesPolicyOverride {
	if parent.Group != nil && string(*parent.Group) != gatewayv1.GroupVersion.Group {
		return nil
	}
	if parent.Kind != nil && string(*parent.Kind) != "Gateway" {
		return nil
	}

	parentNamespace := routeNamespace
	if parent.Namespace != nil && *parent.Namespace != "" {
		parentNamespace = string(*parent.Namespace)
	}
	host, ok := resources.GatewayVirtualToHost(ctx, types.NamespacedName{Namespace: parentNamespace, Name: string(parent.Name)})
	if !ok {
		return nil
	}

	for i := range ctx.Config.Sync.FromHost.Gateways.AllowedRoutes.Overrides {
		override := &ctx.Config.Sync.FromHost.Gateways.AllowedRoutes.Overrides[i]
		if override.HostNamespace == host.Namespace && override.Name == host.Name {
			return override
		}
	}
	return nil
}

func hostnameAllowed(hostname string, allowed []string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	for _, pattern := range allowed {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == hostname {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(hostname, suffix) && hostname != strings.TrimPrefix(suffix, ".") {
				return true
			}
		}
	}
	return false
}
