package endpoints

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func newFakeSyncer(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		targetNamespace:        "test",
		currentNamespace:       "test",
		currentNamespaceClient: pClient,
		virtualClient:          vClient,

		creator:    generic.NewGenericCreator(pClient, &testingutil.FakeEventRecorder{}, "endpoints"),
		translator: translate.NewDefaultTranslator("test"),
	}
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
			Name:      translate.PhysicalName(baseEndpoints.Name, baseEndpoints.Namespace),
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:      baseEndpoints.Name,
				translate.NamespaceAnnotation: baseEndpoints.Namespace,
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

	physicalKubernetesEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "fake-kuberentes",
		},
		Subsets: updatedEndpoints.Subsets,
	}
	virtualKubernetesEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}
	syncedVirtualKubernetesEndpoints := &corev1.Endpoints{
		ObjectMeta: virtualKubernetesEndpoints.ObjectMeta,
		Subsets:    updatedEndpoints.Subsets,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
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
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Forward(ctx, baseEndpoints, log)
				if err != nil {
					t.Fatal(err)
				}
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
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, syncedEndpoints, updatedEndpoints, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Sync kubernetes service endpoints",
			InitialVirtualState: []runtime.Object{
				virtualKubernetesEndpoints,
			},
			InitialPhysicalState: []runtime.Object{
				physicalKubernetesEndpoints,
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Endpoints"): {
					syncedVirtualKubernetesEndpoints,
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Endpoints"): {
					physicalKubernetesEndpoints,
				},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				err := SyncKubernetesServiceEndpoints(ctx, vClient, pClient, physicalKubernetesEndpoints.Namespace, physicalKubernetesEndpoints.Name)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
