package tlsroutes

import (
	"context"
	"strings"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestTLSRouteSpecToHostRejectsDisallowedImportedGatewayHostname(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "shared-edge",
		AllowedHostnames: []string{"*.team-a.example.com"},
	}}

	fromAll := gatewayv1.NamespacesFromAll
	parentNamespace := gatewayv1.Namespace("tenant-gateways")
	virtualGateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant-gateways", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
			Name:     "tls",
			Port:     443,
			Protocol: gatewayv1.TLSProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{
				From: &fromAll,
			}},
		}}},
	}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme, virtualGateway)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	ctx := registerCtx.ToSyncContext("tlsroute-test")
	ctx.Context = context.Background()

	route := &gatewayv1.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "app"},
		Spec: gatewayv1.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "edge", Namespace: &parentNamespace}}},
			Hostnames:       []gatewayv1.Hostname{"admin.example.com"},
		},
	}

	_, err := specToHost(ctx, route, false)
	if err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("expected disallowed TLSRoute hostname to be rejected, got %v", err)
	}
}
