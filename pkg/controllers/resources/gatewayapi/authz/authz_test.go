package authz

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestReferenceGrantAllowsEmptyNameWildcard(t *testing.T) {
	emptyName := gatewayv1.ObjectName("")
	grant := gatewayv1.ReferenceGrant{
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{
				{
					Group:     gatewayv1.Group(gatewayv1.GroupVersion.Group),
					Kind:      gatewayv1.Kind("HTTPRoute"),
					Namespace: gatewayv1.Namespace("route-ns"),
				},
			},
			To: []gatewayv1.ReferenceGrantTo{
				{
					Group: gatewayv1.Group(corev1.GroupName),
					Kind:  gatewayv1.Kind("Service"),
					Name:  &emptyName,
				},
			},
		},
	}

	assert.Assert(t, referenceGrantAllows(grant, gatewayv1.GroupVersion.Group, "HTTPRoute", "route-ns", referenceTarget{
		group:     corev1.GroupName,
		kind:      "Service",
		namespace: "backend-ns",
		name:      "backend",
	}))
}

func TestHTTPRouteAttachmentEvaluatesAllowedRoutesInMultiNamespaceMode(t *testing.T) {
	restoreTranslator(t, testTranslator{singleNamespace: false})

	namespace := gatewayv1.Namespace("gateway-ns")
	ref := &gatewayv1.ParentReference{
		Name:      gatewayv1.ObjectName("gateway"),
		Namespace: &namespace,
	}

	for _, tc := range []struct {
		name        string
		labels      map[string]string
		expectAllow bool
	}{
		{
			name:        "selector matches",
			labels:      map[string]string{"team": "blue"},
			expectAllow: true,
		},
		{
			name:        "selector does not match",
			labels:      map[string]string{"team": "red"},
			expectAllow: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := authzSyncContext(
				virtualGatewayWithAllowedRoutesSelector(),
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "route-ns", Labels: tc.labels}},
			)

			err := HTTPRouteAttachment(ctx, "route-ns", ref)
			if tc.expectAllow {
				assert.NilError(t, err)
			} else {
				assert.Assert(t, IsNotPermitted(err), "expected not permitted error, got %v", err)
			}
		})
	}
}

func TestHTTPRouteAttachmentAllowsCrossNamespaceServiceParentWithoutReferenceGrant(t *testing.T) {
	restoreTranslator(t, testTranslator{singleNamespace: false})

	group := gatewayv1.Group(corev1.GroupName)
	kind := gatewayv1.Kind("Service")
	namespace := gatewayv1.Namespace("service-ns")
	ref := &gatewayv1.ParentReference{
		Group:     &group,
		Kind:      &kind,
		Name:      gatewayv1.ObjectName("backend"),
		Namespace: &namespace,
	}

	err := HTTPRouteAttachment(authzSyncContext(), "route-ns", ref)
	assert.NilError(t, err)
}

func authzSyncContext(objects ...runtime.Object) *synccontext.SyncContext {
	runtimeObjects := make([]runtime.Object, 0, len(objects))
	runtimeObjects = append(runtimeObjects, objects...)

	return &synccontext.SyncContext{
		Context:       context.Background(),
		VirtualClient: testingutil.NewFakeClient(scheme.Scheme, runtimeObjects...),
	}
}

func virtualGatewayWithAllowedRoutesSelector() *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gateway",
			Namespace: "gateway-ns",
		},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     gatewayv1.SectionName("http"),
					Port:     gatewayv1.PortNumber(80),
					Protocol: gatewayv1.HTTPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr.To(gatewayv1.NamespacesFromSelector),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"team": "blue"},
							},
						},
					},
				},
			},
		},
	}
}

func restoreTranslator(t *testing.T, translator utiltranslate.Translator) {
	t.Helper()

	original := utiltranslate.Default
	utiltranslate.Default = translator
	t.Cleanup(func() {
		utiltranslate.Default = original
	})
}

type testTranslator struct {
	singleNamespace bool
}

func (t testTranslator) SingleNamespaceTarget() bool {
	return t.singleNamespace
}

func (t testTranslator) IsManaged(_ *synccontext.SyncContext, _ client.Object) bool {
	return false
}

func (t testTranslator) IsTargetedNamespace(_ *synccontext.SyncContext, _ string) bool {
	return false
}

func (t testTranslator) MarkerLabelCluster() string {
	return ""
}

func (t testTranslator) HostName(_ *synccontext.SyncContext, _, _ string) types.NamespacedName {
	return types.NamespacedName{}
}

func (t testTranslator) HostNameShort(_ *synccontext.SyncContext, _, _ string) types.NamespacedName {
	return types.NamespacedName{}
}

func (t testTranslator) HostNameCluster(_ string) string {
	return ""
}

func (t testTranslator) HostNamespace(_ *synccontext.SyncContext, namespace string) string {
	return namespace
}

func (t testTranslator) LabelsToTranslate() map[string]bool {
	return nil
}
