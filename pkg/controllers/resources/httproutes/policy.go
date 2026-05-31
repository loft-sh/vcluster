package httproutes

import (
	"fmt"
	"strings"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func validateImportedGatewayHostnamePolicy(ctx *synccontext.SyncContext, route *gatewayv1.HTTPRoute) error {
	if ctx == nil || ctx.Config == nil || route == nil {
		return nil
	}

	for _, parent := range route.Spec.ParentRefs {
		imp := importedGatewayForParent(ctx, route.Namespace, parent)
		if imp == nil || len(imp.AllowedHostnames) == 0 {
			continue
		}
		for _, hostname := range route.Spec.Hostnames {
			if !hostnameAllowed(string(hostname), imp.AllowedHostnames) {
				return fmt.Errorf("hostname %q is not permitted by imported Gateway %q hostname policy", hostname, parent.Name)
			}
		}
	}
	return nil
}

func importedGatewayForParent(ctx *synccontext.SyncContext, routeNamespace string, parent gatewayv1.ParentReference) *rootconfig.GatewayImport {
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
	virtualNamespace := ctx.Config.Sync.FromHost.Gateways.VirtualNamespace
	if virtualNamespace == "" {
		virtualNamespace = "vcluster-gateways"
	}
	if parentNamespace != virtualNamespace {
		return nil
	}

	for i := range ctx.Config.Sync.FromHost.Gateways.Imports {
		imp := &ctx.Config.Sync.FromHost.Gateways.Imports[i]
		virtualName := imp.VirtualName
		if virtualName == "" {
			virtualName = imp.Name
		}
		if virtualName == string(parent.Name) {
			return imp
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
