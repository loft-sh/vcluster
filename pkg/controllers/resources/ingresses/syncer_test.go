package ingresses

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vBaseSpec := networkingv1.IngressSpec{
		DefaultBackend: &networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: "testservice",
			},
			Resource: &corev1.TypedLocalObjectReference{
				Name: "testbackendresource",
			},
		},
		Rules: []networkingv1.IngressRule{
			{
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "testbackendservice",
									},
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
		TLS: []networkingv1.IngressTLS{
			{
				SecretName: "testtlssecret",
			},
		},
	}
	pBaseSpec := networkingv1.IngressSpec{
		DefaultBackend: &networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{
				Name: translate.Default.PhysicalName("testservice", "test"),
			},
			Resource: &corev1.TypedLocalObjectReference{
				Name: translate.Default.PhysicalName("testbackendresource", "test"),
			},
		},
		Rules: []networkingv1.IngressRule{
			{
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: translate.Default.PhysicalName("testbackendservice", "test"),
									},
									Resource: &corev1.TypedLocalObjectReference{
										Name: translate.Default.PhysicalName("testbackendresource", "test"),
									},
								},
							},
						},
					},
				},
			},
		},
		TLS: []networkingv1.IngressTLS{
			{
				SecretName: translate.Default.PhysicalName("testtlssecret", "test"),
			},
		},
	}
	changedIngressStatus := networkingv1.IngressStatus{
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
		Name:      "testingress",
		Namespace: "test",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.PhysicalName("testingress", "test"),
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:      vObjectMeta.Name,
			translate.NamespaceAnnotation: vObjectMeta.Namespace,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.Suffix,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	baseIngress := &networkingv1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec:       vBaseSpec,
	}
	createdIngress := &networkingv1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec:       pBaseSpec,
	}
	noUpdateIngress := &networkingv1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec:       vBaseSpec,
		Status:     changedIngressStatus,
	}
	backwardUpdateIngress := &networkingv1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPointer("backwardsupdatedingressclass"),
		},
		Status: changedIngressStatus,
	}
	pBackwardUpdatedIngress := &networkingv1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec:       pBaseSpec,
		Status:     changedIngressStatus,
	}
	pBackwardUpdatedIngress.Spec.IngressClassName = stringPointer("backwardsupdatedingressclass")
	backwardNoUpdateIngress := &networkingv1.Ingress{
		ObjectMeta: pObjectMeta,
		Spec:       networkingv1.IngressSpec{},
	}
	backwardUpdatedIngress := &networkingv1.Ingress{
		ObjectMeta: vObjectMeta,
		Spec: networkingv1.IngressSpec{
			DefaultBackend:   vBaseSpec.DefaultBackend,
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
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*ingressSyncer).SyncDown(syncCtx, baseIngress.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update forward",
			InitialVirtualState: []runtime.Object{&networkingv1.Ingress{
				ObjectMeta: vObjectMeta,
				Spec:       *vBaseSpec.DeepCopy(),
			}},
			InitialPhysicalState: []runtime.Object{&networkingv1.Ingress{
				ObjectMeta: pObjectMeta,
				Spec:       networkingv1.IngressSpec{},
			}},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {&networkingv1.Ingress{
					ObjectMeta: vObjectMeta,
					Spec:       *vBaseSpec.DeepCopy(),
				}},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {&networkingv1.Ingress{
					ObjectMeta: pObjectMeta,
					Spec:       *pBaseSpec.DeepCopy(),
				}},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)
				pIngress := &networkingv1.Ingress{
					ObjectMeta: pObjectMeta,
					Spec:       networkingv1.IngressSpec{},
				}
				pIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, pIngress, &networkingv1.Ingress{
					ObjectMeta: vObjectMeta,
					Spec:       *vBaseSpec.DeepCopy(),
				})
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)
				vIngress := noUpdateIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, createdIngress.DeepCopy(), vIngress)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backwards",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{backwardUpdateIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {backwardUpdatedIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {pBackwardUpdatedIngress.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)
				backwardUpdateIngress := backwardUpdateIngress.DeepCopy()
				vIngress := baseIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, backwardUpdateIngress, vIngress)
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx.Context, types.NamespacedName{Namespace: vIngress.Namespace, Name: vIngress.Name}, vIngress)
				assert.NilError(t, err)

				err = syncCtx.PhysicalClient.Get(syncCtx.Context, types.NamespacedName{Namespace: backwardUpdateIngress.Namespace, Name: backwardUpdateIngress.Name}, backwardUpdateIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, backwardUpdateIngress, vIngress)
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx.Context, types.NamespacedName{Namespace: vIngress.Namespace, Name: vIngress.Name}, vIngress)
				assert.NilError(t, err)

				err = syncCtx.PhysicalClient.Get(syncCtx.Context, types.NamespacedName{Namespace: backwardUpdateIngress.Namespace, Name: backwardUpdateIngress.Name}, backwardUpdateIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, backwardUpdateIngress, vIngress)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backwards not needed",
			InitialVirtualState:  []runtime.Object{baseIngress.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdIngress.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {baseIngress.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {createdIngress.DeepCopy()},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				pIngress := backwardNoUpdateIngress.DeepCopy()
				pIngress.ResourceVersion = "999"

				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*ingressSyncer).Sync(syncCtx, pIngress, baseIngress.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name: "Translate annotation",
			InitialVirtualState: []runtime.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      baseIngress.Name,
						Namespace: baseIngress.Namespace,
						Labels:    baseIngress.Labels,
						Annotations: map[string]string{
							"nginx.ingress.kubernetes.io/auth-secret": "my-secret",
						},
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      createdIngress.Name,
						Namespace: createdIngress.Namespace,
						Labels:    createdIngress.Labels,
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      baseIngress.Name,
							Namespace: baseIngress.Namespace,
							Labels:    baseIngress.Labels,
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/auth-secret": "my-secret",
							},
						},
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      createdIngress.Name,
							Namespace: createdIngress.Namespace,
							Labels:    createdIngress.Labels,
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/auth-secret": translate.Default.PhysicalName("my-secret", baseIngress.Namespace),
								"vcluster.loft.sh/managed-annotations":    "nginx.ingress.kubernetes.io/auth-secret",
								"vcluster.loft.sh/object-name":            baseIngress.Name,
								"vcluster.loft.sh/object-namespace":       baseIngress.Namespace,
							},
						},
					},
				},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)

				vIngress := &networkingv1.Ingress{}
				err := syncCtx.VirtualClient.Get(syncCtx.Context, types.NamespacedName{Name: baseIngress.Name, Namespace: baseIngress.Namespace}, vIngress)
				assert.NilError(t, err)

				pIngress := &networkingv1.Ingress{}
				err = syncCtx.PhysicalClient.Get(syncCtx.Context, types.NamespacedName{Name: createdIngress.Name, Namespace: createdIngress.Namespace}, pIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, pIngress, vIngress)
				assert.NilError(t, err)
			},
		},
		{
			Name: "Translate annotation with alb annotations",
			InitialVirtualState: []runtime.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      baseIngress.Name,
						Namespace: baseIngress.Namespace,
						Labels:    baseIngress.Labels,
						Annotations: map[string]string{
							"nginx.ingress.kubernetes.io/auth-secret":       "my-secret",
							"alb.ingress.kubernetes.io/actions.testservice": "{\"forwardConfig\":{\"targetGroups\":[{\"serviceName\":\"nginx-service\",\"servicePort\":\"80\",\"weight\":100}]}}",
						},
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      createdIngress.Name,
						Namespace: createdIngress.Namespace,
						Labels:    createdIngress.Labels,
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      baseIngress.Name,
							Namespace: baseIngress.Namespace,
							Labels:    baseIngress.Labels,
							Annotations: map[string]string{
								"alb.ingress.kubernetes.io/actions.testservice": "{\"forwardConfig\":{\"targetGroups\":[{\"serviceName\":\"nginx-service\",\"servicePort\":\"80\",\"weight\":100}]}}",
								"nginx.ingress.kubernetes.io/auth-secret":       "my-secret",
							},
						},
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				networkingv1.SchemeGroupVersion.WithKind("Ingress"): {
					&networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      createdIngress.Name,
							Namespace: createdIngress.Namespace,
							Labels:    createdIngress.Labels,
							Annotations: map[string]string{
								"vcluster.loft.sh/managed-annotations":                          "alb.ingress.kubernetes.io/actions.testservice-x-test-x-suffix\nnginx.ingress.kubernetes.io/auth-secret",
								"nginx.ingress.kubernetes.io/auth-secret":                       translate.Default.PhysicalName("my-secret", baseIngress.Namespace),
								"vcluster.loft.sh/object-name":                                  baseIngress.Name,
								"vcluster.loft.sh/object-namespace":                             baseIngress.Namespace,
								"alb.ingress.kubernetes.io/actions.testservice-x-test-x-suffix": "{\"forwardConfig\":{\"targetGroups\":[{\"serviceName\":\"nginx-service-x-test-x-suffix\",\"servicePort\":\"80\",\"weight\":100}]}}",
							},
						},
					},
				},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, registerContext, NewSyncer)

				vIngress := &networkingv1.Ingress{}
				err := syncCtx.VirtualClient.Get(syncCtx.Context, types.NamespacedName{Name: baseIngress.Name, Namespace: baseIngress.Namespace}, vIngress)
				assert.NilError(t, err)

				pIngress := &networkingv1.Ingress{}
				err = syncCtx.PhysicalClient.Get(syncCtx.Context, types.NamespacedName{Name: createdIngress.Name, Namespace: createdIngress.Namespace}, pIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, pIngress, vIngress)
				assert.NilError(t, err)
			},
		},
	})
}

func stringPointer(str string) *string {
	return &str
}
