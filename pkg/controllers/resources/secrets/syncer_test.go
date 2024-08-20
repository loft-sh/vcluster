package secrets

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, syncer.Object) {
	return generictesting.FakeStartSyncer(t, ctx, func(ctx *synccontext.RegisterContext) (syncer.Object, error) {
		return NewSyncer(ctx)
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
			Name:      translate.Default.HostName(nil, baseSecret.Name, baseSecret.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          baseSecret.Name,
				translate.NamespaceAnnotation:     baseSecret.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, baseSecret.Name, baseSecret.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: baseSecret.Namespace,
				testLabel:                testLabelValue,
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
				_, err := syncer.(*secretSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(baseSecret))
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
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).SyncToHost(syncContext, synccontext.NewSyncToHostEvent(baseSecret))
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
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.(*secretSyncer).Sync(syncContext, synccontext.NewSyncEvent(syncedSecret, updatedSecret))
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
				_, err := syncer.(*secretSyncer).Sync(syncContext, synccontext.NewSyncEvent(syncedSecret, updatedSecret))
				assert.NilError(t, err)
			},
		},
	})
}
