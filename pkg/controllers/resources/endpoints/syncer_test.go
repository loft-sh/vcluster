package endpoints

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *endpointsSyncer) {
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	syncCtx, fakeSyncer := syncertesting.FakeStartSyncer(t, ctx, New)
	return syncCtx, fakeSyncer.(*endpointsSyncer)
}

func TestExistingEndpoints(t *testing.T) {
	vEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoints",
			Namespace: "test",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
				},
			},
		},
	}
	vService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoints",
			Namespace: "test",
		},
	}
	pEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpoints.Name, vEndpoints.Namespace).Name,
			Namespace: "test",
		},
	}
	pService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpoints.Name, vEndpoints.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpoints.Name,
				translate.NamespaceAnnotation:     vEndpoints.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Service").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vEndpoints.Name, vEndpoints.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vEndpoints.Namespace,
			},
		},
	}
	expectedEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpoints.Name, vEndpoints.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpoints.Name,
				translate.NamespaceAnnotation:     vEndpoints.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Endpoints").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vEndpoints.Name, vEndpoints.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vEndpoints.Namespace,
			},
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
				},
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Override endpoints even if they are not managed",
			InitialVirtualState: []runtime.Object{
				vEndpoints,
				vService,
			},
			InitialPhysicalState: []runtime.Object{
				pEndpoints,
				pService,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Endpoints"): {
					expectedEndpoints,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: vEndpoints.Namespace,
					Name:      vEndpoints.Name,
				}})
				assert.NilError(t, err)
			},
		},
	})
}

func TestSync(t *testing.T) {
	baseEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test",
		},
	}
	updatedEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-secret",
			Namespace:       "test",
			ResourceVersion: "1",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
			},
		},
	}
	syncedEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, baseEndpoints.Name, baseEndpoints.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          baseEndpoints.Name,
				translate.NamespaceAnnotation:     baseEndpoints.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Endpoints").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, baseEndpoints.Name, baseEndpoints.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: baseEndpoints.Namespace,
			},
		},
	}
	syncedUpdatedEndpoints := &corev1.Endpoints{
		ObjectMeta: syncedEndpoints.ObjectMeta,
		Subsets:    updatedEndpoints.Subsets,
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
				baseEndpoints,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Endpoints"): {
					syncedEndpoints,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointsSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseEndpoints))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Forward update",
			InitialVirtualState: []runtime.Object{
				updatedEndpoints,
			},
			InitialPhysicalState: []runtime.Object{
				syncedEndpoints,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Endpoints"): {
					syncedUpdatedEndpoints,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointsSyncer).Sync(syncCtx, synccontext.NewSyncEvent(syncedEndpoints, updatedEndpoints))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Don't sync default/kubernetes endpoint",
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				ok, _ := syncer.(*endpointsSyncer).ReconcileStart(syncCtx, request)
				assert.Equal(t, ok, true)
			},
		},
	})
}
