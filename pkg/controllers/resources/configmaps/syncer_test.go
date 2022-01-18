package configmaps

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func newFakeSyncer(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	err := vClient.IndexField(ctx, &corev1.Pod{}, constants.IndexByConfigMap, indexPodByConfigmap)
	if err != nil {
		panic(err)
	}

	return &syncer{
		virtualClient: vClient,
		localClient:   pClient,

		creator:    generic.NewGenericCreator(pClient, &testingutil.FakeEventRecorder{}, "endpoints"),
		translator: translate.NewDefaultTranslator("test"),
	}
}

func TestSync(t *testing.T) {
	baseConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "test",
		},
	}
	updatedConfigMap := &corev1.ConfigMap{
		ObjectMeta: baseConfigMap.ObjectMeta,
		Data: map[string]string{
			"test": "test",
		},
	}
	syncedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.PhysicalName(baseConfigMap.Name, baseConfigMap.Namespace),
			Namespace: "test",
			Annotations: map[string]string{
				translator.NameAnnotation:      baseConfigMap.Name,
				translator.NamespaceAnnotation: baseConfigMap.Namespace,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: baseConfigMap.Namespace,
			},
		},
	}
	updatedSyncedConfigMap := &corev1.ConfigMap{
		ObjectMeta: syncedConfigMap.ObjectMeta,
		Data:       updatedConfigMap.Data,
	}
	basePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: baseConfigMap.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "test",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: baseConfigMap.Name,
							},
						},
					},
				},
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name: "Unused config map",
			InitialVirtualState: []runtime.Object{
				baseConfigMap,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Forward(ctx, baseConfigMap, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Used config map",
			InitialVirtualState: []runtime.Object{
				baseConfigMap,
				basePod,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {
					syncedConfigMap,
				},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Forward(ctx, baseConfigMap, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Update used config map",
			InitialVirtualState: []runtime.Object{
				updatedConfigMap,
				basePod,
			},
			InitialPhysicalState: []runtime.Object{
				syncedConfigMap,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {
					updatedSyncedConfigMap,
				},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Update(ctx, syncedConfigMap, updatedConfigMap, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Remove unused config map",
			InitialVirtualState: []runtime.Object{
				updatedConfigMap,
			},
			InitialPhysicalState: []runtime.Object{
				syncedConfigMap,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Update(ctx, syncedConfigMap, updatedConfigMap, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}

func TestMapping(t *testing.T) {
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
								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
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
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "b",
							},
						},
					},
				},
			},
		},
	}
	requests := mapPods(pod)
	if len(requests) != 2 || requests[0].Name != "a" || requests[0].Namespace != "test" || requests[1].Name != "b" || requests[1].Namespace != "test" {
		t.Fatalf("Wrong pod requests returned: %#+v", requests)
	}
}
