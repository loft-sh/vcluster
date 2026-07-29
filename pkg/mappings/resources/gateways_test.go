package resources

import (
	"context"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestImportedGatewayMapperUsesExactAndWildcardMappings(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{
		"networking/*":      "tenant-gateways/*",
		"platform/edge":     "shared-gateways/edge-public",
		"platform/internal": "shared-gateways/internal-public",
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	got := mapper.HostToVirtual(ctx, types.NamespacedName{Namespace: "networking", Name: "shared-edge"}, &gatewayv1.Gateway{})
	if got.Namespace != "tenant-gateways" || got.Name != "shared-edge" {
		t.Fatalf("expected wildcard mapping to tenant-gateways/shared-edge, got %s/%s", got.Namespace, got.Name)
	}

	got = mapper.HostToVirtual(ctx, types.NamespacedName{Namespace: "platform", Name: "edge"}, &gatewayv1.Gateway{})
	if got.Namespace != "shared-gateways" || got.Name != "edge-public" {
		t.Fatalf("expected exact mapping to shared-gateways/edge-public, got %s/%s", got.Namespace, got.Name)
	}

	back := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "shared-gateways", Name: "edge-public"}, &gatewayv1.Gateway{})
	if back.Namespace != "platform" || back.Name != "edge" {
		t.Fatalf("expected exact reverse mapping to platform/edge, got %s/%s", back.Namespace, back.Name)
	}

	back = mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "tenant-gateways", Name: "shared-edge"}, &gatewayv1.Gateway{})
	if back.Namespace != "networking" || back.Name != "shared-edge" {
		t.Fatalf("expected wildcard reverse mapping to networking/shared-edge, got %s/%s", back.Namespace, back.Name)
	}
}

func TestGatewayMapperUsesTenantTranslationForUnmappedTenantGateway(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"platform/edge": "shared-gateways/edge"}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	got := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "team-a", Name: "edge"}, &gatewayv1.Gateway{})
	expected := translate.Default.HostName(ctx, "edge", "team-a")
	if got != expected {
		t.Fatalf("expected tenant Gateway to use standard physical translation %s, got %s", expected.String(), got.String())
	}
}

func TestGatewayMapperUsesTenantTranslationUnderUmbrella(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Enabled = true
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	got := mapper.VirtualToHost(ctx, types.NamespacedName{Namespace: "team-a", Name: "edge"}, &gatewayv1.Gateway{})
	expected := translate.Default.HostName(ctx, "edge", "team-a")
	if got != expected {
		t.Fatalf("expected umbrella-enabled tenant Gateway to use standard physical translation %s, got %s", expected.String(), got.String())
	}
}

func TestImportedGatewayMapperManagedOnlyForMappedHostGateways(t *testing.T) {
	mapper := NewImportedGatewayMapper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{
		"networking/*":  "tenant-gateways/*",
		"platform/edge": "shared-gateways/edge-public",
	}
	vcConfig.Sync.FromHost.Gateways.Selector.MatchLabels = map[string]string{"expose": "true"}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	selectedWildcard := &gatewayv1.Gateway{}
	selectedWildcard.SetNamespace("networking")
	selectedWildcard.SetName("shared-edge")
	selectedWildcard.SetLabels(map[string]string{"expose": "true"})
	managed, err := mapper.IsManaged(ctx, selectedWildcard)
	if err != nil || !managed {
		t.Fatalf("expected selector-matching wildcard Gateway to be managed, managed=%v err=%v", managed, err)
	}

	unselectedWildcard := &gatewayv1.Gateway{}
	unselectedWildcard.SetNamespace("networking")
	unselectedWildcard.SetName("private-edge")
	managed, err = mapper.IsManaged(ctx, unselectedWildcard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Fatalf("expected selector-missing wildcard Gateway to be unmanaged")
	}

	exact := &gatewayv1.Gateway{}
	exact.SetNamespace("platform")
	exact.SetName("edge")
	managed, err = mapper.IsManaged(ctx, exact)
	if err != nil || !managed {
		t.Fatalf("expected exact mapped Gateway to bypass selector and be managed, managed=%v err=%v", managed, err)
	}

	unmapped := &gatewayv1.Gateway{}
	unmapped.SetNamespace("demo")
	unmapped.SetName("tenant-gw")
	unmapped.SetLabels(map[string]string{"expose": "true"})
	managed, err = mapper.IsManaged(ctx, unmapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Fatalf("expected unmapped host Gateway to be unmanaged")
	}
}
