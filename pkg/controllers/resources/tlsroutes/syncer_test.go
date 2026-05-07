package tlsroutes

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	testRouteName        = "testroute"
	testRouteNamespace   = "test"
	testParentNamespace  = "gateway-ns"
	testGatewayName      = "testgateway"
	testServiceName      = "testservice"
	testControllerName   = gatewayv1.GatewayController("example.com/gateway-controller")
	testUnsupportedGroup = gatewayv1.Group("example.com")
)

func TestSync(t *testing.T) {
	vBaseSpec := routeSpec()
	pBaseSpec := hostRouteSpec()
	hostStatus := hostRouteStatus()
	virtualStatus := virtualRouteStatus()
	vObjectMeta := virtualRouteMeta()
	pObjectMeta := hostRouteMeta()
	baseRoute := tlsRoute(vObjectMeta, vBaseSpec)
	createdRoute := tlsRoute(pObjectMeta, pBaseSpec)
	hostRouteWithStatus := tlsRoute(pObjectMeta, gatewayv1.TLSRouteSpec{}, withStatus(hostStatus))
	expectedHostRouteWithStatus := tlsRoute(pObjectMeta, pBaseSpec, withStatus(hostStatus))
	expectedVirtualRouteWithStatus := tlsRoute(vObjectMeta, vBaseSpec, withStatus(virtualStatus))

	syncertesting.RunTestsWithContext(t, newTLSRouteRegisterContext, []*syncertesting.SyncTest{
		{
			Name:                 "Create forward",
			InitialVirtualState:  []runtime.Object{baseRoute.DeepCopy()},
			InitialPhysicalState: hostRefObjects(testRouteNamespace),
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.TLSRoutes(): {baseRoute.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.TLSRoutes(): {createdRoute.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*tlsRouteSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseRoute.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward and status back",
			InitialVirtualState:  []runtime.Object{baseRoute.DeepCopy(), virtualGateway()},
			InitialPhysicalState: append([]runtime.Object{hostRouteWithStatus.DeepCopy()}, hostRefObjects(testRouteNamespace)...),
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.TLSRoutes(): {expectedVirtualRouteWithStatus.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.TLSRoutes(): {expectedHostRouteWithStatus.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				pRoute := hostRouteWithStatus.DeepCopy()
				pRoute.ResourceVersion = "999"
				vRoute := baseRoute.DeepCopy()
				vRoute.ResourceVersion = "999"

				_, err := syncer.(*tlsRouteSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute, pRoute, vRoute, vRoute))
				assert.NilError(t, err)
			},
		},
	})
}

func TestSyncRejectsUnsyncedParentGateway(t *testing.T) {
	vRoute := tlsRoute(virtualRouteMeta(), routeSpec())
	syncCtx, syncer := startTLSRouteSyncer(t, hostServiceObjects(testRouteNamespace), []runtime.Object{vRoute}, nil)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.ErrorContains(t, err, `referenced Gateway "testgateway" in namespace "test" has no synced host object`)

	storedHostRoute := &gatewayv1.TLSRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.Assert(t, apierrors.IsNotFound(err))
}

func TestSyncContinuesWhenStatusTranslationFails(t *testing.T) {
	vRoute := tlsRoute(virtualRouteMeta(), routeSpec())
	pRoute := tlsRoute(hostRouteMeta(), gatewayv1.TLSRouteSpec{}, withStatus(gatewayv1.TLSRouteStatus{
		RouteStatus: gatewayv1.RouteStatus{
			Parents: []gatewayv1.RouteParentStatus{
				{
					ParentRef:      gatewayv1.ParentReference{Name: gatewayv1.ObjectName(hostName("missing-gateway"))},
					ControllerName: testControllerName,
				},
			},
		},
	}))
	syncCtx, syncer := startTLSRouteSyncer(
		t,
		append([]runtime.Object{pRoute.DeepCopy()}, hostRefObjects(testRouteNamespace)...),
		[]runtime.Object{vRoute.DeepCopy()},
		nil,
	)

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	assert.ErrorContains(t, err, `failed to translate status`)

	storedHostRoute := &gatewayv1.TLSRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: pRoute.Name, Namespace: pRoute.Namespace}, storedHostRoute)
	assert.NilError(t, err)
	assert.DeepEqual(t, storedHostRoute.Spec, hostRouteSpec())
}

func TestSyncCrossNamespaceParentRef(t *testing.T) {
	vRoute := tlsRoute(virtualRouteMeta(), routeSpecWithParentNamespace(testParentNamespace))
	pRoute := tlsRoute(hostRouteMeta(), hostRouteSpecWithParentNamespace(testParentNamespace), withStatus(hostRouteStatusForNamespace(testParentNamespace, false)))
	syncCtx, syncer := startTLSRouteSyncer(
		t,
		append([]runtime.Object{pRoute.DeepCopy()}, hostRefObjects(testRouteNamespace, testParentNamespace)...),
		[]runtime.Object{vRoute.DeepCopy(), virtualGatewayWithNamespace(testParentNamespace)},
		nil,
	)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.TLSRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.NilError(t, err)
	assert.Equal(t, string(storedHostRoute.Spec.ParentRefs[0].Name), hostNameForNamespace(testGatewayName, testParentNamespace))
	assert.Assert(t, storedHostRoute.Spec.ParentRefs[0].Namespace != nil)
	assert.Equal(t, string(*storedHostRoute.Spec.ParentRefs[0].Namespace), hostNamespace(testParentNamespace))

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err = syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedVirtualRoute := &gatewayv1.TLSRoute{}
	err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, storedVirtualRoute)
	assert.NilError(t, err)
	assert.Equal(t, string(storedVirtualRoute.Status.Parents[0].ParentRef.Name), testGatewayName)
	assert.Assert(t, storedVirtualRoute.Status.Parents[0].ParentRef.Namespace != nil)
	assert.Equal(t, string(*storedVirtualRoute.Status.Parents[0].ParentRef.Namespace), testParentNamespace)
}

func TestSyncRejectsUnsupportedRefs(t *testing.T) {
	tests := []struct {
		name        string
		route       *gatewayv1.TLSRoute
		expectedErr string
	}{
		{
			name: "Unsupported parentRef",
			route: tlsRoute(virtualRouteMeta(), gatewayv1.TLSRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{
							Group: ptr.To(testUnsupportedGroup),
							Kind:  ptr.To(gatewayv1.Kind("ExampleGateway")),
							Name:  gatewayv1.ObjectName(testGatewayName),
						},
					},
				},
			}),
			expectedErr: `parentRef group "example.com" kind "ExampleGateway" is not supported`,
		},
		{
			name: "Unsupported backendRef",
			route: tlsRoute(virtualRouteMeta(), gatewayv1.TLSRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(testGatewayName)}},
				},
				Rules: []gatewayv1.TLSRouteRule{
					{
						BackendRefs: []gatewayv1.BackendRef{
							{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Group: ptr.To(testUnsupportedGroup),
									Kind:  ptr.To(gatewayv1.Kind("ExampleBackend")),
									Name:  gatewayv1.ObjectName(testServiceName),
								},
							},
						},
					},
				},
			}),
			expectedErr: `backendRef group "example.com" kind "ExampleBackend" is not supported`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			syncCtx, syncer := startTLSRouteSyncer(t, hostRefObjects(testRouteNamespace), []runtime.Object{tc.route}, nil)
			_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(tc.route.DeepCopy()))
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func newTLSRouteRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	vConfig.Sync.ToHost.Gateways.Enabled = true
	return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
}

func startTLSRouteSyncer(
	t *testing.T,
	initialPhysicalState []runtime.Object,
	initialVirtualState []runtime.Object,
	adjustConfig func(*config.VirtualClusterConfig),
) (*synccontext.SyncContext, *tlsRouteSyncer) {
	t.Helper()

	pClient := testingutil.NewFakeClient(scheme.Scheme, initialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, initialVirtualState...)
	vConfig := testingutil.NewFakeConfig()
	if adjustConfig != nil {
		adjustConfig(vConfig)
	}

	registerContext := newTLSRouteRegisterContext(vConfig, pClient, vClient)
	syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
	return syncCtx, syncer.(*tlsRouteSyncer)
}

type tlsRouteOption func(*gatewayv1.TLSRoute)

func withStatus(status gatewayv1.TLSRouteStatus) tlsRouteOption {
	return func(route *gatewayv1.TLSRoute) {
		route.Status = status
	}
}

func tlsRoute(meta metav1.ObjectMeta, spec gatewayv1.TLSRouteSpec, opts ...tlsRouteOption) *gatewayv1.TLSRoute {
	ret := &gatewayv1.TLSRoute{
		ObjectMeta: meta,
		Spec:       spec,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

func routeSpec() gatewayv1.TLSRouteSpec {
	return routeSpecWithParentNamespace("")
}

func routeSpecWithParentNamespace(parentNamespace string) gatewayv1.TLSRouteSpec {
	parentRef := gatewayv1.ParentReference{Name: gatewayv1.ObjectName(testGatewayName)}
	if parentNamespace != "" {
		parentRef.Namespace = ptr.To(gatewayv1.Namespace(parentNamespace))
	}

	return gatewayv1.TLSRouteSpec{
		CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{
				parentRef,
			},
		},
		Hostnames: []gatewayv1.Hostname{"example.com"},
		Rules: []gatewayv1.TLSRouteRule{
			{
				Name:        ptr.To(gatewayv1.SectionName("tls")),
				BackendRefs: []gatewayv1.BackendRef{serviceBackendRef(testServiceName)},
			},
		},
	}
}

func hostRouteSpec() gatewayv1.TLSRouteSpec {
	return hostRouteSpecWithParentNamespace("")
}

func hostRouteSpecWithParentNamespace(parentNamespace string) gatewayv1.TLSRouteSpec {
	spec := routeSpecWithParentNamespace(parentNamespace)
	ret := *spec.DeepCopy()
	ret.ParentRefs[0].Name = gatewayv1.ObjectName(hostNameForNamespace(testGatewayName, refNamespaceOrDefault(parentNamespace, testRouteNamespace)))
	if parentNamespace != "" {
		ret.ParentRefs[0].Namespace = ptr.To(gatewayv1.Namespace(hostNamespace(parentNamespace)))
	}
	ret.Rules[0].BackendRefs[0].Name = gatewayv1.ObjectName(hostName(testServiceName))
	return ret
}

func refNamespaceOrDefault(namespace, defaultNamespace string) string {
	if namespace == "" {
		return defaultNamespace
	}

	return namespace
}

func serviceBackendRef(name string) gatewayv1.BackendRef {
	return gatewayv1.BackendRef{
		BackendObjectReference: gatewayv1.BackendObjectReference{
			Name: gatewayv1.ObjectName(name),
			Port: ptr.To(gatewayv1.PortNumber(443)),
		},
		Weight: ptr.To[int32](2),
	}
}

func virtualRouteStatus() gatewayv1.TLSRouteStatus {
	status := hostRouteStatus()
	status.Parents[0].ParentRef.Name = gatewayv1.ObjectName(testGatewayName)
	status.Parents[0].ParentRef.Namespace = nil
	return status
}

func hostRouteStatus() gatewayv1.TLSRouteStatus {
	return hostRouteStatusForNamespace(testRouteNamespace, true)
}

func hostRouteStatusForNamespace(parentNamespace string, includeNamespace bool) gatewayv1.TLSRouteStatus {
	parentRef := gatewayv1.ParentReference{Name: gatewayv1.ObjectName(hostNameForNamespace(testGatewayName, parentNamespace))}
	if includeNamespace {
		parentRef.Namespace = ptr.To(gatewayv1.Namespace(hostNamespace(parentNamespace)))
	}

	return gatewayv1.TLSRouteStatus{
		RouteStatus: gatewayv1.RouteStatus{
			Parents: []gatewayv1.RouteParentStatus{
				{
					ParentRef:      parentRef,
					ControllerName: testControllerName,
					Conditions: []metav1.Condition{
						{
							Type:   string(gatewayv1.RouteConditionAccepted),
							Status: metav1.ConditionTrue,
							Reason: string(gatewayv1.RouteReasonAccepted),
						},
					},
				},
			},
		},
	}
}

func virtualRouteMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      testRouteName,
		Namespace: testRouteNamespace,
	}
}

func hostRouteMeta() metav1.ObjectMeta {
	hostRouteName := hostName(testRouteName)
	return metav1.ObjectMeta{
		Name:      hostRouteName,
		Namespace: testRouteNamespace,
		Annotations: map[string]string{
			translate.NameAnnotation:          testRouteName,
			translate.NamespaceAnnotation:     testRouteNamespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          mappings.TLSRoutes().String(),
			translate.HostNamespaceAnnotation: testRouteNamespace,
			translate.HostNameAnnotation:      hostRouteName,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: testRouteNamespace,
		},
		ResourceVersion: "999",
	}
}

func hostName(name string) string {
	return hostNameForNamespace(name, testRouteNamespace)
}

func hostNameForNamespace(name, namespace string) string {
	return translate.SingleNamespaceHostName(name, namespace, translate.VClusterName)
}

func hostNamespace(namespace string) string {
	if namespace == "" {
		return ""
	}

	return testingutil.DefaultTestTargetNamespace
}

func virtualGateway() *gatewayv1.Gateway {
	return virtualGatewayWithNamespace(testRouteNamespace)
}

func virtualGatewayWithNamespace(namespace string) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testGatewayName,
			Namespace: namespace,
		},
	}
}

func hostRefObjects(namespaces ...string) []runtime.Object {
	ret := hostServiceObjects(testRouteNamespace)
	for _, namespace := range namespaces {
		ret = append(ret, hostGateway(namespace))
	}
	return ret
}

func hostServiceObjects(namespace string) []runtime.Object {
	return []runtime.Object{
		hostService(testServiceName, namespace),
	}
}

func hostGateway(namespace string) *gatewayv1.Gateway {
	return translate.HostMetadata(virtualGatewayWithNamespace(namespace), types.NamespacedName{
		Name:      hostNameForNamespace(testGatewayName, namespace),
		Namespace: hostNamespace(namespace),
	})
}

func hostService(name, namespace string) *corev1.Service {
	return translate.HostMetadata(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, types.NamespacedName{
		Name:      hostNameForNamespace(name, namespace),
		Namespace: hostNamespace(namespace),
	})
}
