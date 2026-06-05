package translate

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestParentRefToHostTranslatesImportedGatewayAndValidatesHostObject(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}

	hostGateway := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "networking", Name: "shared-edge"}}
	pClient := testingutil.NewFakeClient(scheme.Scheme, hostGateway)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	ref := gatewayv1.ParentReference{Name: "edge", Namespace: ptr.To(gatewayv1.Namespace("tenant-gateways"))}
	if err := ParentRefToHost(syncCtx, "routes", &ref); err != nil {
		t.Fatalf("expected imported Gateway parentRef to translate: %v", err)
	}
	if ref.Name != "shared-edge" || ref.Namespace == nil || *ref.Namespace != "networking" {
		t.Fatalf("expected parentRef to translate to networking/shared-edge, got namespace=%v name=%q", ref.Namespace, ref.Name)
	}
}

func TestParentRefToVirtualPreservesExplicitNamespaceFromSpecParentRef(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}

	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("gatewayroute-translate-test")
	ref := gatewayv1.ParentReference{Name: "shared-edge", Namespace: ptr.To(gatewayv1.Namespace("networking"))}
	specParentRefs := []gatewayv1.ParentReference{{Name: "shared-edge", Namespace: ptr.To(gatewayv1.Namespace("networking"))}}

	if err := ParentRefToVirtual(syncCtx, "host-routes", "routes", &ref, specParentRefs); err != nil {
		t.Fatalf("expected host parentRef status to translate: %v", err)
	}
	if ref.Name != "edge" || ref.Namespace == nil || *ref.Namespace != "tenant-gateways" {
		t.Fatalf("expected explicit namespace to be preserved as tenant-gateways/edge, got namespace=%v name=%q", ref.Namespace, ref.Name)
	}
}

func TestParentRefToVirtualOmitsNamespaceWhenImplicitAndRouteNamespaceMatches(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Enabled = true
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "routes/edge"}

	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("gatewayroute-translate-test")
	ref := gatewayv1.ParentReference{Name: "shared-edge"}

	if err := ParentRefToVirtual(syncCtx, "networking", "routes", &ref, nil); err != nil {
		t.Fatalf("expected host parentRef status to translate: %v", err)
	}
	if ref.Name != "edge" || ref.Namespace != nil {
		t.Fatalf("expected implicit same-namespace parentRef to omit namespace, got namespace=%v name=%q", ref.Namespace, ref.Name)
	}
}

func TestBackendObjectRefToHostRequiresManagedHostObject(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	ref := gatewayv1.BackendObjectReference{Name: "api"}
	err := BackendObjectRefToHost(syncCtx, "team-a", &ref)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") {
		t.Fatalf("expected missing host Service to be rejected, got %v", err)
	}

	hostName := utiltranslate.Default.HostName(syncCtx, "api", "team-a")
	unmanaged := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: hostName.Namespace, Name: hostName.Name}}
	if err := pClient.Create(context.Background(), unmanaged); err != nil {
		t.Fatalf("create unmanaged host Service: %v", err)
	}
	ref = gatewayv1.BackendObjectReference{Name: "api"}
	err = BackendObjectRefToHost(syncCtx, "team-a", &ref)
	if err == nil || !strings.Contains(err.Error(), "is not managed by vCluster") {
		t.Fatalf("expected unmanaged host Service to be rejected, got %v", err)
	}
}

func TestBackendObjectRefToHostTranslatesManagedServiceReference(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	hostName := utiltranslate.Default.HostName(syncCtx, "api", "team-a")
	virtualService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "api"}}
	managedHostService := utiltranslate.HostMetadata(virtualService, hostName)
	if err := pClient.Create(context.Background(), managedHostService); err != nil {
		t.Fatalf("create managed host Service: %v", err)
	}

	ref := gatewayv1.BackendObjectReference{Name: "api"}
	if err := BackendObjectRefToHost(syncCtx, "team-a", &ref); err != nil {
		t.Fatalf("expected managed Service reference to translate: %v", err)
	}
	if ref.Name != gatewayv1.ObjectName(hostName.Name) || ref.Namespace != nil {
		t.Fatalf("expected backendRef name to translate and implicit namespace to remain implicit, got namespace=%v name=%q", ref.Namespace, ref.Name)
	}
}

func TestReferenceGrantToHostTranslatesNamedServiceTarget(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	hostName := utiltranslate.Default.HostName(syncCtx, "api", "team-a")
	virtualService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "api"}}
	managedHostService := utiltranslate.HostMetadata(virtualService, hostName)
	if err := pClient.Create(context.Background(), managedHostService); err != nil {
		t.Fatalf("create managed host Service: %v", err)
	}

	ref := gatewayv1.ReferenceGrantTo{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Service"), Name: ptr.To(gatewayv1.ObjectName("api"))}
	if err := ReferenceGrantToHost(syncCtx, "team-a", &ref); err != nil {
		t.Fatalf("expected ReferenceGrant target to translate: %v", err)
	}
	if ref.Name == nil || *ref.Name != gatewayv1.ObjectName(hostName.Name) {
		t.Fatalf("expected ReferenceGrant target name %q, got %v", hostName.Name, ref.Name)
	}
}

func TestUnsupportedReferencesAreClassifiedAsTerminalTranslationErrors(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("gatewayroute-translate-test")

	ref := gatewayv1.BackendObjectReference{Group: ptr.To(gatewayv1.Group("example.com")), Kind: ptr.To(gatewayv1.Kind("Backend")), Name: "api"}
	err := BackendObjectRefToHost(syncCtx, "team-a", &ref)
	if !IsUnsupportedReference(err) {
		t.Fatalf("expected unsupported backendRef to be classified as unsupported, got %v", err)
	}
}

func TestSecretObjectRefToHostTranslatesExplicitNamespace(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	hostName := utiltranslate.Default.HostName(syncCtx, "cert", "certs")
	virtualSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "certs", Name: "cert"}}
	managedHostSecret := utiltranslate.HostMetadata(virtualSecret, hostName)
	if err := pClient.Create(context.Background(), managedHostSecret); err != nil {
		t.Fatalf("create managed host Secret: %v", err)
	}

	refNamespace := gatewayv1.Namespace("certs")
	ref := gatewayv1.SecretObjectReference{Name: "cert", Namespace: &refNamespace}
	if err := SecretObjectRefToHost(syncCtx, "team-a", &ref); err != nil {
		t.Fatalf("expected explicit Secret reference to translate: %v", err)
	}
	if ref.Name != gatewayv1.ObjectName(hostName.Name) || ref.Namespace == nil || *ref.Namespace != gatewayv1.Namespace(hostName.Namespace) {
		t.Fatalf("expected Secret ref to translate to %s/%s, got namespace=%v name=%q", hostName.Namespace, hostName.Name, ref.Namespace, ref.Name)
	}
}

func TestPolicyTargetRefToHostTranslatesLocalServiceReference(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	syncCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("gatewayroute-translate-test")

	hostName := utiltranslate.Default.HostName(syncCtx, "api", "team-a")
	virtualService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "api"}}
	managedHostService := utiltranslate.HostMetadata(virtualService, hostName)
	if err := pClient.Create(context.Background(), managedHostService); err != nil {
		t.Fatalf("create managed host Service: %v", err)
	}

	ref := gatewayv1.LocalPolicyTargetReferenceWithSectionName{LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Service"), Name: "api"}}
	if err := PolicyTargetRefToHost(syncCtx, "team-a", &ref); err != nil {
		t.Fatalf("expected policy targetRef to translate: %v", err)
	}
	if ref.Name != gatewayv1.ObjectName(hostName.Name) {
		t.Fatalf("expected policy targetRef name %q, got %q", hostName.Name, ref.Name)
	}
}

func TestParentStatusHostNamespaceUsesRouteNamespaceForImplicitRefs(t *testing.T) {
	if got := ParentStatusHostNamespace("route-ns", nil, gatewayv1.ParentReference{Name: "edge"}); got != "route-ns" {
		t.Fatalf("expected implicit status parent namespace to default to route namespace, got %q", got)
	}
}
