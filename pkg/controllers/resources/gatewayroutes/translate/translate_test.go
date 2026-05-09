package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	testRouteName      = "route"
	testRouteNamespace = "default"
	testServiceName    = "backend"
)

func TestParentRefToHostSupportsService(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostService(testServiceName, testRouteNamespace),
	}, nil)
	ref := serviceParentRef(testServiceName)

	err := ParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName(testServiceName, testRouteNamespace))
	assert.Assert(t, ref.Namespace == nil)
}

func TestParentRefToVirtualSupportsService(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	addMapping(t, syncCtx, serviceMapping(testServiceName, testRouteNamespace))
	ref := serviceParentRef(hostName(testServiceName, testRouteNamespace))

	err := ParentRefToVirtual(syncCtx, hostNamespace(testRouteNamespace), testRouteNamespace, &ref, nil)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), testServiceName)
	assert.Assert(t, ref.Namespace == nil)
}

func TestParentRefToVirtualPreservesExplicitSpecNamespace(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	addMapping(t, syncCtx, gatewayMapping("gateway", testRouteNamespace))
	ref := gatewayv1.ParentReference{
		Name: gatewayv1.ObjectName(hostName("gateway", testRouteNamespace)),
	}
	specParentRefs := []gatewayv1.ParentReference{
		{
			Name:      gatewayv1.ObjectName(hostName("gateway", testRouteNamespace)),
			Namespace: ptr.To(gatewayv1.Namespace(hostNamespace(testRouteNamespace))),
		},
	}

	err := ParentRefToVirtual(syncCtx, hostNamespace(testRouteNamespace), testRouteNamespace, &ref, specParentRefs)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), "gateway")
	assert.Assert(t, ref.Namespace != nil)
	assert.Equal(t, string(*ref.Namespace), testRouteNamespace)
}

func TestParentRefToVirtualIgnoresExplicitSpecNamespaceOnDifferentParent(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	addMapping(t, syncCtx, gatewayMapping("gateway", testRouteNamespace))
	ref := gatewayv1.ParentReference{
		Name: gatewayv1.ObjectName(hostName("gateway", testRouteNamespace)),
	}
	specParentRefs := []gatewayv1.ParentReference{
		{
			Name:      gatewayv1.ObjectName(hostName("gateway", testRouteNamespace)),
			Namespace: ptr.To(gatewayv1.Namespace("other-host-ns")),
		},
		{
			Name: gatewayv1.ObjectName(hostName("gateway", testRouteNamespace)),
		},
	}

	err := ParentRefToVirtual(syncCtx, hostNamespace(testRouteNamespace), testRouteNamespace, &ref, specParentRefs)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), "gateway")
	assert.Assert(t, ref.Namespace == nil)
}

func TestSecretObjectRefToHost(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostSecret("tls-cert", testRouteNamespace),
	}, nil)
	ref := gatewayv1.SecretObjectReference{Name: gatewayv1.ObjectName("tls-cert")}

	err := SecretObjectRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName("tls-cert", testRouteNamespace))
	assert.Assert(t, ref.Namespace == nil)
}

func TestLocalObjectRefToHost(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostConfigMap("ca-bundle", testRouteNamespace),
	}, nil)
	ref := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(corev1.GroupName),
		Kind:  gatewayv1.Kind("ConfigMap"),
		Name:  gatewayv1.ObjectName("ca-bundle"),
	}

	err := LocalObjectRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName("ca-bundle", testRouteNamespace))
}

func TestLocalObjectRefToHostSupportsSecret(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostSecret("ca-bundle", testRouteNamespace),
	}, nil)
	ref := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(corev1.GroupName),
		Kind:  gatewayv1.Kind("Secret"),
		Name:  gatewayv1.ObjectName("ca-bundle"),
	}

	err := LocalObjectRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName("ca-bundle", testRouteNamespace))
}

func TestPolicyTargetRefToHost(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostService(testServiceName, testRouteNamespace),
	}, nil)
	ref := gatewayv1.LocalPolicyTargetReferenceWithSectionName{
		LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
			Group: gatewayv1.Group(corev1.GroupName),
			Kind:  gatewayv1.Kind("Service"),
			Name:  gatewayv1.ObjectName(testServiceName),
		},
	}

	err := PolicyTargetRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName(testServiceName, testRouteNamespace))
}

func TestReferenceGrantToToHostSupportsService(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostService(testServiceName, testRouteNamespace),
	}, nil)
	name := gatewayv1.ObjectName(testServiceName)
	ref := gatewayv1.ReferenceGrantTo{
		Group: corev1.GroupName,
		Kind:  "Service",
		Name:  &name,
	}

	err := ReferenceGrantToToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Assert(t, ref.Name != nil)
	assert.Equal(t, string(*ref.Name), hostName(testServiceName, testRouteNamespace))
}

func TestReferenceGrantToToHostSupportsSecret(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostSecret("tls-cert", testRouteNamespace),
	}, nil)
	name := gatewayv1.ObjectName("tls-cert")
	ref := gatewayv1.ReferenceGrantTo{
		Group: corev1.GroupName,
		Kind:  "Secret",
		Name:  &name,
	}

	err := ReferenceGrantToToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(*ref.Name), hostName("tls-cert", testRouteNamespace))
}

func TestReferenceGrantToToHostSupportsConfigMap(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostConfigMap("ca-bundle", testRouteNamespace),
	}, nil)
	name := gatewayv1.ObjectName("ca-bundle")
	ref := gatewayv1.ReferenceGrantTo{
		Group: corev1.GroupName,
		Kind:  "ConfigMap",
		Name:  &name,
	}

	err := ReferenceGrantToToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(*ref.Name), hostName("ca-bundle", testRouteNamespace))
}

func TestReferenceGrantToToHostNilNameIsNoOp(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	ref := gatewayv1.ReferenceGrantTo{
		Group: corev1.GroupName,
		Kind:  "Service",
	}

	err := ReferenceGrantToToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Assert(t, ref.Name == nil)
}

func TestReferenceGrantToToHostRejectsUnsupportedKind(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	name := gatewayv1.ObjectName("anything")
	ref := gatewayv1.ReferenceGrantTo{
		Group: "example.com",
		Kind:  "CustomThing",
		Name:  &name,
	}

	err := ReferenceGrantToToHost(syncCtx, testRouteNamespace, &ref)
	assert.ErrorContains(t, err, `referenceGrant to group "example.com" kind "CustomThing" is not supported`)
}

func TestParentRefToHostRejectsObjectWhenManagedChecksDisagree(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		bareHostService(testServiceName, testRouteNamespace),
	}, nil)
	addMapping(t, syncCtx, serviceMapping(testServiceName, testRouteNamespace))
	ref := serviceParentRef(testServiceName)

	err := ParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.ErrorContains(t, err, `is not managed by vCluster`)
}

func TestMissingServiceParentRefRecordsReferenceForRequeue(t *testing.T) {
	syncCtx, registerCtx := newGatewayRouteTestContext(t, nil, nil)
	routeMapping := addRouteMapping(t, syncCtx)
	ref := serviceParentRef(testServiceName)

	err := ParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.ErrorContains(t, err, `referenced Service "backend" in namespace "default" has no synced host object`)

	references := syncCtx.Mappings.Store().ReferencesTo(syncCtx, synccontext.Object{
		GroupVersionKind: mappings.Services(),
		NamespacedName:   types.NamespacedName{Name: testServiceName, Namespace: testRouteNamespace},
	})
	assert.Equal(t, len(references), 1)
	assert.DeepEqual(t, references[0], routeMapping)

	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]())
	defer queue.ShutDown()

	enqueueObjectsReferencingObject(registerCtx, mappings.HTTPRoutes(), serviceMapping(testServiceName, testRouteNamespace), queue)
	item, shutdown := queue.Get()
	assert.Assert(t, !shutdown)
	defer queue.Done(item)
	assert.DeepEqual(t, item, reconcile.Request{NamespacedName: routeMapping.VirtualName})
}

func TestEnqueueObjectsReferencingObject(t *testing.T) {
	syncCtx, registerCtx := newGatewayRouteTestContext(t, nil, nil)
	policyMapping := synccontext.NameMapping{
		GroupVersionKind: mappings.BackendTLSPolicies(),
		VirtualName:      types.NamespacedName{Name: "policy", Namespace: testRouteNamespace},
		HostName:         types.NamespacedName{Name: hostName("policy", testRouteNamespace), Namespace: hostNamespace(testRouteNamespace)},
	}
	configMapMapping := synccontext.NameMapping{
		GroupVersionKind: mappings.ConfigMaps(),
		VirtualName:      types.NamespacedName{Name: "ca-bundle", Namespace: testRouteNamespace},
		HostName:         types.NamespacedName{Name: hostName("ca-bundle", testRouteNamespace), Namespace: hostNamespace(testRouteNamespace)},
	}
	addMapping(t, syncCtx, policyMapping)
	err := syncCtx.Mappings.Store().AddReferenceAndSave(syncCtx, configMapMapping, policyMapping)
	assert.NilError(t, err)

	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]())
	defer queue.ShutDown()

	enqueueObjectsReferencingObject(registerCtx, mappings.BackendTLSPolicies(), configMapMapping, queue)
	item, shutdown := queue.Get()
	assert.Assert(t, !shutdown)
	defer queue.Done(item)
	assert.DeepEqual(t, item, reconcile.Request{NamespacedName: policyMapping.VirtualName})
}

func TestParentStatusHostNamespace(t *testing.T) {
	sectionName := gatewayv1.SectionName("https")
	port := gatewayv1.PortNumber(443)
	specParentRefs := []gatewayv1.ParentReference{
		{
			Name:        gatewayv1.ObjectName("gateway"),
			Namespace:   ptr.To(gatewayv1.Namespace("gateway-ns")),
			SectionName: &sectionName,
			Port:        &port,
		},
	}

	assert.Equal(t, ParentStatusHostNamespace("route-ns", specParentRefs, gatewayv1.ParentReference{
		Name:        gatewayv1.ObjectName("gateway"),
		SectionName: &sectionName,
		Port:        &port,
	}), "route-ns")
	assert.Equal(t, ParentStatusHostNamespace("route-ns", specParentRefs, gatewayv1.ParentReference{
		Name:      gatewayv1.ObjectName("gateway"),
		Namespace: ptr.To(gatewayv1.Namespace("status-ns")),
	}), "status-ns")
	assert.Equal(t, ParentStatusHostNamespace("route-ns", specParentRefs, gatewayv1.ParentReference{
		Name: gatewayv1.ObjectName("other-gateway"),
	}), "route-ns")

	ambiguousSpecParentRefs := []gatewayv1.ParentReference{
		{
			Name:        gatewayv1.ObjectName("shared-gateway"),
			Namespace:   ptr.To(gatewayv1.Namespace("other-ns")),
			SectionName: &sectionName,
			Port:        &port,
		},
		{
			Name:        gatewayv1.ObjectName("shared-gateway"),
			SectionName: &sectionName,
			Port:        &port,
		},
	}
	assert.Equal(t, ParentStatusHostNamespace("route-ns", ambiguousSpecParentRefs, gatewayv1.ParentReference{
		Name:        gatewayv1.ObjectName("shared-gateway"),
		SectionName: &sectionName,
		Port:        &port,
	}), "route-ns")
}

func newGatewayRouteTestContext(t *testing.T, initialPhysicalState, initialVirtualState []runtime.Object) (*synccontext.SyncContext, *synccontext.RegisterContext) {
	t.Helper()

	pClient := testingutil.NewFakeClient(scheme.Scheme, initialPhysicalState...)
	vClient := testingutil.NewFakeClient(scheme.Scheme, initialVirtualState...)
	vConfig := testingutil.NewFakeConfig()
	registerCtx := newGatewayRouteRegisterContext(vConfig, pClient, vClient)
	return registerCtx.ToSyncContext("gatewayroutes-test"), registerCtx
}

func newGatewayRouteRegisterContext(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
	vConfig.Sync.ToHost.GatewayAPI.Enabled = true
	return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
}

func addRouteMapping(t *testing.T, ctx *synccontext.SyncContext) synccontext.NameMapping {
	t.Helper()

	nameMapping := synccontext.NameMapping{
		GroupVersionKind: mappings.HTTPRoutes(),
		VirtualName:      types.NamespacedName{Name: testRouteName, Namespace: testRouteNamespace},
		HostName:         types.NamespacedName{Name: hostName(testRouteName, testRouteNamespace), Namespace: hostNamespace(testRouteNamespace)},
	}
	addMapping(t, ctx, nameMapping)
	ctx.Context = synccontext.WithMapping(ctx.Context, nameMapping)
	return nameMapping
}

func addMapping(t *testing.T, ctx *synccontext.SyncContext, nameMapping synccontext.NameMapping) {
	t.Helper()

	err := ctx.Mappings.Store().AddReferenceAndSave(ctx, nameMapping, nameMapping)
	assert.NilError(t, err)
}

func serviceMapping(name, namespace string) synccontext.NameMapping {
	return synccontext.NameMapping{
		GroupVersionKind: mappings.Services(),
		VirtualName:      types.NamespacedName{Name: name, Namespace: namespace},
		HostName:         types.NamespacedName{Name: hostName(name, namespace), Namespace: hostNamespace(namespace)},
	}
}

func gatewayMapping(name, namespace string) synccontext.NameMapping {
	return synccontext.NameMapping{
		GroupVersionKind: mappings.Gateways(),
		VirtualName:      types.NamespacedName{Name: name, Namespace: namespace},
		HostName:         types.NamespacedName{Name: hostName(name, namespace), Namespace: hostNamespace(namespace)},
	}
}

func serviceParentRef(name string) gatewayv1.ParentReference {
	return gatewayv1.ParentReference{
		Group: ptr.To(gatewayv1.Group(corev1.GroupName)),
		Kind:  ptr.To(gatewayv1.Kind("Service")),
		Name:  gatewayv1.ObjectName(name),
	}
}

func managedHostService(name, namespace string) *corev1.Service {
	return utiltranslate.HostMetadata(virtualService(name, namespace), types.NamespacedName{
		Name:      hostName(name, namespace),
		Namespace: hostNamespace(namespace),
	})
}

func managedHostSecret(name, namespace string) *corev1.Secret {
	return utiltranslate.HostMetadata(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, types.NamespacedName{
		Name:      hostName(name, namespace),
		Namespace: hostNamespace(namespace),
	})
}

func managedHostConfigMap(name, namespace string) *corev1.ConfigMap {
	return utiltranslate.HostMetadata(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}, types.NamespacedName{
		Name:      hostName(name, namespace),
		Namespace: hostNamespace(namespace),
	})
}

func bareHostService(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostName(name, namespace),
			Namespace: hostNamespace(namespace),
		},
	}
}

func virtualService(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func hostName(name, namespace string) string {
	return utiltranslate.SingleNamespaceHostName(name, namespace, utiltranslate.VClusterName)
}

func hostNamespace(namespace string) string {
	if namespace == "" {
		return ""
	}

	return testingutil.DefaultTestTargetNamespace
}
