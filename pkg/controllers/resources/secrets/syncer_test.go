package secrets

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, syncer.Object) {
	return generictesting.FakeStartSyncer(t, ctx, func(ctx *synccontext.RegisterContext) (syncer.Object, error) {
		return NewSyncer(ctx, false)
	})
}

func TestSync(t *testing.T) {
	testLabel := "test-label"
	testLabelValue := "label-value"
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test",
			Labels: map[string]string{
				testLabel: testLabelValue,
			},
		},
	}
	updatedSecret := &corev1.Secret{
		ObjectMeta: baseSecret.ObjectMeta,
		Data: map[string][]byte{
			"test": []byte("test"),
		},
	}
	syncedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(baseSecret.Name, baseSecret.Namespace),
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:      baseSecret.Name,
				translate.NamespaceAnnotation: baseSecret.Namespace,
			},
			Labels: map[string]string{
				translate.NamespaceLabel:             baseSecret.Namespace,
				testLabel:                            testLabelValue,
				translate.ConvertLabelKey(testLabel): testLabelValue,
			},
		},
	}
	updatedSyncedSecret := &corev1.Secret{
		ObjectMeta: syncedSecret.ObjectMeta,
		Data:       updatedSecret.Data,
	}
	basePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: baseSecret.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "test",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: baseSecret.Name,
						},
					},
				},
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name: "Unused secret",
			InitialVirtualState: []runtime.Object{
				baseSecret,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Secret"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).SyncDown(syncContext, baseSecret)
				assert.NilError(t, err)
			},
		},
		{
			Name: "Used secret",
			InitialVirtualState: []runtime.Object{
				baseSecret,
				basePod,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					syncedSecret,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.SyncLabels = []string{testLabel}
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).SyncDown(syncContext, baseSecret)
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update used secret",
			InitialVirtualState: []runtime.Object{
				updatedSecret,
				basePod,
			},
			InitialPhysicalState: []runtime.Object{
				syncedSecret,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Secret"): {
					updatedSyncedSecret,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Options.SyncLabels = []string{testLabel}
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).Sync(syncContext, syncedSecret, updatedSecret)
				assert.NilError(t, err)
			},
		},
		{
			Name: "Remove unused secret",
			InitialVirtualState: []runtime.Object{
				updatedSecret,
			},
			InitialPhysicalState: []runtime.Object{
				syncedSecret,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Secret"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).Sync(syncContext, syncedSecret, updatedSecret)
				assert.NilError(t, err)
			},
		},
	})
}

func TestMapping(t *testing.T) {
	// test ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					SecretName: "a",
				},
				{
					SecretName: "b",
				},
			},
		},
	}

	// test ingress mapping
	requests := mapIngresses(ingress)
	if len(requests) != 2 || requests[0].Name != "a" || requests[0].Namespace != "test" || requests[1].Name != "b" || requests[1].Namespace != "test" {
		t.Fatalf("Wrong secret requests returned: %#+v", requests)
	}

	// test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test",
					Env: []corev1.EnvVar{
						{
							Name: "test",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "a",
									},
								},
							},
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "test",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "b",
						},
					},
				},
			},
		},
	}
	requests = mapPods(pod)
	if len(requests) != 2 || requests[0].Name != "a" || requests[0].Namespace != "test" || requests[1].Name != "b" || requests[1].Namespace != "test" {
		t.Fatalf("Wrong pod requests returned: %#+v", requests)
	}
}
