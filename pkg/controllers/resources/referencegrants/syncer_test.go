package referencegrants

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	testGrantName        = "allow-team-routes"
	testGrantNamespace   = "shared-infra"
	testFromKind         = "HTTPRoute"
	testFromGroup        = "gateway.networking.k8s.io"
	testFromNamespace    = "team-a"
	testTargetServiceA   = "shared-svc"
	testTargetServiceB   = "another-svc"
	testUnsupportedGroup = "example.com"
)

func TestSyncToHostTranslatesFromAndTo(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
	}, []gatewayv1.ReferenceGrantTo{
		{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gatewayv1.ObjectName(testTargetServiceA))},
	})
	syncCtx, syncer := startReferenceGrantSyncer(t, []runtime.Object{
		managedHostService(testTargetServiceA, testGrantNamespace),
	}, []runtime.Object{vGrant.DeepCopy()})

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vGrant.DeepCopy()))
	assert.NilError(t, err)

	stored := &gatewayv1.ReferenceGrant{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
		Name:      hostName(testGrantName, testGrantNamespace),
		Namespace: hostNamespace(testGrantNamespace),
	}, stored)
	assert.NilError(t, err)

	// Single-namespace mode collapses every from[].namespace to the configured target namespace.
	assert.Equal(t, len(stored.Spec.From), 1)
	assert.Equal(t, string(stored.Spec.From[0].Namespace), hostNamespace(testFromNamespace))
	assert.Equal(t, len(stored.Spec.To), 1)
	assert.Assert(t, stored.Spec.To[0].Name != nil)
	assert.Equal(t, string(*stored.Spec.To[0].Name), hostName(testTargetServiceA, testGrantNamespace))
}

func TestSyncToHostMultiNamespaceMode(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
		{Group: testFromGroup, Kind: "TLSRoute", Namespace: "team-b"},
	}, []gatewayv1.ReferenceGrantTo{
		{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gatewayv1.ObjectName(testTargetServiceA))},
	})
	syncCtx, _ := startReferenceGrantSyncer(t, []runtime.Object{
		managedHostService(testTargetServiceA, testGrantNamespace),
	}, []runtime.Object{vGrant.DeepCopy()})

	// Install the stub translator AFTER startReferenceGrantSyncer — the fake
	// register context resets translate.Default, so the swap has to happen here.
	previousTranslator := translate.Default
	translate.Default = &prefixingNamespaceTranslator{Translator: previousTranslator, prefix: "p-"}
	t.Cleanup(func() { translate.Default = previousTranslator })

	hSpec, err := specToHost(syncCtx, vGrant.DeepCopy(), false)
	assert.NilError(t, err)
	assert.Equal(t, len(hSpec.From), 2)
	assert.Equal(t, string(hSpec.From[0].Namespace), "p-"+testFromNamespace)
	assert.Equal(t, string(hSpec.From[1].Namespace), "p-team-b")
}

func TestSyncToHostRejectsUnsyncedTargetReference(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
	}, []gatewayv1.ReferenceGrantTo{
		{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gatewayv1.ObjectName(testTargetServiceA))},
	})
	syncCtx, syncer := startReferenceGrantSyncer(t, nil, []runtime.Object{vGrant.DeepCopy()})

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vGrant.DeepCopy()))
	assert.ErrorContains(t, err, `referenced Service "shared-svc"`)

	// And nothing was written.
	stored := &gatewayv1.ReferenceGrant{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testGrantName, testGrantNamespace), Namespace: hostNamespace(testGrantNamespace)}, stored)
	assert.Assert(t, kerrors.IsNotFound(err))
}

func TestSyncSkipsValidationOnUpdate(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
	}, []gatewayv1.ReferenceGrantTo{
		// Reference an unsynced Service. Sync should still succeed because Update path passes validateRefs=false.
		{Group: corev1.GroupName, Kind: "Service", Name: ptr.To(gatewayv1.ObjectName(testTargetServiceB))},
	})
	pGrant := referenceGrant(hostName(testGrantName, testGrantNamespace), hostNamespace(testGrantNamespace), nil, nil, withHostMeta())
	syncCtx, syncer := startReferenceGrantSyncer(t, []runtime.Object{pGrant.DeepCopy()}, []runtime.Object{vGrant.DeepCopy()})

	pGrant.ResourceVersion = "999"
	vGrant.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pGrant.DeepCopy(), pGrant.DeepCopy(), vGrant.DeepCopy(), vGrant.DeepCopy()))
	assert.NilError(t, err)
}

func TestSyncToHostSkipsUnsupportedToKind(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
	}, []gatewayv1.ReferenceGrantTo{
		{Group: testUnsupportedGroup, Kind: "CustomThing", Name: ptr.To(gatewayv1.ObjectName("anything"))},
	})
	syncCtx, syncer := startReferenceGrantSyncer(t, nil, []runtime.Object{vGrant.DeepCopy()})

	// Unsupported `to` kinds are terminal: the grant is not synced to the host and is
	// not requeued.
	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vGrant.DeepCopy()))
	assert.NilError(t, err)

	stored := &gatewayv1.ReferenceGrant{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
		Name:      hostName(testGrantName, testGrantNamespace),
		Namespace: hostNamespace(testGrantNamespace),
	}, stored)
	assert.Assert(t, kerrors.IsNotFound(err))
}

func TestSyncToHostNilToNameTranslatesNothing(t *testing.T) {
	vGrant := referenceGrant(testGrantName, testGrantNamespace, []gatewayv1.ReferenceGrantFrom{
		{Group: testFromGroup, Kind: testFromKind, Namespace: testFromNamespace},
	}, []gatewayv1.ReferenceGrantTo{
		// Name unset — covers all objects of Kind in the target namespace.
		{Group: corev1.GroupName, Kind: "Service"},
	})
	syncCtx, syncer := startReferenceGrantSyncer(t, nil, []runtime.Object{vGrant.DeepCopy()})

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vGrant.DeepCopy()))
	assert.NilError(t, err)

	stored := &gatewayv1.ReferenceGrant{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
		Name:      hostName(testGrantName, testGrantNamespace),
		Namespace: hostNamespace(testGrantNamespace),
	}, stored)
	assert.NilError(t, err)
	assert.Assert(t, stored.Spec.To[0].Name == nil)
}

// prefixingNamespaceTranslator wraps the ambient single-namespace translator
// but rewrites HostNamespace to a per-virtual prefix so we can assert that
// from[].namespace translation is mode-aware.
type prefixingNamespaceTranslator struct {
	translate.Translator
	prefix string
}

func (p *prefixingNamespaceTranslator) HostNamespace(_ *synccontext.SyncContext, vNamespace string) string {
	if vNamespace == "" {
		return ""
	}
	return p.prefix + vNamespace
}

func newReferenceGrantRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	vConfig.Sync.ToHost.GatewayAPI.Enabled = true
	return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
}

func startReferenceGrantSyncer(t *testing.T, initialPhysicalState, initialVirtualState []runtime.Object) (*synccontext.SyncContext, *referenceGrantSyncer) {
	t.Helper()

	pClient := testingutil.NewFakeClient(scheme.Scheme, initialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, initialVirtualState...)
	vConfig := testingutil.NewFakeConfig()
	registerContext := newReferenceGrantRegisterContext(vConfig, pClient, vClient)
	syncCtx, syncer := startSyncer(t, registerContext)
	return syncCtx, syncer
}

func startSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *referenceGrantSyncer) {
	t.Helper()
	syncCtx, raw := syncertesting.FakeStartSyncer(t, ctx, NewSyncer)
	return syncCtx, mustReferenceGrantSyncer(t, raw)
}

func mustReferenceGrantSyncer(t *testing.T, raw syncertypes.Object) *referenceGrantSyncer {
	t.Helper()
	s, ok := raw.(*referenceGrantSyncer)
	if !ok {
		t.Fatalf("expected *referenceGrantSyncer, got %T", raw)
	}
	return s
}

type grantOption func(*gatewayv1.ReferenceGrant)

func withHostMeta() grantOption {
	return func(g *gatewayv1.ReferenceGrant) {
		if g.Annotations == nil {
			g.Annotations = map[string]string{}
		}
		g.Annotations[translate.NameAnnotation] = testGrantName
		g.Annotations[translate.NamespaceAnnotation] = testGrantNamespace
		g.Annotations[translate.UIDAnnotation] = ""
		g.Annotations[translate.KindAnnotation] = mappings.ReferenceGrants().String()
		g.Annotations[translate.HostNamespaceAnnotation] = g.Namespace
		g.Annotations[translate.HostNameAnnotation] = g.Name

		if g.Labels == nil {
			g.Labels = map[string]string{}
		}
		g.Labels[translate.MarkerLabel] = translate.VClusterName
		g.Labels[translate.NamespaceLabel] = testGrantNamespace
	}
}

func referenceGrant(name, namespace string, from []gatewayv1.ReferenceGrantFrom, to []gatewayv1.ReferenceGrantTo, opts ...grantOption) *gatewayv1.ReferenceGrant {
	ret := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: from,
			To:   to,
		},
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

func managedHostService(name, namespace string) *corev1.Service {
	return translate.HostMetadata(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}, types.NamespacedName{
		Name:      hostName(name, namespace),
		Namespace: hostNamespace(namespace),
	})
}

func hostName(name, namespace string) string {
	return translate.SingleNamespaceHostName(name, namespace, translate.VClusterName)
}

func hostNamespace(namespace string) string {
	if namespace == "" {
		return ""
	}
	return testingutil.DefaultTestTargetNamespace
}
