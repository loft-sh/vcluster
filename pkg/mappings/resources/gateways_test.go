package resources

import (
	"context"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestImportedGatewayMapperUsesVirtualNamespaceAndExplicitRename(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.VirtualNamespace = "vcluster-gateways"
	vcConfig.Sync.FromHost.Gateways.HostNamespaces = []string{"networking"}
	vcConfig.Sync.FromHost.Gateways.Imports = []rootconfig.GatewayImport{{
		HostNamespace: "platform",
		Name:          "shared-edge",
		VirtualName:   "platform-shared-edge",
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	got := mapper.HostToVirtual(ctx, types.NamespacedName{Namespace: "networking", Name: "shared-edge"}, &gatewayv1.Gateway{})
	if got.Namespace != "vcluster-gateways" || got.Name != "shared-edge" {
		t.Fatalf("expected networking/shared-edge to map to vcluster-gateways/shared-edge, got %s/%s", got.Namespace, got.Name)
	}

	got = mapper.HostToVirtual(ctx, types.NamespacedName{Namespace: "platform", Name: "shared-edge"}, &gatewayv1.Gateway{})
	if got.Namespace != "vcluster-gateways" || got.Name != "platform-shared-edge" {
		t.Fatalf("expected explicit import rename to map to vcluster-gateways/platform-shared-edge, got %s/%s", got.Namespace, got.Name)
	}

	back := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "vcluster-gateways", Name: "platform-shared-edge"}, &gatewayv1.Gateway{})
	if back.Namespace != "platform" || back.Name != "shared-edge" {
		t.Fatalf("expected explicit virtual name to reverse-map to platform/shared-edge, got %s/%s", back.Namespace, back.Name)
	}
}

func TestGatewayMapperUsesTenantTranslationOutsideImportNamespace(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.VirtualNamespace = "vcluster-gateways"
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	got := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "team-a", Name: "edge"}, &gatewayv1.Gateway{})
	expected := translate.Default.HostName(ctx, "edge", "team-a")
	if got != expected {
		t.Fatalf("expected tenant Gateway to use standard physical translation %s, got %s", expected.String(), got.String())
	}

	reserved := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "vcluster-gateways", Name: "edge"}, &gatewayv1.Gateway{})
	if reserved == expected {
		t.Fatalf("expected reserved import namespace Gateway to avoid tenant toHost translation")
	}
}

func TestImportedGatewayMapperManagedOnlyForSelectedHostGateways(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.HostNamespaces = []string{"networking"}
	vcConfig.Sync.FromHost.Gateways.Selector.MatchLabels = map[string]string{"expose": "true"}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	selected := &gatewayv1.Gateway{}
	selected.SetNamespace("networking")
	selected.SetName("shared-edge")
	selected.SetLabels(map[string]string{"expose": "true"})
	managed, err := mapper.IsManaged(ctx, selected)
	if err != nil || !managed {
		t.Fatalf("expected selected Gateway to be managed, managed=%v err=%v", managed, err)
	}

	unselected := &gatewayv1.Gateway{}
	unselected.SetNamespace("demo")
	unselected.SetName("tenant-gw")
	unselected.SetLabels(map[string]string{"expose": "true"})
	managed, err = mapper.IsManaged(ctx, unselected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Fatalf("expected Gateway outside configured hostNamespaces to be unmanaged")
	}
}
