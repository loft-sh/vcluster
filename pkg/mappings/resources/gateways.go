package resources

import (
	_ "embed"

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
	if ctx.Config.Sync.FromHost.Gateways.Enabled {
		err := ensureHostGatewayAPIKind(ctx, mappings.Gateways(), "sync.fromHost.gateways.enabled")
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

	virtualNamespace := gatewayVirtualNamespace(ctx)
	if req.Namespace != virtualNamespace {
		if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
			return translate.Default.HostName(ctx, req.Name, req.Namespace)
		}
		return req
	}

	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		virtualName := imp.VirtualName
		if virtualName == "" {
			virtualName = imp.Name
		}
		if virtualName == req.Name {
			return types.NamespacedName{Namespace: imp.HostNamespace, Name: imp.Name}
		}
	}

	if len(ctx.Config.Sync.FromHost.Gateways.HostNamespaces) == 1 {
		return types.NamespacedName{Namespace: ctx.Config.Sync.FromHost.Gateways.HostNamespaces[0], Name: req.Name}
	}

	return req
}

func (m *importedGatewayMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if ctx == nil || ctx.Config == nil {
		return req
	}

	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		if imp.HostNamespace == req.Namespace && imp.Name == req.Name {
			virtualName := imp.VirtualName
			if virtualName == "" {
				virtualName = imp.Name
			}
			return types.NamespacedName{Namespace: gatewayVirtualNamespace(ctx), Name: virtualName}
		}
	}

	if containsGatewayHostNamespace(ctx.Config.Sync.FromHost.Gateways.HostNamespaces, req.Namespace) {
		return types.NamespacedName{Namespace: gatewayVirtualNamespace(ctx), Name: req.Name}
	}

	if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		vName := generic.TryToTranslateBackByAnnotations(ctx, req, pObj, m.gvk)
		if vName.Name != "" {
			return vName
		}
		return generic.TryToTranslateBackByName(ctx, req, m.gvk)
	}

	return types.NamespacedName{Namespace: gatewayVirtualNamespace(ctx), Name: req.Name}
}

func (m *importedGatewayMapper) IsManaged(ctx *synccontext.SyncContext, obj client.Object) (bool, error) {
	if ctx == nil || ctx.Config == nil || obj == nil {
		return false, nil
	}
	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		if imp.HostNamespace == obj.GetNamespace() && imp.Name == obj.GetName() {
			return true, nil
		}
	}
	if containsGatewayHostNamespace(ctx.Config.Sync.FromHost.Gateways.HostNamespaces, obj.GetNamespace()) {
		return ctx.Config.Sync.FromHost.Gateways.Selector.Matches(obj)
	}
	if ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled {
		return translate.Default.IsManaged(ctx, obj), nil
	}
	return false, nil
}

func containsGatewayHostNamespace(namespaces []string, namespace string) bool {
	for _, candidate := range namespaces {
		if candidate == namespace {
			return true
		}
	}
	return false
}

func gatewayVirtualNamespace(ctx *synccontext.SyncContext) string {
	if ctx != nil && ctx.Config != nil && ctx.Config.Sync.FromHost.Gateways.VirtualNamespace != "" {
		return ctx.Config.Sync.FromHost.Gateways.VirtualNamespace
	}
	return "vcluster-gateways"
}
