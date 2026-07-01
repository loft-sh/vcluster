package gateways

import (
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestSpecToHostTranslatesDefaultFrontendCACertificateRefs(t *testing.T) {
	tests := []struct {
		name      string
		ref       gatewayv1.ObjectReference
		configMap *corev1.ConfigMap
		secret    *corev1.Secret
	}{
		{
			name:      "configmap",
			ref:       gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca"},
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "client-ca"}},
		},
		{
			name:   "secret",
			ref:    gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "Secret", Name: "client-ca"},
			secret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "client-ca"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, tt.configMap, tt.secret)
			expected := utiltranslate.Default.HostName(syncCtx, "client-ca", "team-a")
			gateway := gatewayWithDefaultFrontendCACertificateRefs(tt.ref)

			spec, err := specToHost(syncCtx, gateway, true)
			if err != nil {
				t.Fatalf("translate Gateway spec: %v", err)
			}
			gotRefs := spec.TLS.Frontend.Default.Validation.CACertificateRefs
			if len(gotRefs) != 1 {
				t.Fatalf("expected one default frontend CA certificate ref, got %#v", gotRefs)
			}
			if gotRefs[0].Name != gatewayv1.ObjectName(expected.Name) {
				t.Fatalf("expected CA certificate ref name %q, got %q", expected.Name, gotRefs[0].Name)
			}
			if gateway.Spec.TLS.Frontend.Default.Validation.CACertificateRefs[0].Name != "client-ca" {
				t.Fatalf("expected virtual Gateway to stay unchanged, got %q", gateway.Spec.TLS.Frontend.Default.Validation.CACertificateRefs[0].Name)
			}
		})
	}
}

func TestSpecToHostAllowsGrantedCrossNamespaceDefaultFrontendCACertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	caNamespace := gatewayv1.Namespace("security")
	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "allow-gateway-ca"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("Gateway"), Namespace: gatewayv1.Namespace("team-a")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("ConfigMap"), Name: ptr.To(gatewayv1.ObjectName("client-ca"))}},
		},
	}
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-ca"}},
		nil,
		grant,
	)
	expected := utiltranslate.Default.HostName(syncCtx, "client-ca", "security")
	gateway := gatewayWithDefaultFrontendCACertificateRefs(gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca", Namespace: &caNamespace})

	spec, err := specToHost(syncCtx, gateway, true)
	if err != nil {
		t.Fatalf("translate Gateway spec: %v", err)
	}
	got := spec.TLS.Frontend.Default.Validation.CACertificateRefs[0]
	if got.Name != gatewayv1.ObjectName(expected.Name) || got.Namespace == nil || *got.Namespace != gatewayv1.Namespace(expected.Namespace) {
		t.Fatalf("expected CA certificate ref %s/%s, got %#v", expected.Namespace, expected.Name, got)
	}
}

func TestSpecToHostRequiresReferenceGrantForCrossNamespaceDefaultFrontendCACertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	caNamespace := gatewayv1.Namespace("security")
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-ca"}},
		nil,
	)
	gateway := gatewayWithDefaultFrontendCACertificateRefs(gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca", Namespace: &caNamespace})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "ReferenceGrant") {
		t.Fatalf("expected cross-namespace default frontend CA certificate ref to require ReferenceGrant, got %v", err)
	}
}

func TestSpecToHostAllowsGrantedCrossNamespacePerPortFrontendCACertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	caNamespace := gatewayv1.Namespace("security")
	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "allow-gateway-ca"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("Gateway"), Namespace: gatewayv1.Namespace("team-a")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("ConfigMap"), Name: ptr.To(gatewayv1.ObjectName("client-ca"))}},
		},
	}
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-ca"}},
		nil,
		grant,
	)
	expected := utiltranslate.Default.HostName(syncCtx, "client-ca", "security")
	gateway := gatewayWithPerPortFrontendCACertificateRefs(443, gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca", Namespace: &caNamespace})

	spec, err := specToHost(syncCtx, gateway, true)
	if err != nil {
		t.Fatalf("translate Gateway spec: %v", err)
	}
	got := spec.TLS.Frontend.PerPort[0].TLS.Validation.CACertificateRefs[0]
	if got.Name != gatewayv1.ObjectName(expected.Name) || got.Namespace == nil || *got.Namespace != gatewayv1.Namespace(expected.Namespace) {
		t.Fatalf("expected per-port CA certificate ref %s/%s, got %#v", expected.Namespace, expected.Name, got)
	}
	if gateway.Spec.TLS.Frontend.PerPort[0].TLS.Validation.CACertificateRefs[0].Name != "client-ca" {
		t.Fatalf("expected virtual Gateway to stay unchanged, got %q", gateway.Spec.TLS.Frontend.PerPort[0].TLS.Validation.CACertificateRefs[0].Name)
	}
}

func TestSpecToHostSkipsNilPerPortFrontendTLSValidationAndTranslatesLaterRefs(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "client-ca"}},
		nil,
	)
	expected := utiltranslate.Default.HostName(syncCtx, "client-ca", "team-a")
	gateway := gatewayWithPerPortFrontendTLSConfigs(
		gatewayv1.TLSPortConfig{
			Port: 80,
			TLS:  gatewayv1.TLSConfig{},
		},
		gatewayv1.TLSPortConfig{
			Port: 443,
			TLS: gatewayv1.TLSConfig{
				Validation: &gatewayv1.FrontendTLSValidation{
					CACertificateRefs: []gatewayv1.ObjectReference{{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca"}},
				},
			},
		},
	)

	spec, err := specToHost(syncCtx, gateway, true)
	if err != nil {
		t.Fatalf("translate Gateway spec: %v", err)
	}
	if spec.TLS.Frontend.PerPort[0].TLS.Validation != nil {
		t.Fatalf("expected nil per-port TLS validation to stay nil, got %#v", spec.TLS.Frontend.PerPort[0].TLS.Validation)
	}
	got := spec.TLS.Frontend.PerPort[1].TLS.Validation.CACertificateRefs[0]
	if got.Name != gatewayv1.ObjectName(expected.Name) {
		t.Fatalf("expected later per-port CA certificate ref name %q, got %q", expected.Name, got.Name)
	}
	if gateway.Spec.TLS.Frontend.PerPort[1].TLS.Validation.CACertificateRefs[0].Name != "client-ca" {
		t.Fatalf("expected virtual Gateway to stay unchanged, got %q", gateway.Spec.TLS.Frontend.PerPort[1].TLS.Validation.CACertificateRefs[0].Name)
	}
}

func TestSpecToHostReportsPerPortFrontendCACertificateRefIndexes(t *testing.T) {
	caNamespace := gatewayv1.Namespace("security")
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "client-ca"}},
		nil,
	)
	gateway := gatewayWithPerPortFrontendTLSConfigs(
		gatewayv1.TLSPortConfig{
			Port: 80,
			TLS:  gatewayv1.TLSConfig{},
		},
		gatewayv1.TLSPortConfig{
			Port: 443,
			TLS: gatewayv1.TLSConfig{
				Validation: &gatewayv1.FrontendTLSValidation{
					CACertificateRefs: []gatewayv1.ObjectReference{
						{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca"},
						{Group: corev1.GroupName, Kind: "ConfigMap", Name: "other-client-ca", Namespace: &caNamespace},
					},
				},
			},
		},
	)

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "ReferenceGrant") || !strings.Contains(err.Error(), "tls.frontend.perPort[1].tls.validation.caCertificateRefs[1]") {
		t.Fatalf("expected cross-namespace per-port frontend CA certificate ref to require ReferenceGrant with indexed field context, got %v", err)
	}
}

func TestSpecToHostRequiresReferenceGrantForCrossNamespacePerPortFrontendCACertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	caNamespace := gatewayv1.Namespace("security")
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-ca"}},
		nil,
	)
	gateway := gatewayWithPerPortFrontendCACertificateRefs(443, gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca", Namespace: &caNamespace})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "ReferenceGrant") || !strings.Contains(err.Error(), "tls.frontend.perPort[0].tls.validation.caCertificateRefs[0]") {
		t.Fatalf("expected cross-namespace per-port frontend CA certificate ref to require ReferenceGrant with field context, got %v", err)
	}
}

func TestSpecToHostTranslatesGrantedBackendTLSClientCertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	certNamespace := gatewayv1.Namespace("security")
	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "allow-gateway-client-cert"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("Gateway"), Namespace: gatewayv1.Namespace("team-a")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Secret"), Name: ptr.To(gatewayv1.ObjectName("client-cert"))}},
		},
	}
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		nil,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-cert"}},
		grant,
	)
	expected := utiltranslate.Default.HostName(syncCtx, "client-cert", "security")
	gateway := gatewayWithBackendTLSClientCertificateRef(gatewayv1.SecretObjectReference{Name: "client-cert", Namespace: &certNamespace})

	spec, err := specToHost(syncCtx, gateway, true)
	if err != nil {
		t.Fatalf("translate Gateway spec: %v", err)
	}
	got := spec.TLS.Backend.ClientCertificateRef
	if got == nil {
		t.Fatalf("expected backend TLS client certificate ref")
	}
	if got.Name != gatewayv1.ObjectName(expected.Name) || got.Namespace == nil || *got.Namespace != gatewayv1.Namespace(expected.Namespace) {
		t.Fatalf("expected backend TLS client certificate ref %s/%s, got %#v", expected.Namespace, expected.Name, got)
	}
	if gateway.Spec.TLS.Backend.ClientCertificateRef.Name != "client-cert" {
		t.Fatalf("expected virtual Gateway to stay unchanged, got %q", gateway.Spec.TLS.Backend.ClientCertificateRef.Name)
	}
}

func TestSpecToHostRequiresReferenceGrantForCrossNamespaceBackendTLSClientCertificateRef(t *testing.T) {
	restore := setDefaultGatewayFrontendTLSTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	certNamespace := gatewayv1.Namespace("security")
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t,
		nil,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "security", Name: "client-cert"}},
	)
	gateway := gatewayWithBackendTLSClientCertificateRef(gatewayv1.SecretObjectReference{Name: "client-cert", Namespace: &certNamespace})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "ReferenceGrant") || !strings.Contains(err.Error(), "tls.backend.clientCertificateRef") {
		t.Fatalf("expected cross-namespace backend TLS client certificate ref to require ReferenceGrant with field context, got %v", err)
	}
}

func TestSpecToHostRequiresManagedHostBackendTLSClientCertificateRef(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithBackendTLSClientCertificateRef(gatewayv1.SecretObjectReference{Name: "client-cert"})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") || !strings.Contains(err.Error(), "tls.backend.clientCertificateRef") {
		t.Fatalf("expected missing host Secret to reject backend TLS client certificate ref with field context, got %v", err)
	}
}

func TestSpecToHostRequiresManagedHostDefaultFrontendCACertificateRef(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithDefaultFrontendCACertificateRefs(gatewayv1.ObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "client-ca"})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") {
		t.Fatalf("expected missing host ConfigMap to reject Gateway, got %v", err)
	}
}

func TestSpecToHostRejectsUnsupportedDefaultFrontendCACertificateRef(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithDefaultFrontendCACertificateRefs(gatewayv1.ObjectReference{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: "GatewayClass", Name: "client-ca"})

	_, err := specToHost(syncCtx, gateway, true)
	if !routetranslate.IsUnsupportedReference(err) {
		t.Fatalf("expected unsupported default frontend CA certificate ref to be terminal, got %v", err)
	}
}

func newGatewayFrontendTLSTranslateSyncContext(t *testing.T, configMap *corev1.ConfigMap, secret *corev1.Secret, virtualObjects ...runtime.Object) *synccontext.SyncContext {
	t.Helper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	seedCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("gateway-frontend-tls-translate-test")

	var hostObjects []runtime.Object
	if configMap != nil {
		hostObjects = append(hostObjects, utiltranslate.HostMetadata(configMap, utiltranslate.Default.HostName(seedCtx, configMap.Name, configMap.Namespace)))
	}
	if secret != nil {
		hostObjects = append(hostObjects, utiltranslate.HostMetadata(secret, utiltranslate.Default.HostName(seedCtx, secret.Name, secret.Namespace)))
	}

	pClient := testingutil.NewFakeClient(scheme.Scheme, hostObjects...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, virtualObjects...)
	return syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gateway-frontend-tls-translate-test")
}

func setDefaultGatewayFrontendTLSTranslator(translator utiltranslate.Translator) func() {
	previous := utiltranslate.Default
	utiltranslate.Default = translator
	return func() { utiltranslate.Default = previous }
}

func gatewayWithDefaultFrontendCACertificateRefs(refs ...gatewayv1.ObjectReference) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "tenant-class",
			TLS: &gatewayv1.GatewayTLSConfig{
				Frontend: &gatewayv1.FrontendTLSConfig{
					Default: gatewayv1.TLSConfig{
						Validation: &gatewayv1.FrontendTLSValidation{CACertificateRefs: refs},
					},
				},
			},
		},
	}
}

func gatewayWithPerPortFrontendCACertificateRefs(port gatewayv1.PortNumber, refs ...gatewayv1.ObjectReference) *gatewayv1.Gateway {
	return gatewayWithPerPortFrontendTLSConfigs(gatewayv1.TLSPortConfig{
		Port: port,
		TLS: gatewayv1.TLSConfig{
			Validation: &gatewayv1.FrontendTLSValidation{CACertificateRefs: refs},
		},
	})
}

func gatewayWithPerPortFrontendTLSConfigs(perPort ...gatewayv1.TLSPortConfig) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "tenant-class",
			TLS: &gatewayv1.GatewayTLSConfig{
				Frontend: &gatewayv1.FrontendTLSConfig{
					Default: gatewayv1.TLSConfig{},
					PerPort: perPort,
				},
			},
		},
	}
}

func gatewayWithBackendTLSClientCertificateRef(ref gatewayv1.SecretObjectReference) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "tenant-class",
			TLS: &gatewayv1.GatewayTLSConfig{
				Backend: &gatewayv1.GatewayBackendTLS{
					ClientCertificateRef: &ref,
				},
			},
		},
	}
}
