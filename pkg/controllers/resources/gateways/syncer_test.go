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
	terminate := gatewayv1.TLSModeTerminate
	host := &gatewayv1.Gateway{Spec: gatewayv1.GatewaySpec{Infrastructure: &gatewayv1.GatewayInfrastructure{}, Listeners: []gatewayv1.Listener{{
		Name:     "https",
		Protocol: gatewayv1.HTTPSProtocolType,
		Port:     443,
		TLS: &gatewayv1.ListenerTLSConfig{
			Mode: &terminate,
			CertificateRefs: []gatewayv1.SecretObjectReference{{
				Name: "edge-cert",
			}},
		},
	}}}}

	spec := gatewaySpecToVirtual(ctx, host)
	if spec.Listeners[0].TLS == nil {
		t.Fatalf("expected TLS config to preserve mode while sanitizing certificateRefs")
	}
	if spec.Listeners[0].TLS.Mode == nil || *spec.Listeners[0].TLS.Mode != gatewayv1.TLSModeTerminate {
		t.Fatalf("expected TLS mode to be preserved, got %#v", spec.Listeners[0].TLS.Mode)
	}
	if len(spec.Listeners[0].TLS.CertificateRefs) != 0 {
		t.Fatalf("expected TLS certificateRefs to be sanitized, got %#v", spec.Listeners[0].TLS.CertificateRefs)
	}
	if spec.Listeners[0].TLS.Options[SanitizedCertificateRefsTLSOption] != "true" {
		t.Fatalf("expected sanitize marker option on Terminate listener so the mirror stays CRD-valid, got %#v", spec.Listeners[0].TLS.Options)
	}
	if spec.Infrastructure != nil {
		t.Fatalf("expected infrastructure to be sanitized")
	}
	if spec.Listeners[0].AllowedRoutes == nil || spec.Listeners[0].AllowedRoutes.Namespaces == nil || *spec.Listeners[0].AllowedRoutes.Namespaces.From != gatewayv1.NamespacesFromAll {
		t.Fatalf("expected tenant-facing allowedRoutes from All, got %#v", spec.Listeners[0].AllowedRoutes)
	}
}

func TestGatewaySpecToVirtualSanitizeKeepsTerminateListenerValid(t *testing.T) {
	terminate := gatewayv1.TLSModeTerminate
	passthrough := gatewayv1.TLSModePassthrough
	certRefs := []gatewayv1.SecretObjectReference{{Name: "edge-cert"}}

	tests := []struct {
		name        string
		sanitize    bool
		tls         *gatewayv1.ListenerTLSConfig
		wantRefs    int
		wantOptions map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue
		wantMode    *gatewayv1.TLSModeType
	}{
		{
			name:        "terminate mode without options gets marker",
			sanitize:    true,
			tls:         &gatewayv1.ListenerTLSConfig{Mode: &terminate, CertificateRefs: certRefs},
			wantRefs:    0,
			wantOptions: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{SanitizedCertificateRefsTLSOption: "true"},
			wantMode:    &terminate,
		},
		{
			name:        "nil mode defaults to terminate and gets marker",
			sanitize:    true,
			tls:         &gatewayv1.ListenerTLSConfig{CertificateRefs: certRefs},
			wantRefs:    0,
			wantOptions: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{SanitizedCertificateRefsTLSOption: "true"},
			wantMode:    nil,
		},
		{
			name:        "existing host options are preserved without marker",
			sanitize:    true,
			tls:         &gatewayv1.ListenerTLSConfig{Mode: &terminate, CertificateRefs: certRefs, Options: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{"example.com/min-version": "TLSv1_2"}},
			wantRefs:    0,
			wantOptions: map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue{"example.com/min-version": "TLSv1_2"},
			wantMode:    &terminate,
		},
		{
			name:        "passthrough mode needs no marker",
			sanitize:    true,
			tls:         &gatewayv1.ListenerTLSConfig{Mode: &passthrough, CertificateRefs: certRefs},
			wantRefs:    0,
			wantOptions: nil,
			wantMode:    &passthrough,
		},
		{
			name:        "sanitize disabled leaves refs and options untouched",
			sanitize:    false,
			tls:         &gatewayv1.ListenerTLSConfig{Mode: &terminate, CertificateRefs: certRefs},
			wantRefs:    1,
			wantOptions: nil,
			wantMode:    &terminate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vcConfig := &pkgconfig.VirtualClusterConfig{}
			vcConfig.Sync.FromHost.Gateways.Sanitize.CertificateRefs = tt.sanitize
			ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
			host := &gatewayv1.Gateway{Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
				Name:     "https",
				Protocol: gatewayv1.HTTPSProtocolType,
				Port:     443,
				TLS:      tt.tls,
			}}}}

			spec := gatewaySpecToVirtual(ctx, host)
			got := spec.Listeners[0].TLS
			if got == nil {
				t.Fatalf("expected TLS config to be preserved")
			}
			if len(got.CertificateRefs) != tt.wantRefs {
				t.Fatalf("expected %d certificateRefs, got %#v", tt.wantRefs, got.CertificateRefs)
			}
			if tt.wantMode == nil {
				if got.Mode != nil {
					t.Fatalf("expected nil TLS mode to stay nil, got %q", *got.Mode)
				}
			} else if got.Mode == nil || *got.Mode != *tt.wantMode {
				t.Fatalf("expected TLS mode %q to be preserved, got %#v", *tt.wantMode, got.Mode)
			}
			if len(got.Options) != len(tt.wantOptions) {
				t.Fatalf("expected options %#v, got %#v", tt.wantOptions, got.Options)
			}
			for k, v := range tt.wantOptions {
				if got.Options[k] != v {
					t.Fatalf("expected option %q=%q, got %#v", k, v, got.Options)
				}
			}
		})
	}
}

func TestGatewaySpecToVirtualAllowedHostnameOverrideInheritsDefaultNamespacePolicy(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy = &rootconfig.GatewayVirtualNamespacePolicy{From: string(gatewayv1.NamespacesFromAll)}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "shared-edge",
		AllowedHostnames: []string{"*.apps.example.com"},
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	host := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "shared-edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
			Name:     "http",
			Protocol: gatewayv1.HTTPProtocolType,
			Port:     80,
		}}},
	}

	spec := gatewaySpecToVirtual(ctx, host)
	if spec.Listeners[0].AllowedRoutes == nil || spec.Listeners[0].AllowedRoutes.Namespaces == nil || spec.Listeners[0].AllowedRoutes.Namespaces.From == nil {
		t.Fatalf("expected hostname-only override to inherit default tenant-facing allowedRoutes, got %#v", spec.Listeners[0].AllowedRoutes)
	}
	if *spec.Listeners[0].AllowedRoutes.Namespaces.From != gatewayv1.NamespacesFromAll {
		t.Fatalf("expected hostname-only override to inherit default namespace policy from All, got %#v", spec.Listeners[0].AllowedRoutes)
	}
}

func TestGatewaySpecToVirtualExplicitOverrideReplacesDefaultNamespacePolicy(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy = &rootconfig.GatewayVirtualNamespacePolicy{From: string(gatewayv1.NamespacesFromAll)}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace: "networking",
		Name:          "shared-edge",
		VirtualNamespacePolicy: rootconfig.GatewayVirtualNamespacePolicy{
			From: string(gatewayv1.NamespacesFromSelector),
			Selector: rootconfig.StandardLabelSelector{
				MatchLabels: map[string]string{"team": "apps"},
			},
		},
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	host := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "shared-edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
			Name:     "http",
			Protocol: gatewayv1.HTTPProtocolType,
			Port:     80,
		}}},
	}

	spec := gatewaySpecToVirtual(ctx, host)
	got := spec.Listeners[0].AllowedRoutes
	if got == nil || got.Namespaces == nil || got.Namespaces.From == nil || *got.Namespaces.From != gatewayv1.NamespacesFromSelector {
		t.Fatalf("expected explicit override selector policy, got %#v", got)
	}
	if got.Namespaces.Selector == nil || got.Namespaces.Selector.MatchLabels["team"] != "apps" {
		t.Fatalf("expected explicit override selector labels, got %#v", got.Namespaces.Selector)
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

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("sync tenant Gateway to host: %v", err)
	}

	expected := translate.Default.HostName(syncCtx, "edge", "team-a")
	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), expected, host); err != nil {
		t.Fatalf("expected host Gateway %s to be created: %v", expected.String(), err)
	}
	if host.Spec.GatewayClassName != "tenant-class" {
		t.Fatalf("expected host GatewayClassName to be preserved, got %q", host.Spec.GatewayClassName)
	}
}

func TestTenantGatewaySyncToHostSkipsUnsupportedParametersRef(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: "example.com", Kind: "GatewayConfig", Name: "params"})
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected unsupported parametersRef kind to be a warning/skip, not hard error: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), host); err == nil {
		t.Fatalf("did not expect host Gateway to be created for unsupported parametersRef kind")
	}
}

func TestTenantGatewaySyncDeletesHostOnUnsupportedParametersRef(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	hostName := types.NamespacedName{Namespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}
	host := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{
		Namespace: hostName.Namespace,
		Name:      hostName.Name,
		Labels:    map[string]string{translate.MarkerLabel: translate.VClusterName},
		Annotations: map[string]string{
			translate.NameAnnotation:      "edge",
			translate.NamespaceAnnotation: "team-a",
		},
	}}
	virtual := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: "example.com", Kind: "GatewayConfig", Name: "params"})
	pClient := testingutil.NewFakeClient(scheme.Scheme, host)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		virtual,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEvent(host, virtual))
	if err != nil {
		t.Fatalf("expected unsupported parametersRef kind on update to be a warning/skip, not hard error: %v", err)
	}

	got := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), hostName, got); err == nil {
		t.Fatalf("expected stale host Gateway to be deleted when virtual reference cannot be synced")
	}
}

func TestTenantGatewaySyncIgnoresImportedMirror(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	mirror := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "shared-gateways", Name: "edge", Labels: map[string]string{ImportedGatewayLabel: "true"}},
		Spec:       gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"},
	}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(mirror))
	if err != nil {
		t.Fatalf("expected imported mirror to be ignored by tenant Gateway syncer: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "shared-gateways"), host); err == nil {
		t.Fatalf("did not expect tenant Gateway syncer to create a host Gateway for imported mirror")
	}
}

func TestTenantGatewaySyncUpdatesManagedHostGatewayWhenVirtualExists(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	hostName := types.NamespacedName{Namespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}
	host := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{
		Namespace: hostName.Namespace,
		Name:      hostName.Name,
		Labels:    map[string]string{translate.MarkerLabel: translate.VClusterName},
		Annotations: map[string]string{
			translate.NameAnnotation:      "edge",
			translate.NamespaceAnnotation: "team-a",
		},
	}}
	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, host)
	vClient := testingutil.NewFakeClient(scheme.Scheme,
		virtual,
		&gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}},
	)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEvent(host, virtual))
	if err != nil {
		t.Fatalf("sync managed host Gateway with existing virtual Gateway: %v", err)
	}
	got := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), hostName, got); err != nil {
		t.Fatalf("expected managed host Gateway to remain: %v", err)
	}
	if got.Spec.GatewayClassName != "tenant-class" {
		t.Fatalf("expected managed host Gateway to be updated, got class %q", got.Spec.GatewayClassName)
	}
}

func TestTenantGatewaySyncToVirtualIgnoresImportedHostMapping(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"platform/edge": "shared-gateways/edge"}
	host := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "platform", Name: "edge"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, host)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	_, err := syncer.SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(host))
	if err != nil {
		t.Fatalf("expected imported host mapping to be ignored by tenant Gateway syncer: %v", err)
	}
	if err := pClient.Get(context.Background(), types.NamespacedName{Namespace: "platform", Name: "edge"}, &gatewayv1.Gateway{}); err != nil {
		t.Fatalf("expected imported host Gateway to remain: %v", err)
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

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "missing"}}
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

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "missing"}}
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

	mapped := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(mapped))
	if err != nil {
		t.Fatalf("expected mapped imported target to be a warning/skip, not hard error: %v", err)
	}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), &gatewayv1.Gateway{}); err == nil {
		t.Fatalf("did not expect mapped imported target Gateway to sync to host")
	}

	other := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "other"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
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

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
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
	existing := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: testingutil.DefaultTestTargetNamespace, Name: "edge-x-team-a-x-suffix"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "existing-class"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, existing)
	vClient := testingutil.NewFakeClient(scheme.Scheme, &gatewayv1.GatewayClass{ObjectMeta: metav1.ObjectMeta{Name: "tenant-class"}})
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx, object := syncertesting.FakeStartSyncer(t, registerCtx, NewToHost)
	syncer := object.(*tenantGatewaySyncer)

	virtual := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{GatewayClassName: "tenant-class"}}
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(virtual))
	if err != nil {
		t.Fatalf("expected unmanaged host conflict to be a warning/skip, not hard error: %v", err)
	}

	host := &gatewayv1.Gateway{}
	if err := pClient.Get(context.Background(), translate.Default.HostName(syncCtx, "edge", "team-a"), host); err != nil {
		t.Fatalf("expected existing host Gateway to remain: %v", err)
	}
	if host.Spec.GatewayClassName != "existing-class" {
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
