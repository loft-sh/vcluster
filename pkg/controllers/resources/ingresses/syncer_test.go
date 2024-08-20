package ingresses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/types"

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
				Name: translate.Default.HostName(nil, "testservice", "test").Name,
			},
			Resource: &corev1.TypedLocalObjectReference{
				Name: translate.Default.HostName(nil, "testbackendresource", "test").Name,
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
										Name: translate.Default.HostName(nil, "testbackendservice", "test").Name,
									},
									Resource: &corev1.TypedLocalObjectReference{
										Name: translate.Default.HostName(nil, "testbackendresource", "test").Name,
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
				SecretName: translate.Default.HostName(nil, "testtlssecret", "test").Name,
			},
		},
	}
	changedIngressStatus := networkingv1.IngressStatus{
		LoadBalancer: networkingv1.IngressLoadBalancerStatus{
			Ingress: []networkingv1.IngressLoadBalancerIngress{
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
		Name:      translate.Default.HostName(nil, "testingress", "test").Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          networkingv1.SchemeGroupVersion.WithKind("Ingress").String(),
			translate.HostNamespaceAnnotation: "test",
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testingress", "test").Name,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
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

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.Ingresses.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*ingressSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseIngress.DeepCopy()))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				pIngress := &networkingv1.Ingress{
					ObjectMeta: pObjectMeta,
					Spec:       networkingv1.IngressSpec{},
				}
				pIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pIngress, &networkingv1.Ingress{
					ObjectMeta: vObjectMeta,
					Spec:       *vBaseSpec.DeepCopy(),
				}))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				vIngress := noUpdateIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEvent(createdIngress.DeepCopy(), vIngress))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				backwardUpdateIngress := backwardUpdateIngress.DeepCopy()
				vIngress := baseIngress.DeepCopy()
				vIngress.ResourceVersion = "999"

				_, err := syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(backwardUpdateIngress, vIngress, synccontext.SyncEventSourceHost))
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Namespace: vIngress.Namespace, Name: vIngress.Name}, vIngress)
				assert.NilError(t, err)

				err = syncCtx.PhysicalClient.Get(syncCtx, types.NamespacedName{Namespace: backwardUpdateIngress.Namespace, Name: backwardUpdateIngress.Name}, backwardUpdateIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(backwardUpdateIngress, vIngress, synccontext.SyncEventSourceHost))
				assert.NilError(t, err)

				err = syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Namespace: vIngress.Namespace, Name: vIngress.Name}, vIngress)
				assert.NilError(t, err)

				err = syncCtx.PhysicalClient.Get(syncCtx, types.NamespacedName{Namespace: backwardUpdateIngress.Namespace, Name: backwardUpdateIngress.Name}, backwardUpdateIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEventWithSource(backwardUpdateIngress, vIngress, synccontext.SyncEventSourceHost))
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

				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)
				_, err := syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pIngress, baseIngress.DeepCopy()))
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
							"nginx.ingress.kubernetes.io/auth-secret":     "my-secret",
							"nginx.ingress.kubernetes.io/auth-tls-secret": baseIngress.Namespace + "/my-secret",
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
								"nginx.ingress.kubernetes.io/auth-secret":     "my-secret",
								"nginx.ingress.kubernetes.io/auth-tls-secret": baseIngress.Namespace + "/my-secret",
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
								"nginx.ingress.kubernetes.io/auth-secret":     translate.Default.HostName(nil, "my-secret", baseIngress.Namespace).Name,
								"nginx.ingress.kubernetes.io/auth-tls-secret": createdIngress.Namespace + "/" + translate.Default.HostName(nil, "my-secret", baseIngress.Namespace).Name,
								"vcluster.loft.sh/managed-annotations":        "nginx.ingress.kubernetes.io/auth-secret\nnginx.ingress.kubernetes.io/auth-tls-secret",
								"vcluster.loft.sh/object-name":                baseIngress.Name,
								"vcluster.loft.sh/object-namespace":           baseIngress.Namespace,
								translate.UIDAnnotation:                       "",
								translate.KindAnnotation:                      networkingv1.SchemeGroupVersion.WithKind("Ingress").String(),
								translate.HostNamespaceAnnotation:             createdIngress.Namespace,
								translate.HostNameAnnotation:                  createdIngress.Name,
							},
						},
					},
				},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)

				vIngress := &networkingv1.Ingress{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Name: baseIngress.Name, Namespace: baseIngress.Namespace}, vIngress)
				assert.NilError(t, err)

				pIngress := &networkingv1.Ingress{}
				err = syncCtx.PhysicalClient.Get(syncCtx, types.NamespacedName{Name: createdIngress.Name, Namespace: createdIngress.Namespace}, pIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pIngress, vIngress))
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
							"nginx.ingress.kubernetes.io/auth-secret":        "my-secret",
							"alb.ingress.kubernetes.io/actions.testservice":  `{"forwardConfig":{"targetGroups":[{"serviceName":"nginx-service","servicePort":"80","weight":100}]}}`,
							"alb.ingress.kubernetes.io/actions.ssl-redirect": `{"type": "redirect", "redirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}`,
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
								"alb.ingress.kubernetes.io/actions.testservice":  `{"forwardConfig":{"targetGroups":[{"serviceName":"nginx-service","servicePort":"80","weight":100}]}}`,
								"alb.ingress.kubernetes.io/actions.ssl-redirect": `{"type": "redirect", "redirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}`,
								"nginx.ingress.kubernetes.io/auth-secret":        "my-secret",
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
								"vcluster.loft.sh/managed-annotations":                           "alb.ingress.kubernetes.io/actions.ssl-redirect-x-test-x-suffix\nalb.ingress.kubernetes.io/actions.testservice-x-test-x-suffix\nnginx.ingress.kubernetes.io/auth-secret",
								"nginx.ingress.kubernetes.io/auth-secret":                        translate.Default.HostName(nil, "my-secret", baseIngress.Namespace).Name,
								"vcluster.loft.sh/object-name":                                   baseIngress.Name,
								"vcluster.loft.sh/object-namespace":                              baseIngress.Namespace,
								translate.UIDAnnotation:                                          "",
								translate.KindAnnotation:                                         networkingv1.SchemeGroupVersion.WithKind("Ingress").String(),
								translate.HostNamespaceAnnotation:                                createdIngress.Namespace,
								translate.HostNameAnnotation:                                     createdIngress.Name,
								"alb.ingress.kubernetes.io/actions.testservice-x-test-x-suffix":  `{"forwardConfig":{"targetGroups":[{"serviceName":"nginx-service-x-test-x-suffix","servicePort":"80","weight":100}]}}`,
								"alb.ingress.kubernetes.io/actions.ssl-redirect-x-test-x-suffix": `{"redirectConfig":{"Port":"443","Protocol":"HTTPS","StatusCode":"HTTP_301"},"type":"redirect","forwardConfig":{}}`,
							},
						},
					},
				},
			},
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, registerContext, NewSyncer)

				vIngress := &networkingv1.Ingress{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{Name: baseIngress.Name, Namespace: baseIngress.Namespace}, vIngress)
				assert.NilError(t, err)

				pIngress := &networkingv1.Ingress{}
				err = syncCtx.PhysicalClient.Get(syncCtx, types.NamespacedName{Name: createdIngress.Name, Namespace: createdIngress.Namespace}, pIngress)
				assert.NilError(t, err)

				_, err = syncer.(*ingressSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pIngress, vIngress))
				assert.NilError(t, err)
			},
		},
	})
}

func stringPointer(str string) *string {
	return &str
}
