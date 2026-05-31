package gateways

import (
	"context"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGatewaySpecToVirtualSanitizesTLSInfrastructureAndAppliesVirtualAllowedRoutes(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Sanitize.CertificateRefs = true
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy = &rootconfig.GatewayVirtualNamespacePolicy{From: string(gatewayv1.NamespacesFromAll)}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	vcConfig.Sync.FromHost.Gateways.Sanitize.Infrastructure = true
	host := &gatewayv1.Gateway{Spec: gatewayv1.GatewaySpec{Infrastructure: &gatewayv1.GatewayInfrastructure{}, Listeners: []gatewayv1.Listener{{
		Name:     "https",
		Protocol: gatewayv1.HTTPSProtocolType,
		Port:     443,
		TLS: &gatewayv1.ListenerTLSConfig{CertificateRefs: []gatewayv1.SecretObjectReference{{
			Name: "edge-cert",
		}}},
	}}}}

	spec := gatewaySpecToVirtual(ctx, host)
	if spec.Listeners[0].TLS != nil {
		t.Fatalf("expected TLS config to be sanitized")
	}
	if spec.Infrastructure != nil {
		t.Fatalf("expected infrastructure to be sanitized")
	}
	if spec.Listeners[0].AllowedRoutes == nil || spec.Listeners[0].AllowedRoutes.Namespaces == nil || *spec.Listeners[0].AllowedRoutes.Namespaces.From != gatewayv1.NamespacesFromAll {
		t.Fatalf("expected tenant-facing allowedRoutes from All, got %#v", spec.Listeners[0].AllowedRoutes)
	}
}

func TestGatewayStatusToVirtualHidesAddressesByDefault(t *testing.T) {
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &pkgconfig.VirtualClusterConfig{}}
	status := gatewayv1.GatewayStatus{Addresses: []gatewayv1.GatewayStatusAddress{{Value: "1.2.3.4"}}}

	got := gatewayStatusToVirtual(ctx, status)
	if len(got.Addresses) != 0 {
		t.Fatalf("expected addresses to be hidden by default")
	}
}

func TestGatewayStatusToVirtualCanExposeAddresses(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Status.ExposeAddresses = true
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	status := gatewayv1.GatewayStatus{Addresses: []gatewayv1.GatewayStatusAddress{{Value: "1.2.3.4"}}}

	got := gatewayStatusToVirtual(ctx, status)
	if len(got.Addresses) != 1 || got.Addresses[0].Value != "1.2.3.4" {
		t.Fatalf("expected addresses to be exposed, got %#v", got.Addresses)
	}
}

func TestToGatewayAllowedRoutesSelectorPolicy(t *testing.T) {
	policy := &rootconfig.GatewayVirtualNamespacePolicy{From: string(gatewayv1.NamespacesFromSelector)}
	policy.Selector.MatchLabels = map[string]string{"team": "a"}

	got := toGatewayAllowedRoutes(policy)
	if got == nil || got.Namespaces == nil || got.Namespaces.Selector == nil {
		t.Fatalf("expected selector allowedRoutes, got %#v", got)
	}
	if got.Namespaces.Selector.MatchLabels["team"] != "a" {
		t.Fatalf("expected selector labels to be preserved, got %#v", got.Namespaces.Selector.MatchLabels)
	}
}

func TestTenantGatewaySyncToHostCreatesGatewayWhenExplicitlyEnabled(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("sync tenant Gateway to host: %v", err)
	}

	expected := translate.Default.HostName(syncCtx, "edge", "team-a")
	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), expected, host); err != nil {
		t.Fatalf("expected host Gateway %s to be created: %v", expected.String(), err)
	}
	if host.Spec.GatewayClassName != gatewayv1.ObjectName("tenant-class") {
		t.Fatalf("expected host GatewayClassName to be preserved, got %q", host.Spec.GatewayClassName)
	}
}

func TestTenantGatewaySyncRequiresVirtualGatewayClass(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("missing")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected missing GatewayClass to be a warning/skip, not hard error: %v", err)
	}

	expected := translate.Default.HostName(syncCtx, "edge", "team-a")
	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), expected, host); err == nil {
		t.Fatalf("did not expect host Gateway to be created without virtual GatewayClass")
	}
}

func TestTenantGatewaySyncSkipsReservedImportNamespace(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.VirtualNamespace = "vcluster-gateways"
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "vcluster-gateways", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected reserved namespace to be a warning/skip, not hard error: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), types.NamespacedName{Namespace: "vcluster-gateways", Name: "edge"}, host); err == nil {
		t.Fatalf("did not expect reserved imported mirror namespace Gateway to sync to host")
	}
}

func TestTenantGatewaySyncSkipsImportedHostNameConflict(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Imports = []rootconfig.GatewayImport{{HostNamespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme, &gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}})
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected imported host conflict to be a warning/skip, not hard error: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), host); err == nil {
		t.Fatalf("did not expect host Gateway to be created over imported Gateway conflict")
	}
}

func TestTenantGatewaySyncDoesNotOverwriteUnmanagedHostGateway(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	existing := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("existing-class")}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, existing)
	vClient := testingutil.NewFakeClient(scheme.Scheme, &gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}})
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected unmanaged host conflict to be a warning/skip, not hard error: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), host); err != nil {
		t.Fatalf("expected existing host Gateway to remain: %v", err)
	}
	if host.Spec.GatewayClassName != gatewayv1.ObjectName("existing-class") {
		t.Fatalf("expected unmanaged host Gateway to remain untouched, got class %q", host.Spec.GatewayClassName)
	}
}

func TestGatewaySelectedExplicitImportOverridesSelector(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Imports = []rootconfig.GatewayImport{{HostNamespace: "networking", Name: "edge"}}
	vcConfig.Sync.FromHost.Gateways.Selector.MatchLabels = map[string]string{"import": "yes"}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	host := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "edge"}}

	selected, reason, err := gatewaySelected(ctx, host)
	if err != nil || !selected || reason != "" {
		t.Fatalf("expected explicit import to be selected regardless of selector, selected=%v reason=%q err=%v", selected, reason, err)
	}
}
