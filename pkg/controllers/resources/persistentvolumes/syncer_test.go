package persistentvolumes

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *persistentVolumeSyncer) {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
	assert.NilError(t, err)

	syncContext, object := generictesting.FakeStartSyncer(t, ctx, NewSyncer)
	return syncContext, object.(*persistentVolumeSyncer)
}

func TestSync(t *testing.T) {
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testpvc",
			Namespace: "test",
		},
	}
	basePPvcReference := &corev1.ObjectReference{
		Name:            translate.PhysicalName("testpvc", "test"),
		Namespace:       "test",
		ResourceVersion: generictesting.FakeClientResourceVersion,
	}
	baseVPvcReference := &corev1.ObjectReference{
		Name:            "testpvc",
		Namespace:       "test",
		ResourceVersion: generictesting.FakeClientResourceVersion,
	}
	basePvObjectMeta := metav1.ObjectMeta{
		Name: "testpv",
		Annotations: map[string]string{
			"vcluster.loft.sh/host-pv": "testpv",
		},
	}
	basePPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: basePPvcReference,
		},
	}
	baseVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: baseVPvcReference,
		},
	}
	wrongNsPPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Name:            "testpvc",
				Namespace:       "wrong",
				ResourceVersion: generictesting.FakeClientResourceVersion,
			},
		},
	}
	noPvcPPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Name:      "wrong",
				Namespace: "test",
			},
		},
	}
	backwardUpdatePPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef:         basePPvcReference,
			StorageClassName: "someStorageClass",
		},
		Status: corev1.PersistentVolumeStatus{
			Message: "someMessage",
		},
	}
	backwardUpdateVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef:         baseVPvcReference,
			StorageClassName: "someStorageClass",
		},
		Status: corev1.PersistentVolumeStatus{
			Message: "someMessage",
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create Backward",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{basePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {baseVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {basePPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncUp(syncContext, basePPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Don't Create Backward, wrong physical namespace",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{wrongNsPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {wrongNsPPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncUp(syncContext, wrongNsPPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Don't Create Backward, no virtual pvc",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{noPvcPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {noPvcPPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncUp(syncContext, noPvcPPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update Backward",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{backwardUpdatePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {backwardUpdateVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardUpdatePPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				backwardUpdatePPv := backwardUpdatePPv.DeepCopy()
				baseVPv := baseVPv.DeepCopy()
				_, err := syncer.Sync(syncContext, backwardUpdatePPv, baseVPv)
				assert.NilError(t, err)

				err = syncContext.VirtualClient.Get(ctx.Context, types.NamespacedName{Name: baseVPv.Name}, baseVPv)
				assert.NilError(t, err)

				err = syncContext.PhysicalClient.Get(ctx.Context, types.NamespacedName{Name: backwardUpdatePPv.Name}, backwardUpdatePPv)
				assert.NilError(t, err)

				_, err = syncer.Sync(syncContext, backwardUpdatePPv, baseVPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete Backward by update backward",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{noPvcPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {noPvcPPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncContext, noPvcPPv, baseVPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete Backward not needed",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{basePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {baseVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {basePPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncContext, basePPv, baseVPv)
				assert.NilError(t, err)
			},
		},
	})
}
