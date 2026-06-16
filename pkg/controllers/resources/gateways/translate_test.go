package gateways

import (
	"strings"
	"testing"

	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestSpecToHostTranslatesInfrastructureParametersRef(t *testing.T) {
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
			syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, tt.configMap, tt.secret)
			expected := utiltranslate.Default.HostName(syncCtx, "params", "team-a")
			gateway := gatewayWithParametersRef(tt.ref)

			spec, err := specToHost(syncCtx, gateway, true)
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

func TestSpecToHostRejectsUnsupportedInfrastructureParametersRef(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: "example.com", Kind: "GatewayConfig", Name: "params"})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "parametersRef group \"example.com\" kind \"GatewayConfig\" is not supported") {
		t.Fatalf("expected unsupported infrastructure.parametersRef to reject Gateway, got %v", err)
	}
}

func TestSpecToHostRequiresManagedHostInfrastructureParametersRef(t *testing.T) {
	syncCtx := newGatewayFrontendTLSTranslateSyncContext(t, nil, nil)
	gateway := gatewayWithParametersRef(gatewayv1.LocalParametersReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "params"})

	_, err := specToHost(syncCtx, gateway, true)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") {
		t.Fatalf("expected missing host ConfigMap to reject Gateway, got %v", err)
	}
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
