package gateways

import (
	"context"
	"strings"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
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

func TestTenantGatewaySyncSkipsUnavailableVirtualGatewayClass(t *testing.T) {
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
		t.Fatalf("expected unavailable GatewayClass to skip without reconcile error, got %v", err)
	}

	expected := translate.Default.HostName(syncCtx, "edge", "team-a")
	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), expected, host); err == nil {
		t.Fatalf("did not expect host Gateway to be created without virtual GatewayClass")
	}
}

func TestTenantGatewaySyncDeletesExistingHostWhenGatewayClassBecomesUnavailable(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	hostName := types.NamespacedName{Namespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}
	host := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: hostName.Namespace, Name: hostName.Name}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, host)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("missing")}}
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEvent(host, virtual))
	if err != nil {
		t.Fatalf("expected unavailable GatewayClass update to delete host without reconcile error, got %v", err)
	}

	got := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), hostName, got); err == nil {
		t.Fatalf("expected existing host Gateway to be deleted when GatewayClass becomes unavailable")
	}
}

func TestTenantGatewaySyncSkipsMappedImportedTargetButAllowsSameNamespace(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"platform/edge": "team-a/edge"}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	mapped := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(mapped))
	if err != nil {
		t.Fatalf("expected mapped imported target to be a warning/skip, not hard error: %v", err)
	}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), &gatewayv1.Gateway{}); err == nil {
		t.Fatalf("did not expect mapped imported target Gateway to sync to host")
	}

	other := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "other"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName("tenant-class")}}
	_, err = syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(other))
	if err != nil {
		t.Fatalf("expected non-mapped Gateway in same namespace to sync: %v", err)
	}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "other", "team-a"), &gatewayv1.Gateway{}); err != nil {
		t.Fatalf("expected non-mapped Gateway in same namespace to sync to host: %v", err)
	}
}

func TestTenantGatewaySyncSkipsImportedHostNameConflict(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{testingutil.DefaultTestTargetNamespace + "/edge-x-team-a-x-suffix": "shared-gateways/edge"}
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

func TestEnsureVirtualNamespaceCreatesRequestedTenantNamespace(t *testing.T) {
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	ctx := &synccontext.SyncContext{Context: context.Background(), VirtualClient: vClient}

	if err := ensureVirtualNamespace(ctx, "shared-gateways"); err != nil {
		t.Fatalf("ensure virtual namespace: %v", err)
	}
	coreNS := &corev1.Namespace{}
	if err := vClient.Get(context.Background(), types.NamespacedName{Name: "shared-gateways"}, coreNS); err != nil {
		t.Fatalf("expected namespace to be created: %v", err)
	}
}

func TestGatewaySelectedUsesMappingsAndExactMappingsBypassSelector(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{
		"networking/*":  "tenant-gateways/*",
		"platform/edge": "shared-gateways/edge",
	}
	vcConfig.Sync.FromHost.Gateways.Selector.MatchLabels = map[string]string{"import": "yes"}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}

	exact := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "platform", Name: "edge"}}
	selected, reason, err := gatewaySelected(ctx, exact)
	if err != nil || !selected || reason != "" {
		t.Fatalf("expected exact mapping to be selected regardless of selector, selected=%v reason=%q err=%v", selected, reason, err)
	}

	wildcardSelected := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "edge", Labels: map[string]string{"import": "yes"}}}
	selected, reason, err = gatewaySelected(ctx, wildcardSelected)
	if err != nil || !selected || reason != "" {
		t.Fatalf("expected selector-matching wildcard mapping to be selected, selected=%v reason=%q err=%v", selected, reason, err)
	}

	wildcardUnselected := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "internal"}}
	selected, reason, err = gatewaySelected(ctx, wildcardUnselected)
	if err != nil || selected || !strings.Contains(reason, "selector") {
		t.Fatalf("expected selector-missing wildcard mapping to be ignored, selected=%v reason=%q err=%v", selected, reason, err)
	}

	unmapped := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "demo", Name: "edge", Labels: map[string]string{"import": "yes"}}}
	selected, reason, err = gatewaySelected(ctx, unmapped)
	if err != nil || selected || !strings.Contains(reason, "not covered") {
		t.Fatalf("expected unmapped Gateway to be ignored, selected=%v reason=%q err=%v", selected, reason, err)
	}
}
