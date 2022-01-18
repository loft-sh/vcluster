package endpoints

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

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
				translator.NameAnnotation:      baseEndpoints.Name,
				translator.NamespaceAnnotation: baseEndpoints.Namespace,
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
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointsSyncer).SyncDown(syncCtx, baseEndpoints)
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
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointsSyncer).Sync(syncCtx, syncedEndpoints, updatedEndpoints)
				assert.NilError(t, err)
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
			Sync: func(ctx *synccontext.RegisterContext) {
				err := SyncKubernetesServiceEndpoints(ctx.Context, ctx.VirtualManager.GetClient(), ctx.PhysicalManager.GetClient(), physicalKubernetesEndpoints.Namespace, physicalKubernetesEndpoints.Name)
				assert.NilError(t, err)
			},
		},
	})
}
