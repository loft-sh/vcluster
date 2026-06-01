package resources

import (
	_ "embed"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed gateways.crd.yaml
var gatewaysCRD string

func CreateGatewayMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	if ctx.Config.Sync.FromHost.Gateways.Enabled || ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		err := ensureHostGatewayAPIKind(ctx, mappings.Gateways(), "sync.fromHost.gateways.enabled or sync.toHost.gatewayApi.gateways.enabled")
		if err != nil {
			return nil, err
		}
	}

	err := EnsureGatewayClassCRD(ctx)
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(gatewaysCRD), mappings.Gateways())
	if err != nil {
		return nil, err
	}

	return NewImportedGatewayMapper(), nil
}

func NewImportedGatewayMapper() synccontext.Mapper {
	return &importedGatewayMapper{gvk: mappings.Gateways()}
}

type importedGatewayMapper struct {
	gvk schema.GroupVersionKind
}

func (m *importedGatewayMapper) GroupVersionKind() schema.GroupVersionKind {
	return m.gvk
}

func (m *importedGatewayMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (m *importedGatewayMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	if ctx == nil || ctx.Config == nil {
		return req
	}

	if host, ok := GatewayVirtualToHost(ctx, req); ok {
		return host
	}
	if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		return translate.Default.HostName(ctx, req.Name, req.Namespace)
	}
	return req
}

func (m *importedGatewayMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if ctx == nil || ctx.Config == nil {
		return req
	}

	if virtual, ok := GatewayHostToVirtual(ctx, req); ok {
		return virtual
	}
	if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		vName := generic.TryToTranslateBackByAnnotations(ctx, req, pObj, m.gvk)
		if vName.Name != "" {
			return vName
		}
		return generic.TryToTranslateBackByName(ctx, req, m.gvk)
	}

	return req
}

func (m *importedGatewayMapper) IsManaged(ctx *synccontext.SyncContext, obj client.Object) (bool, error) {
	if ctx == nil || ctx.Config == nil || obj == nil {
		return false, nil
	}
	host := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	if GatewayHostExactMapped(ctx, host) {
		return true, nil
	}
	if GatewayHostWildcardMapped(ctx, host.Namespace) {
		return ctx.Config.Sync.FromHost.Gateways.Selector.Matches(obj)
	}
	if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		return translate.Default.IsManaged(ctx, obj), nil
	}
	return false, nil
}

func GatewayHostToVirtual(ctx *synccontext.SyncContext, host types.NamespacedName) (types.NamespacedName, bool) {
	if ctx == nil || ctx.Config == nil {
		return types.NamespacedName{}, false
	}
	mappings := ctx.Config.Sync.FromHost.Gateways.Mappings.ByName
	if target, ok := mappings[host.Namespace+"/"+host.Name]; ok {
		return ParseGatewayNamespacedName(target, host.Name)
	}
	if target, ok := mappings[host.Namespace+"/*"]; ok {
		return ParseGatewayNamespacedName(target, host.Name)
	}
	return types.NamespacedName{}, false
}

func GatewayVirtualToHost(ctx *synccontext.SyncContext, virtual types.NamespacedName) (types.NamespacedName, bool) {
	if ctx == nil || ctx.Config == nil {
		return types.NamespacedName{}, false
	}
	for source, target := range ctx.Config.Sync.FromHost.Gateways.Mappings.ByName {
		targetName, ok := ParseGatewayNamespacedName(target, virtual.Name)
		if !ok || targetName != virtual {
			continue
		}
		return ParseGatewayNamespacedName(source, virtual.Name)
	}
	return types.NamespacedName{}, false
}

func GatewayHostCoveredByMapping(ctx *synccontext.SyncContext, host types.NamespacedName) bool {
	return GatewayHostExactMapped(ctx, host) || GatewayHostWildcardMapped(ctx, host.Namespace)
}

func GatewayHostExactMapped(ctx *synccontext.SyncContext, host types.NamespacedName) bool {
	if ctx == nil || ctx.Config == nil {
		return false
	}
	_, ok := ctx.Config.Sync.FromHost.Gateways.Mappings.ByName[host.Namespace+"/"+host.Name]
	return ok
}

func GatewayHostWildcardMapped(ctx *synccontext.SyncContext, hostNamespace string) bool {
	if ctx == nil || ctx.Config == nil {
		return false
	}
	_, ok := ctx.Config.Sync.FromHost.Gateways.Mappings.ByName[hostNamespace+"/*"]
	return ok
}

func GatewayTenantTargetMapped(ctx *synccontext.SyncContext, tenant types.NamespacedName) bool {
	_, ok := GatewayVirtualToHost(ctx, tenant)
	return ok
}

func GatewayMappedTenantNamespaces(ctx *synccontext.SyncContext) map[string]struct{} {
	namespaces := map[string]struct{}{}
	if ctx == nil || ctx.Config == nil {
		return namespaces
	}
	for _, target := range ctx.Config.Sync.FromHost.Gateways.Mappings.ByName {
		targetName, ok := ParseGatewayNamespacedName(target, "*")
		if ok {
			namespaces[targetName.Namespace] = struct{}{}
		}
	}
	return namespaces
}

func ParseGatewayNamespacedName(value, wildcardName string) (types.NamespacedName, bool) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return types.NamespacedName{}, false
	}
	name := parts[1]
	if name == "*" {
		name = wildcardName
	}
	return types.NamespacedName{Namespace: parts[0], Name: name}, true
}
