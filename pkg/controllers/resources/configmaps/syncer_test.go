package configmaps

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
			Name:      translate.Default.HostName(nil, baseConfigMap.Name, baseConfigMap.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          baseConfigMap.Name,
				translate.NamespaceAnnotation:     baseConfigMap.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("ConfigMap").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, baseConfigMap.Name, baseConfigMap.Namespace).Name,
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

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Unused config map",
			InitialVirtualState: []runtime.Object{
				baseConfigMap,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ConfigMap"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*configMapSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseConfigMap))
				assert.NilError(t, err)
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
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*configMapSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseConfigMap))
				assert.NilError(t, err)
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
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*configMapSyncer).Sync(syncCtx, synccontext.NewSyncEvent(syncedConfigMap, updatedConfigMap))
				assert.NilError(t, err)
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
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*configMapSyncer).Sync(syncCtx, synccontext.NewSyncEvent(syncedConfigMap, updatedConfigMap))
				assert.NilError(t, err)
			},
		},
	})
}
