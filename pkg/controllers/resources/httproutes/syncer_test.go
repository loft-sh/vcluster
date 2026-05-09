package httproutes

import (
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	gatewayapitestutil "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/testutil"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	testBackendNamespace = "backend-ns"
	testGatewayName      = "testgateway"
	testServiceName      = "testservice"
	testMirrorService    = "mirrorservice"
	testAuthService      = "authservice"
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
	baseRoute := httpRoute(vObjectMeta, vBaseSpec)
	createdRoute := httpRoute(pObjectMeta, pBaseSpec)
	hostRouteWithStatus := httpRoute(pObjectMeta, gatewayv1.HTTPRouteSpec{}, withStatus(hostStatus))
	expectedHostRouteWithStatus := httpRoute(pObjectMeta, pBaseSpec, withStatus(hostStatus))
	expectedVirtualRouteWithStatus := httpRoute(vObjectMeta, vBaseSpec, withStatus(virtualStatus))

	syncertesting.RunTestsWithContext(t, newHTTPRouteRegisterContext, []*syncertesting.SyncTest{
		{
			Name:                 "Create forward",
			InitialVirtualState:  []runtime.Object{baseRoute.DeepCopy(), virtualGateway()},
			InitialPhysicalState: hostRefObjects(testRouteNamespace),
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.HTTPRoutes(): {baseRoute.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.HTTPRoutes(): {createdRoute.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*httpRouteSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseRoute.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward and status back",
			InitialVirtualState:  []runtime.Object{baseRoute.DeepCopy(), virtualGateway()},
			InitialPhysicalState: append([]runtime.Object{hostRouteWithStatus.DeepCopy()}, hostRefObjects(testRouteNamespace)...),
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.HTTPRoutes(): {expectedVirtualRouteWithStatus.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				mappings.HTTPRoutes(): {expectedHostRouteWithStatus.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				pRoute := hostRouteWithStatus.DeepCopy()
				pRoute.ResourceVersion = "999"
				vRoute := baseRoute.DeepCopy()
				vRoute.ResourceVersion = "999"

				_, err := syncer.(*httpRouteSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute, pRoute, vRoute, vRoute))
				assert.NilError(t, err)
			},
		},
	})
}

func TestSyncRejectsUnsyncedParentGateway(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpec())
	syncCtx, syncer := startHTTPRouteSyncer(t, hostServiceObjects(testRouteNamespace), []runtime.Object{vRoute, virtualGateway()}, nil)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.ErrorContains(t, err, `referenced Gateway "testgateway" in namespace "test" has no synced host object`)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.Assert(t, kerrors.IsNotFound(err))
}

func TestSyncSkipsReferenceValidationOnUpdate(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpec())
	pRoute := httpRoute(hostRouteMeta(), gatewayv1.HTTPRouteSpec{})
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		[]runtime.Object{pRoute.DeepCopy()},
		[]runtime.Object{vRoute.DeepCopy(), virtualGateway()},
		nil,
	)

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	assert.NilError(t, err)
}

func TestSyncAppliesSpecAndRequeuesWhenStatusTranslationFails(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpec())
	pRoute := httpRoute(hostRouteMeta(), gatewayv1.HTTPRouteSpec{}, withStatus(gatewayv1.HTTPRouteStatus{
		RouteStatus: gatewayv1.RouteStatus{
			Parents: []gatewayv1.RouteParentStatus{
				{
					ParentRef:      gatewayv1.ParentReference{Name: gatewayv1.ObjectName(hostName("missing-gateway"))},
					ControllerName: testControllerName,
				},
			},
		},
	}))
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		append([]runtime.Object{pRoute.DeepCopy()}, hostRefObjects(testRouteNamespace)...),
		[]runtime.Object{vRoute.DeepCopy(), virtualGateway()},
		nil,
	)

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	// Status translation failure is surfaced so the route is requeued and retried;
	// the spec is still applied to the host independently of status.
	assert.ErrorContains(t, err, "translate status:")

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: pRoute.Name, Namespace: pRoute.Namespace}, storedHostRoute)
	assert.NilError(t, err)
	assert.DeepEqual(t, storedHostRoute.Spec, hostRouteSpec())
}

func TestSyncCrossNamespaceParentRef(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpecWithParentNamespace(testParentNamespace))
	pRoute := httpRoute(hostRouteMeta(), hostRouteSpecWithParentNamespace(testParentNamespace), withStatus(hostRouteStatusForNamespace(testParentNamespace, false)))
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		append([]runtime.Object{pRoute.DeepCopy()}, hostRefObjects(testRouteNamespace, testParentNamespace)...),
		[]runtime.Object{vRoute.DeepCopy(), virtualGatewayWithNamespace(testParentNamespace, withAllowedRoutesAll())},
		nil,
	)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.NilError(t, err)
	assert.Equal(t, string(storedHostRoute.Spec.ParentRefs[0].Name), hostNameForNamespace(testGatewayName, testParentNamespace))
	assert.Assert(t, storedHostRoute.Spec.ParentRefs[0].Namespace != nil)
	assert.Equal(t, string(*storedHostRoute.Spec.ParentRefs[0].Namespace), hostNamespace(testParentNamespace))

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err = syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedVirtualRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, storedVirtualRoute)
	assert.NilError(t, err)
	assert.Equal(t, string(storedVirtualRoute.Status.Parents[0].ParentRef.Name), testGatewayName)
	assert.Assert(t, storedVirtualRoute.Status.Parents[0].ParentRef.Namespace != nil)
	assert.Equal(t, string(*storedVirtualRoute.Status.Parents[0].ParentRef.Namespace), testParentNamespace)
}

func TestSyncRejectsCrossNamespaceParentRefWithoutAllowedRoutes(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpecWithParentNamespace(testParentNamespace))
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		hostRefObjects(testRouteNamespace, testParentNamespace),
		[]runtime.Object{vRoute.DeepCopy(), virtualGatewayWithNamespace(testParentNamespace)},
		nil,
	)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.Assert(t, kerrors.IsNotFound(err))
}

func TestSyncCrossNamespaceParentRefWithAllowedRoutesSelector(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpecWithParentNamespace(testParentNamespace))
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		hostRefObjects(testRouteNamespace, testParentNamespace),
		[]runtime.Object{
			vRoute.DeepCopy(),
			virtualGatewayWithNamespace(testParentNamespace, withAllowedRoutesSelector(map[string]string{"team": "blue"})),
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testRouteNamespace, Labels: map[string]string{"team": "blue"}}},
		},
		nil,
	)

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.NilError(t, err)
}

func TestSyncCrossNamespaceBackendRefRequiresReferenceGrant(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpec())
	vRoute.Spec.Rules[0].BackendRefs[0].Namespace = ptr.To(gatewayv1.Namespace(testBackendNamespace))
	hostObjects := append(hostRefObjects(testRouteNamespace), hostService(testServiceName, testBackendNamespace))

	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		hostObjects,
		[]runtime.Object{vRoute.DeepCopy(), virtualGateway()},
		nil,
	)
	translator, recorder := gatewayapitestutil.WithFakeEventRecorder(syncer.GenericTranslator)
	syncer.GenericTranslator = translator

	_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.Assert(t, kerrors.IsNotFound(err))

	event, ok := gatewayapitestutil.NextEvent(recorder)
	assert.Assert(t, ok)
	assert.Assert(t, strings.Contains(event, "Warning RefNotPermitted"))
	assert.Assert(t, strings.Contains(event, `no matching virtual ReferenceGrant in namespace "backend-ns" permits HTTPRoute`))

	syncCtx, syncer = startHTTPRouteSyncer(
		t,
		hostObjects,
		[]runtime.Object{
			vRoute.DeepCopy(),
			virtualGateway(),
			referenceGrant(testBackendNamespace, "HTTPRoute", testRouteNamespace, "Service", testServiceName),
		},
		nil,
	)
	_, err = syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute = &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.NilError(t, err)
	assert.Equal(t, string(storedHostRoute.Spec.Rules[0].BackendRefs[0].Name), hostNameForNamespace(testServiceName, testBackendNamespace))
	assert.Assert(t, storedHostRoute.Spec.Rules[0].BackendRefs[0].Namespace != nil)
	assert.Equal(t, string(*storedHostRoute.Spec.Rules[0].BackendRefs[0].Namespace), hostNamespace(testBackendNamespace))
}

func TestSyncDeletesHostRouteWhenReferenceGrantRemoved(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), routeSpec())
	vRoute.Spec.Rules[0].BackendRefs[0].Namespace = ptr.To(gatewayv1.Namespace(testBackendNamespace))
	pRoute := httpRoute(hostRouteMeta(), hostRouteSpec())
	syncCtx, syncer := startHTTPRouteSyncer(
		t,
		[]runtime.Object{
			pRoute.DeepCopy(),
			hostGateway(testRouteNamespace),
			hostService(testServiceName, testBackendNamespace),
			hostService(testMirrorService, testRouteNamespace),
			hostService(testAuthService, testRouteNamespace),
		},
		[]runtime.Object{vRoute.DeepCopy(), virtualGateway()},
		nil,
	)

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
	assert.Assert(t, kerrors.IsNotFound(err))
}

func TestSyncRejectsUnsupportedRefs(t *testing.T) {
	tests := []struct {
		name        string
		route       *gatewayv1.HTTPRoute
		expectedErr string
	}{
		{
			name: "Unsupported parentRef",
			route: httpRoute(virtualRouteMeta(), gatewayv1.HTTPRouteSpec{
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
			route: httpRoute(virtualRouteMeta(), gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(testGatewayName)}},
				},
				Rules: []gatewayv1.HTTPRouteRule{
					{
						BackendRefs: []gatewayv1.HTTPBackendRef{
							{
								BackendRef: gatewayv1.BackendRef{
									BackendObjectReference: gatewayv1.BackendObjectReference{
										Group: ptr.To(testUnsupportedGroup),
										Kind:  ptr.To(gatewayv1.Kind("ExampleBackend")),
										Name:  gatewayv1.ObjectName(testServiceName),
									},
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
			syncCtx, syncer := startHTTPRouteSyncer(t, hostRefObjects(testRouteNamespace), []runtime.Object{tc.route, virtualGateway()}, nil)
			translator, recorder := gatewayapitestutil.WithFakeEventRecorder(syncer.GenericTranslator)
			syncer.GenericTranslator = translator

			_, err := syncer.SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(tc.route.DeepCopy()))
			// Unsupported reference kinds are terminal: the route is not synced to the
			// host and is not requeued.
			assert.NilError(t, err)

			storedHostRoute := &gatewayv1.HTTPRoute{}
			err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: hostName(testRouteName), Namespace: hostNamespace(testRouteNamespace)}, storedHostRoute)
			assert.Assert(t, kerrors.IsNotFound(err))

			event, ok := gatewayapitestutil.NextEvent(recorder)
			assert.Assert(t, ok)
			assert.Assert(t, strings.Contains(event, "Warning UnsupportedReference"))
			assert.Assert(t, strings.Contains(event, tc.expectedErr))
		})
	}
}

func TestSyncDeletesHostForUnsupportedRefsOnUpdate(t *testing.T) {
	vRoute := httpRoute(virtualRouteMeta(), gatewayv1.HTTPRouteSpec{
		CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(testGatewayName)}},
		},
		Rules: []gatewayv1.HTTPRouteRule{
			{
				BackendRefs: []gatewayv1.HTTPBackendRef{
					{
						BackendRef: gatewayv1.BackendRef{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Group: ptr.To(testUnsupportedGroup),
								Kind:  ptr.To(gatewayv1.Kind("ExampleBackend")),
								Name:  gatewayv1.ObjectName(testServiceName),
							},
						},
					},
				},
			},
		},
	})
	pRoute := httpRoute(hostRouteMeta(), gatewayv1.HTTPRouteSpec{})
	syncCtx, syncer := startHTTPRouteSyncer(t, []runtime.Object{pRoute.DeepCopy()}, []runtime.Object{vRoute.DeepCopy(), virtualGateway()}, nil)
	translator, recorder := gatewayapitestutil.WithFakeEventRecorder(syncer.GenericTranslator)
	syncer.GenericTranslator = translator

	pRoute.ResourceVersion = "999"
	vRoute.ResourceVersion = "999"
	_, err := syncer.Sync(syncCtx, synccontext.NewSyncEventWithOld(pRoute.DeepCopy(), pRoute.DeepCopy(), vRoute.DeepCopy(), vRoute.DeepCopy()))
	// Introducing an unsupported reference on update is terminal: the previously synced
	// host route is removed and the route is not requeued.
	assert.NilError(t, err)

	storedHostRoute := &gatewayv1.HTTPRoute{}
	err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{Name: pRoute.Name, Namespace: pRoute.Namespace}, storedHostRoute)
	assert.Assert(t, kerrors.IsNotFound(err))

	event, ok := gatewayapitestutil.NextEvent(recorder)
	assert.Assert(t, ok)
	assert.Assert(t, strings.Contains(event, "Warning UnsupportedReference"))
	assert.Assert(t, strings.Contains(event, `backendRef group "example.com" kind "ExampleBackend" is not supported`))
}

func TestPreserveRequestMirrorFilters(t *testing.T) {
	mirrorFilterForService := func(service string) gatewayv1.HTTPRouteFilter {
		return gatewayv1.HTTPRouteFilter{
			Type: gatewayv1.HTTPRouteFilterRequestMirror,
			RequestMirror: &gatewayv1.HTTPRequestMirrorFilter{
				BackendRef: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(service),
				},
			},
		}
	}

	enabledAnnotations := map[string]string{
		constants.PreserveRequestMirrorFiltersAnnotation: "true",
	}

	hostSpec := gatewayv1.HTTPRouteSpec{
		Rules: []gatewayv1.HTTPRouteRule{
			{Filters: []gatewayv1.HTTPRouteFilter{mirrorFilterForService(testMirrorService)}},
		},
	}

	t.Run("annotation preserves mirror filter", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{}},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1)
		assert.Equal(t, desiredSpec.Rules[0].Filters[0].Type, gatewayv1.HTTPRouteFilterRequestMirror)
		assert.Equal(t, string(desiredSpec.Rules[0].Filters[0].RequestMirror.BackendRef.Name), testMirrorService)
	})

	t.Run("missing annotation does not preserve mirror filter", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{}},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, nil)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 0)
	})

	t.Run("annotation set to non-true does not preserve mirror filter", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{}},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, map[string]string{
			constants.PreserveRequestMirrorFiltersAnnotation: "false",
		})

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 0)
	})

	t.Run("does not duplicate an existing mirror filter on the desired rule", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{
				Filters: []gatewayv1.HTTPRouteFilter{mirrorFilterForService(testMirrorService)},
			}},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1)
	})

	t.Run("ignores extra rules on host that do not exist on desired", func(t *testing.T) {
		hostWithExtra := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{Filters: []gatewayv1.HTTPRouteFilter{mirrorFilterForService(testMirrorService)}},
				{Filters: []gatewayv1.HTTPRouteFilter{mirrorFilterForService("other-mirror")}},
			},
		}

		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{}},
		}

		preserveRequestMirrorFilters(hostWithExtra, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules), 1)
		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1)
		assert.Equal(t, string(desiredSpec.Rules[0].Filters[0].RequestMirror.BackendRef.Name), testMirrorService)
	})

	t.Run("no host rule at index does not panic and skips that desired rule", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{},
				{},
				{},
			},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1)
		assert.Equal(t, len(desiredSpec.Rules[1].Filters), 0)
		assert.Equal(t, len(desiredSpec.Rules[2].Filters), 0)
	})

	t.Run("named rules correlate by name even when reordered", func(t *testing.T) {
		ruleA := gatewayv1.SectionName("rule-a")
		ruleB := gatewayv1.SectionName("rule-b")

		// Host has [rule-b at index 0, rule-a at index 1]; mirror filter only on rule-a.
		hostNamed := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{Name: &ruleB},
				{Name: &ruleA, Filters: []gatewayv1.HTTPRouteFilter{mirrorFilterForService(testMirrorService)}},
			},
		}

		// Desired has [rule-a at index 0, rule-b at index 1]; positional matching would attach
		// the mirror filter to the wrong rule.
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{Name: &ruleA},
				{Name: &ruleB},
			},
		}

		preserveRequestMirrorFilters(hostNamed, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1)
		assert.Equal(t, string(desiredSpec.Rules[0].Filters[0].RequestMirror.BackendRef.Name), testMirrorService)
		assert.Equal(t, len(desiredSpec.Rules[1].Filters), 0)
	})

	t.Run("named desired rule without matching host rule does not fall back to positional", func(t *testing.T) {
		ruleA := gatewayv1.SectionName("rule-a")

		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{Name: &ruleA}},
		}

		preserveRequestMirrorFilters(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 0)
	})

	t.Run("nil desired spec is a no-op", func(t *testing.T) {
		// Just verify no panic on a nil pointer.
		preserveRequestMirrorFilters(hostSpec, nil, enabledAnnotations)
	})
}

func TestPreserveHostRule(t *testing.T) {
	const managedRuleName gatewayv1.SectionName = "external-managed-rule"

	managedRule := func() gatewayv1.HTTPRouteRule {
		name := managedRuleName
		return gatewayv1.HTTPRouteRule{
			Name: &name,
			Matches: []gatewayv1.HTTPRouteMatch{
				{Path: &gatewayv1.HTTPPathMatch{Type: ptr.To(gatewayv1.PathMatchPathPrefix), Value: ptr.To("/")}},
			},
			BackendRefs: []gatewayv1.HTTPBackendRef{
				{BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: "external-backend",
					Port: ptr.To(gatewayv1.PortNumber(9090)),
				}}},
			},
		}
	}

	userRule := func(name string) gatewayv1.HTTPRouteRule {
		ret := gatewayv1.HTTPRouteRule{
			BackendRefs: []gatewayv1.HTTPBackendRef{
				{BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(name),
					Port: ptr.To(gatewayv1.PortNumber(80)),
				}}},
			},
		}
		return ret
	}

	enabledAnnotations := map[string]string{
		constants.PreserveHostRuleAnnotation: string(managedRuleName),
	}

	t.Run("annotation prepends a deep copy of the named host rule", func(t *testing.T) {
		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedRule(), userRule("a"), userRule("b")},
		}
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a"), userRule("b")},
		}

		preserveHostRule(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules), 3)
		assert.Assert(t, desiredSpec.Rules[0].Name != nil)
		assert.Equal(t, *desiredSpec.Rules[0].Name, managedRuleName)
		assert.Equal(t, string(desiredSpec.Rules[1].BackendRefs[0].Name), "a")
		assert.Equal(t, string(desiredSpec.Rules[2].BackendRefs[0].Name), "b")
	})

	t.Run("missing annotation is a no-op", func(t *testing.T) {
		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedRule(), userRule("a")},
		}
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a")},
		}

		preserveHostRule(hostSpec, desiredSpec, nil)

		assert.Equal(t, len(desiredSpec.Rules), 1)
	})

	t.Run("empty annotation value is a no-op", func(t *testing.T) {
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a")},
		}

		preserveHostRule(
			gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{managedRule()}},
			desiredSpec,
			map[string]string{constants.PreserveHostRuleAnnotation: ""},
		)

		assert.Equal(t, len(desiredSpec.Rules), 1)
	})

	t.Run("annotation present but no host rule with that name is a no-op", func(t *testing.T) {
		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a"), userRule("b")},
		}
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a"), userRule("b")},
		}

		preserveHostRule(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules), 2)
	})

	t.Run("desired already contains a rule with the managed name is a no-op", func(t *testing.T) {
		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedRule(), userRule("a")},
		}
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedRule(), userRule("a")},
		}

		preserveHostRule(hostSpec, desiredSpec, enabledAnnotations)

		assert.Equal(t, len(desiredSpec.Rules), 2)
	})

	t.Run("prepended rule is a deep copy", func(t *testing.T) {
		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedRule(), userRule("a")},
		}
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{userRule("a")},
		}

		preserveHostRule(hostSpec, desiredSpec, enabledAnnotations)

		desiredSpec.Rules[0].BackendRefs[0].Name = "mutated"
		assert.Equal(t, string(hostSpec.Rules[0].BackendRefs[0].Name), "external-backend",
			"mutating desired rule must not affect hostSpec")
	})

	t.Run("nil desired spec is a no-op", func(t *testing.T) {
		preserveHostRule(
			gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{managedRule()}},
			nil,
			enabledAnnotations,
		)
	})

	t.Run("interaction with preserveRequestMirrorFilters: managed rule keeps its mirror filter exactly once", func(t *testing.T) {
		// Managed host rule carries a RequestMirror filter. User rule at index 1 also
		// carries one. Both annotations set. After running both preservation steps in
		// Sync's order, the managed rule must have exactly one mirror filter (no
		// duplication), and the user rule's mirror filter must be preserved too.
		mirror := gatewayv1.HTTPRouteFilter{
			Type: gatewayv1.HTTPRouteFilterRequestMirror,
			RequestMirror: &gatewayv1.HTTPRequestMirrorFilter{
				BackendRef: gatewayv1.BackendObjectReference{Name: "mirror-backend", Port: ptr.To(gatewayv1.PortNumber(9091))},
			},
		}
		managedWithMirror := managedRule()
		managedWithMirror.Filters = []gatewayv1.HTTPRouteFilter{mirror}

		userNameA := gatewayv1.SectionName("user-a")
		userWithMirror := userRule("a")
		userWithMirror.Name = &userNameA
		userWithMirror.Filters = []gatewayv1.HTTPRouteFilter{mirror}

		hostSpec := gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{managedWithMirror, userWithMirror},
		}
		// desiredSpec is what specToHost produces from virtual — the tenant's view, no
		// managed rule, mirror filters not yet present.
		desiredUser := userRule("a")
		desiredUser.Name = &userNameA
		desiredSpec := &gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{desiredUser},
		}

		annotations := map[string]string{
			constants.PreserveHostRuleAnnotation:             string(managedRuleName),
			constants.PreserveRequestMirrorFiltersAnnotation: "true",
		}

		preserveHostRule(hostSpec, desiredSpec, annotations)
		preserveRequestMirrorFilters(hostSpec, desiredSpec, annotations)

		assert.Equal(t, len(desiredSpec.Rules), 2)
		assert.Assert(t, desiredSpec.Rules[0].Name != nil)
		assert.Equal(t, *desiredSpec.Rules[0].Name, managedRuleName)
		assert.Equal(t, len(desiredSpec.Rules[0].Filters), 1,
			"managed rule must carry exactly one mirror filter (no duplication)")
		assert.Equal(t, desiredSpec.Rules[0].Filters[0].Type, gatewayv1.HTTPRouteFilterRequestMirror)
		assert.Equal(t, len(desiredSpec.Rules[1].Filters), 1,
			"user rule's mirror filter must be preserved via name-based correlation")
		assert.Equal(t, desiredSpec.Rules[1].Filters[0].Type, gatewayv1.HTTPRouteFilterRequestMirror)
	})
}

func newHTTPRouteRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	vConfig.Sync.ToHost.GatewayAPI.Enabled = true
	return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
}

func startHTTPRouteSyncer(
	t *testing.T,
	initialPhysicalState []runtime.Object,
	initialVirtualState []runtime.Object,
	adjustConfig func(*config.VirtualClusterConfig),
) (*synccontext.SyncContext, *httpRouteSyncer) {
	t.Helper()

	pClient := testingutil.NewFakeClient(scheme.Scheme, initialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, initialVirtualState...)
	vConfig := testingutil.NewFakeConfig()
	if adjustConfig != nil {
		adjustConfig(vConfig)
	}

	registerContext := newHTTPRouteRegisterContext(vConfig, pClient, vClient)
	syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
	return syncCtx, syncer.(*httpRouteSyncer)
}

type httpRouteOption func(*gatewayv1.HTTPRoute)

func withStatus(status gatewayv1.HTTPRouteStatus) httpRouteOption {
	return func(route *gatewayv1.HTTPRoute) {
		route.Status = status
	}
}

func httpRoute(meta metav1.ObjectMeta, spec gatewayv1.HTTPRouteSpec, opts ...httpRouteOption) *gatewayv1.HTTPRoute {
	ret := &gatewayv1.HTTPRoute{
		ObjectMeta: meta,
		Spec:       spec,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}

func routeSpec() gatewayv1.HTTPRouteSpec {
	return routeSpecWithParentNamespace("")
}

func routeSpecWithParentNamespace(parentNamespace string) gatewayv1.HTTPRouteSpec {
	parentRef := gatewayv1.ParentReference{Name: gatewayv1.ObjectName(testGatewayName)}
	if parentNamespace != "" {
		parentRef.Namespace = ptr.To(gatewayv1.Namespace(parentNamespace))
	}

	return gatewayv1.HTTPRouteSpec{
		CommonRouteSpec: gatewayv1.CommonRouteSpec{
			ParentRefs: []gatewayv1.ParentReference{
				parentRef,
			},
		},
		Hostnames: []gatewayv1.Hostname{"example.com"},
		Rules: []gatewayv1.HTTPRouteRule{
			{
				BackendRefs: []gatewayv1.HTTPBackendRef{
					serviceBackendRef(testServiceName, withBackendRefFilter(mirrorFilter(testMirrorService))),
				},
				Filters: []gatewayv1.HTTPRouteFilter{
					mirrorFilter(testMirrorService),
					externalAuthFilter(testAuthService),
				},
			},
		},
	}
}

func hostRouteSpec() gatewayv1.HTTPRouteSpec {
	return hostRouteSpecWithParentNamespace("")
}

func hostRouteSpecWithParentNamespace(parentNamespace string) gatewayv1.HTTPRouteSpec {
	spec := routeSpecWithParentNamespace(parentNamespace)
	ret := *spec.DeepCopy()
	ret.ParentRefs[0].Name = gatewayv1.ObjectName(hostNameForNamespace(testGatewayName, refNamespaceOrDefault(parentNamespace, testRouteNamespace)))
	if parentNamespace != "" {
		ret.ParentRefs[0].Namespace = ptr.To(gatewayv1.Namespace(hostNamespace(parentNamespace)))
	}
	ret.Rules[0].BackendRefs[0].Name = gatewayv1.ObjectName(hostName(testServiceName))
	ret.Rules[0].BackendRefs[0].Filters[0].RequestMirror.BackendRef.Name = gatewayv1.ObjectName(hostName(testMirrorService))
	ret.Rules[0].Filters[0].RequestMirror.BackendRef.Name = gatewayv1.ObjectName(hostName(testMirrorService))
	ret.Rules[0].Filters[1].ExternalAuth.BackendRef.Name = gatewayv1.ObjectName(hostName(testAuthService))
	return ret
}

func refNamespaceOrDefault(namespace, defaultNamespace string) string {
	if namespace == "" {
		return defaultNamespace
	}

	return namespace
}

type backendRefOption func(*gatewayv1.HTTPBackendRef)

func withBackendRefFilter(filter gatewayv1.HTTPRouteFilter) backendRefOption {
	return func(ref *gatewayv1.HTTPBackendRef) {
		ref.Filters = append(ref.Filters, filter)
	}
}

func serviceBackendRef(name string, opts ...backendRefOption) gatewayv1.HTTPBackendRef {
	ret := gatewayv1.HTTPBackendRef{
		BackendRef: gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Name: gatewayv1.ObjectName(name),
				Port: ptr.To(gatewayv1.PortNumber(80)),
			},
		},
	}
	for _, opt := range opts {
		opt(&ret)
	}
	return ret
}

func mirrorFilter(serviceName string) gatewayv1.HTTPRouteFilter {
	return gatewayv1.HTTPRouteFilter{
		Type: gatewayv1.HTTPRouteFilterRequestMirror,
		RequestMirror: &gatewayv1.HTTPRequestMirrorFilter{
			BackendRef: gatewayv1.BackendObjectReference{
				Name: gatewayv1.ObjectName(serviceName),
				Port: ptr.To(gatewayv1.PortNumber(80)),
			},
		},
	}
}

func externalAuthFilter(serviceName string) gatewayv1.HTTPRouteFilter {
	return gatewayv1.HTTPRouteFilter{
		Type: gatewayv1.HTTPRouteFilterExternalAuth,
		ExternalAuth: &gatewayv1.HTTPExternalAuthFilter{
			ExternalAuthProtocol: gatewayv1.HTTPRouteExternalAuthHTTPProtocol,
			BackendRef: gatewayv1.BackendObjectReference{
				Name: gatewayv1.ObjectName(serviceName),
				Port: ptr.To(gatewayv1.PortNumber(80)),
			},
		},
	}
}

func virtualRouteStatus() gatewayv1.HTTPRouteStatus {
	status := hostRouteStatus()
	status.Parents[0].ParentRef.Name = gatewayv1.ObjectName(testGatewayName)
	status.Parents[0].ParentRef.Namespace = nil
	return status
}

func hostRouteStatus() gatewayv1.HTTPRouteStatus {
	return hostRouteStatusForNamespace(testRouteNamespace, true)
}

func hostRouteStatusForNamespace(parentNamespace string, includeNamespace bool) gatewayv1.HTTPRouteStatus {
	parentRef := gatewayv1.ParentReference{Name: gatewayv1.ObjectName(hostNameForNamespace(testGatewayName, parentNamespace))}
	if includeNamespace {
		parentRef.Namespace = ptr.To(gatewayv1.Namespace(hostNamespace(parentNamespace)))
	}

	return gatewayv1.HTTPRouteStatus{
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
			translate.KindAnnotation:          mappings.HTTPRoutes().String(),
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

type gatewayOption func(*gatewayv1.Gateway)

func withAllowedRoutesAll() gatewayOption {
	return func(gateway *gatewayv1.Gateway) {
		from := gatewayv1.NamespacesFromAll
		gateway.Spec.Listeners[0].AllowedRoutes = &gatewayv1.AllowedRoutes{
			Namespaces: &gatewayv1.RouteNamespaces{From: &from},
		}
	}
}

func withAllowedRoutesSelector(matchLabels map[string]string) gatewayOption {
	return func(gateway *gatewayv1.Gateway) {
		from := gatewayv1.NamespacesFromSelector
		gateway.Spec.Listeners[0].AllowedRoutes = &gatewayv1.AllowedRoutes{
			Namespaces: &gatewayv1.RouteNamespaces{
				From:     &from,
				Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			},
		}
	}
}

func virtualGatewayWithNamespace(namespace string, opts ...gatewayOption) *gatewayv1.Gateway {
	ret := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testGatewayName,
			Namespace: namespace,
		},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
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
		hostService(testMirrorService, namespace),
		hostService(testAuthService, namespace),
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

func referenceGrant(namespace, fromKind, fromNamespace, toKind, toName string) *gatewayv1.ReferenceGrant {
	return &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-" + toName,
			Namespace: namespace,
		},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{
				{
					Group:     gatewayv1.Group(gatewayv1.GroupVersion.Group),
					Kind:      gatewayv1.Kind(fromKind),
					Namespace: gatewayv1.Namespace(fromNamespace),
				},
			},
			To: []gatewayv1.ReferenceGrantTo{
				{
					Group: corev1.GroupName,
					Kind:  gatewayv1.Kind(toKind),
					Name:  ptr.To(gatewayv1.ObjectName(toName)),
				},
			},
		},
	}
}
