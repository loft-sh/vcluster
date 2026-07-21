package endpointslices

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *endpointSliceSyncer) {
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	syncCtx, fakeSyncer := syncertesting.FakeStartSyncer(t, ctx, New)
	return syncCtx, fakeSyncer.(*endpointSliceSyncer)
}

func TestExistingEndpointSlices(t *testing.T) {
	vEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-endpoint-slices",
			Namespace:       "test",
			ResourceVersion: "999",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-endpoint-slices",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"1.1.1.1"},
			},
		},
	}
	vService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint-slices",
			Namespace: "test",
		},
	}
	pEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
		},
	}
	pService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vService.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vService.Name,
				translate.NamespaceAnnotation:     vEndpointSlice.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Service").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vService.Name, vEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vEndpointSlice.Namespace,
			},
		},
	}
	expectedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpointSlice.Name,
				translate.NamespaceAnnotation:     vEndpointSlice.Namespace,
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.UIDAnnotation:           "",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.K8sServiceNameLabel: translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
				translate.NamespaceLabel:      vEndpointSlice.Namespace,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"1.1.1.1"},
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Override endpointSlices even if they are not managed",
			InitialVirtualState: []runtime.Object{
				vEndpointSlice,
				vService,
			},
			InitialPhysicalState: []runtime.Object{
				pEndpointSlice,
				pService,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: vEndpointSlice.Namespace,
					Name:      vEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
	})
}

func TestSync(t *testing.T) {
	baseEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-eps",
			Namespace: "test",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-svc",
			},
		},
	}
	updatedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-eps",
			Namespace:       "test",
			ResourceVersion: "1",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-svc",
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"127.0.0.1,"},
			},
		},
	}
	syncedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "999",
			Name:            translate.Default.HostName(nil, baseEndpointSlice.Name, baseEndpointSlice.Namespace).Name,
			Namespace:       "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          baseEndpointSlice.Name,
				translate.NamespaceAnnotation:     baseEndpointSlice.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, baseEndpointSlice.Name, baseEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel:      baseEndpointSlice.Namespace,
				translate.K8sServiceNameLabel: translate.Default.HostName(nil, "test-svc", baseEndpointSlice.Namespace).Name,
			},
		},
	}
	syncedUpdatedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: syncedEndpointSlice.ObjectMeta,
		Endpoints:  updatedEndpointSlice.Endpoints,
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Forward create",
			InitialVirtualState: []runtime.Object{
				baseEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					syncedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointSliceSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseEndpointSlice))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Forward update",
			InitialVirtualState: []runtime.Object{
				updatedEndpointSlice,
			},
			InitialPhysicalState: []runtime.Object{
				syncedEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					syncedUpdatedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointSliceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					syncedEndpointSlice,
					syncedEndpointSlice,
					baseEndpointSlice,
					updatedEndpointSlice,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Don't sync default/kubernetes endpointSlice",
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				ok, _ := syncer.(*endpointSliceSyncer).ReconcileStart(syncCtx, request)
				assert.Equal(t, ok, true)
			},
		},
	})
}
