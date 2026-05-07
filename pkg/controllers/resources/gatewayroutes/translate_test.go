package gatewayroutes

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

func TestTranslateParentRefToHostSupportsService(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		managedHostService(testServiceName, testRouteNamespace),
	}, nil)
	ref := serviceParentRef(testServiceName)

	err := TranslateParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), hostName(testServiceName, testRouteNamespace))
	assert.Assert(t, ref.Namespace == nil)
}

func TestTranslateParentRefToVirtualSupportsService(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, nil, nil)
	addMapping(t, syncCtx, serviceMapping(testServiceName, testRouteNamespace))
	ref := serviceParentRef(hostName(testServiceName, testRouteNamespace))

	err := TranslateParentRefToVirtual(syncCtx, hostNamespace(testRouteNamespace), testRouteNamespace, &ref)
	assert.NilError(t, err)
	assert.Equal(t, string(ref.Name), testServiceName)
	assert.Assert(t, ref.Namespace == nil)
}

func TestTranslateParentRefToHostRejectsObjectWhenManagedChecksDisagree(t *testing.T) {
	syncCtx, _ := newGatewayRouteTestContext(t, []runtime.Object{
		bareHostService(testServiceName, testRouteNamespace),
	}, nil)
	addMapping(t, syncCtx, serviceMapping(testServiceName, testRouteNamespace))
	ref := serviceParentRef(testServiceName)

	err := TranslateParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.ErrorContains(t, err, `is not managed by vCluster`)
}

func TestMissingServiceParentRefRecordsReferenceForRequeue(t *testing.T) {
	syncCtx, registerCtx := newGatewayRouteTestContext(t, nil, nil)
	routeMapping := addRouteMapping(t, syncCtx)
	ref := serviceParentRef(testServiceName)

	err := TranslateParentRefToHost(syncCtx, testRouteNamespace, &ref)
	assert.ErrorContains(t, err, `referenced Service "backend" in namespace "default" has no synced host object`)

	references := syncCtx.Mappings.Store().ReferencesTo(syncCtx, synccontext.Object{
		GroupVersionKind: mappings.Services(),
		NamespacedName:   types.NamespacedName{Name: testServiceName, Namespace: testRouteNamespace},
	})
	assert.Equal(t, len(references), 1)
	assert.DeepEqual(t, references[0], routeMapping)

	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[ctrl.Request]())
	defer queue.ShutDown()

	enqueueRoutesReferencingObject(registerCtx, mappings.HTTPRoutes(), serviceMapping(testServiceName, testRouteNamespace), queue)
	item, shutdown := queue.Get()
	assert.Assert(t, !shutdown)
	defer queue.Done(item)
	assert.DeepEqual(t, item, reconcile.Request{NamespacedName: routeMapping.VirtualName})
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
	}), "gateway-ns")
	assert.Equal(t, ParentStatusHostNamespace("route-ns", specParentRefs, gatewayv1.ParentReference{
		Name:      gatewayv1.ObjectName("gateway"),
		Namespace: ptr.To(gatewayv1.Namespace("status-ns")),
	}), "status-ns")
	assert.Equal(t, ParentStatusHostNamespace("route-ns", specParentRefs, gatewayv1.ParentReference{
		Name: gatewayv1.ObjectName("other-gateway"),
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
	vConfig.Sync.ToHost.Gateways.Enabled = true
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

func serviceParentRef(name string) gatewayv1.ParentReference {
	return gatewayv1.ParentReference{
		Group: ptr.To(gatewayv1.Group(corev1.GroupName)),
		Kind:  ptr.To(gatewayv1.Kind("Service")),
		Name:  gatewayv1.ObjectName(name),
	}
}

func managedHostService(name, namespace string) *corev1.Service {
	return translate.HostMetadata(virtualService(name, namespace), types.NamespacedName{
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
	return translate.SingleNamespaceHostName(name, namespace, translate.VClusterName)
}

func hostNamespace(namespace string) string {
	if namespace == "" {
		return ""
	}

	return testingutil.DefaultTestTargetNamespace
}
