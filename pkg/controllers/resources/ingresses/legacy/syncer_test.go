package legacy

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		virtualClient:   vClient,
		localClient:     pClient,

		creator:    generic.NewGenericCreator(pClient, &testingutil.FakeEventRecorder{}, "endpoints"),
		translator: translate.NewDefaultTranslator("test"),
	}
}

func TestSync(t *testing.T) {
	vBaseSpec := networkingv1beta1.IngressSpec{
		Backend: &networkingv1beta1.IngressBackend{
			ServiceName: "testservice",
			Resource: &corev1.TypedLocalObjectReference{
				Name: "testbackendresource",
			},
		},
		Rules: []networkingv1beta1.IngressRule{
			{
				IngressRuleValue: networkingv1beta1.IngressRuleValue{
					HTTP: &networkingv1beta1.HTTPIngressRuleValue{
						Paths: []networkingv1beta1.HTTPIngressPath{
							{
								Backend: networkingv1beta1.IngressBackend{
									ServiceName: "testbackendservice",
									Resource: &corev1.TypedLocalObjectReference{
										Name: "testbackendresource",
									},
								},
							},
						},
					},
				},
			},
		},
		TLS: []networkingv1beta1.IngressTLS{
			{
				SecretName: "testtlssecret",
			},
		},
	}
	pBaseSpec := networkingv1beta1.IngressSpec{
		Backend: &networkingv1beta1.IngressBackend{
			ServiceName: translate.PhysicalName("testservice", "test"),
			Resource: &corev1.TypedLocalObjectReference{
				Name: translate.PhysicalName("testbackendresource", "test"),
			},
		},
		Rules: []networkingv1beta1.IngressRule{
			{
				IngressRuleValue: networkingv1beta1.IngressRuleValue{
					HTTP: &networkingv1beta1.HTTPIngressRuleValue{
						Paths: []networkingv1beta1.HTTPIngressPath{
							{
								Backend: networkingv1beta1.IngressBackend{
									ServiceName: translate.PhysicalName("testbackendservice", "test"),
									Resource: &corev1.TypedLocalObjectReference{
										Name: translate.PhysicalName("testbackendresource", "test"),
									},
								},
							},
						},
					},
				},
			},
		},
		TLS: []networkingv1beta1.IngressTLS{
			{
				SecretName: translate.PhysicalName("testtlssecret", "test"),
			},
		},
	}
	changedIngressStatus := networkingv1beta1.IngressStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					IP:       "123:123:123:123",
					Hostname: "testhost",
				},
			},
		},
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:        "testingress",
		Namespace:   "test",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.PhysicalName("testingress", "test"),
		Namespace: "test",
		Labels: map[string]string{
			translate.MarkerLabel:    translate.Suffix,
			translate.NameLabel:      vObjectMeta.Name,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	baseIngress := &networkingv1beta1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec:       vBaseSpec,
	}
	createdIngress := &networkingv1beta1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec:       pBaseSpec,
	}
	noUpdateIngress := &networkingv1beta1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec:       vBaseSpec,
		Status:     changedIngressStatus,
	}
	backwardUpdateIngress := &networkingv1beta1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec: networkingv1beta1.IngressSpec{
			IngressClassName: stringPointer("backwardsupdatedingressclass"),
		},
		Status: changedIngressStatus,
	}
	backwardNoUpdateIngress := &networkingv1beta1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec:       networkingv1beta1.IngressSpec{},
	}
	backwardUpdatedIngress := &networkingv1beta1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec: networkingv1beta1.IngressSpec{
			Backend:   vBaseSpec.Backend,
			IngressClassName: stringPointer("backwardsupdatedingressclass"),
			Rules:            vBaseSpec.Rules,
			TLS:              vBaseSpec.TLS,
		},
		Status: changedIngressStatus,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create forward",
			InitialVirtualState: []runtime.Object{baseIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Forward(ctx, baseIngress.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{&networkingv1beta1.Ingress{
				ObjectMeta: vObjectMeta,
				Spec:       vBaseSpec,
			}},
			InitialPhysicalState: []runtime.Object{&networkingv1beta1.Ingress{
				ObjectMeta: pObjectMeta,
				Spec: networkingv1beta1.IngressSpec{},
			}},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {&networkingv1beta1.Ingress{
					ObjectMeta: vObjectMeta,
					Spec:       vBaseSpec,
				}},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {&networkingv1beta1.Ingress{
					ObjectMeta: pObjectMeta,
					Spec:       pBaseSpec,
				}},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				pIngress := &networkingv1beta1.Ingress{
					ObjectMeta: pObjectMeta,
					Spec: networkingv1beta1.IngressSpec{},
				}
				pIngress.ResourceVersion = "999"

				_, err := syncer.Update(ctx, pIngress, &networkingv1beta1.Ingress{
					ObjectMeta: vObjectMeta,
					Spec:       vBaseSpec,
				}, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				vIngress := noUpdateIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.Update(ctx, createdIngress.DeepCopy(), vIngress, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backwards",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {backwardUpdatedIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				backwardUpdateIngress := backwardUpdateIngress.DeepCopy()
				vIngress := baseIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.Update(ctx, backwardUpdateIngress, vIngress, log)
				if err != nil {
					t.Fatal(err)
				}

				err = vClient.Get(ctx, types.NamespacedName{Namespace: vIngress.Namespace, Name: vIngress.Name}, vIngress)
				if err != nil {
					t.Fatal(err)
				}

				err = pClient.Get(ctx, types.NamespacedName{Namespace: backwardUpdateIngress.Namespace, Name: backwardUpdateIngress.Name}, backwardUpdateIngress)
				if err != nil {
					t.Fatal(err)
				}

				_, err = syncer.Update(ctx, backwardUpdateIngress, vIngress, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backwards not needed",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1beta1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				pIngress := backwardNoUpdateIngress.DeepCopy()
				pIngress.ResourceVersion = "999"

				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, pIngress, baseIngress.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}

func stringPointer(str string) *string {
	return &str
}
