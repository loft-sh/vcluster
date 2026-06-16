package gateways

import (
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestListenersToHostTranslatesInfrastructureParametersRef(t *testing.T) {
	tests := []struct {
		name      string
		ref       gatewayv1.LocalParametersReference
		configMap *corev1.ConfigMap
		secret    *corev1.Secret
	}{
		{
			name:      "configmap",
			ref:       gatewayv1.LocalParametersReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "params"},
			configMap: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "params"}},
		},
		{
			name:   "secret",
			ref:    gatewayv1.LocalParametersReference{Group: corev1.GroupName, Kind: "Secret", Name: "params"},
			secret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "params"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncCtx := newGatewayTranslateSyncContext(t, tt.configMap, tt.secret)
			expected := utiltranslate.Default.HostName(syncCtx, "params", "team-a")
			gateway := gatewayWithParametersRef(tt.ref)

			spec, err := listenersToHost(syncCtx, gateway, true)
			if err != nil {
				t.Fatalf("translate Gateway spec: %v", err)
			}
			if spec.Infrastructure == nil || spec.Infrastructure.ParametersRef == nil {
				t.Fatalf("expected infrastructure.parametersRef to be preserved")
			}
			if spec.Infrastructure.ParametersRef.Name != expected.Name {
				t.Fatalf("expected parametersRef name %q, got %q", expected.Name, spec.Infrastructure.ParametersRef.Name)
			}
			if gateway.Spec.Infrastructure.ParametersRef.Name != "params" {
				t.Fatalf("expected virtual Gateway to stay unchanged, got %q", gateway.Spec.Infrastructure.ParametersRef.Name)
			}
		})
	}
}

func TestListenersToHostLeavesUnsupportedInfrastructureParametersRefUnchanged(t *testing.T) {
	syncCtx := newGatewayTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: "example.com", Kind: "GatewayConfig", Name: "params"})

	spec, err := listenersToHost(syncCtx, gateway, true)
	if err != nil {
		t.Fatalf("expected unsupported infrastructure.parametersRef to be skipped, got %v", err)
	}
	if spec.Infrastructure.ParametersRef.Name != "params" {
		t.Fatalf("expected unsupported parametersRef name to stay verbatim, got %q", spec.Infrastructure.ParametersRef.Name)
	}
}

func TestListenersToHostRequiresManagedHostInfrastructureParametersRef(t *testing.T) {
	syncCtx := newGatewayTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "params"})

	_, err := listenersToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") {
		t.Fatalf("expected missing host ConfigMap to reject Gateway, got %v", err)
	}
}

func newGatewayTranslateSyncContext(t *testing.T, configMap *corev1.ConfigMap, secret *corev1.Secret) *synccontext.SyncContext {
	t.Helper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	seedCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("gateway-translate-test")

	var hostObjects []runtime.Object
	if configMap != nil {
		hostObjects = append(hostObjects, utiltranslate.HostMetadata(configMap, utiltranslate.Default.HostName(seedCtx, configMap.Name, configMap.Namespace)))
	}
	if secret != nil {
		hostObjects = append(hostObjects, utiltranslate.HostMetadata(secret, utiltranslate.Default.HostName(seedCtx, secret.Name, secret.Namespace)))
	}

	pClient := testingutil.NewFakeClient(scheme.Scheme, hostObjects...)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	return syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gateway-translate-test")
}

func gatewayWithParametersRef(ref gatewayv1.LocalParametersReference) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "tenant-class",
			Infrastructure: &gatewayv1.GatewayInfrastructure{
				ParametersRef: &ref,
			},
		},
	}
}
