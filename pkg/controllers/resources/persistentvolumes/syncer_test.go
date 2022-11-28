package persistentvolumes

import (
	"testing"
	"time"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/api/resource"

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
		return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
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
		Name:            translate.Default.PhysicalName("testpvc", "test"),
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
			HostClusterPersistentVolumeAnnotation: "testpv",
		},
	}
	basePvWithDelTSObjectMeta := metav1.ObjectMeta{
		Name:              "testpv",
		DeletionTimestamp: &metav1.Time{Time: time.Now()},
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
	backwardRetainPPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			ClaimRef: &corev1.ObjectReference{
				Name:      "retainPVC-x-test-x-suffix",
				Namespace: "test",
			},
			StorageClassName: "vcluster-retainSC-x-test-x-suffix",
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeReleased,
		},
	}
	backwardRetainInitialVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			ClaimRef: &corev1.ObjectReference{
				Name:      "retainPVC",
				Namespace: "test",
			},
			StorageClassName: "retainSC",
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	backwardRetainedVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			ClaimRef: &corev1.ObjectReference{
				Name:      "retainPVC",
				Namespace: "test",
			},
			StorageClassName: "retainSC",
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeReleased,
		},
	}
	backwardDeletePPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			ClaimRef: &corev1.ObjectReference{
				Name:      "deletedPVC",
				Namespace: "test",
			},
		},
	}
	backwardDeleteVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			ClaimRef: &corev1.ObjectReference{
				Name:      "deletedPVC",
				Namespace: "test",
			},
		},
	}
	backwardDeleteVPvwithDelTS := &corev1.PersistentVolume{
		ObjectMeta: basePvWithDelTSObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			ClaimRef: &corev1.ObjectReference{
				Name:      "deletedPVC",
				Namespace: "test",
			},
		},
	}
	pPVforDeletePVWithoutClaim := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		},
	}
	vPVforDeletePVWithoutClaim := &corev1.PersistentVolume{
		ObjectMeta: basePvWithDelTSObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
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
		{
			Name: "Sync PV Size",
			InitialVirtualState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: basePvc.ObjectMeta,
				},
				&corev1.PersistentVolume{
					ObjectMeta: baseVPv.ObjectMeta,
					Spec: corev1.PersistentVolumeSpec{
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
						ClaimRef: baseVPv.Spec.ClaimRef,
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&corev1.PersistentVolume{
					ObjectMeta: basePPv.ObjectMeta,
					Spec: corev1.PersistentVolumeSpec{
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("20Gi"),
						},
						ClaimRef: basePPv.Spec.ClaimRef,
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {
					&corev1.PersistentVolume{
						ObjectMeta: baseVPv.ObjectMeta,
						Spec: corev1.PersistentVolumeSpec{
							Capacity: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("20Gi"),
							},
							ClaimRef: baseVPv.Spec.ClaimRef,
						},
					},
				},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: basePvc.ObjectMeta,
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {
					&corev1.PersistentVolume{
						ObjectMeta: basePPv.ObjectMeta,
						Spec: corev1.PersistentVolumeSpec{
							Capacity: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("20Gi"),
							},
							ClaimRef: basePPv.Spec.ClaimRef,
						},
					},
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)

				vPv := &corev1.PersistentVolume{}
				err := syncContext.VirtualClient.Get(ctx.Context, types.NamespacedName{Name: baseVPv.Name}, vPv)
				assert.NilError(t, err)

				pPv := &corev1.PersistentVolume{}
				err = syncContext.PhysicalClient.Get(ctx.Context, types.NamespacedName{Name: basePPv.Name}, pPv)
				assert.NilError(t, err)

				_, err = syncer.Sync(syncContext, pPv, vPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Retain PV and update PV Status when reclaim policy is Retain",
			InitialVirtualState:  []runtime.Object{backwardRetainInitialVPv},
			InitialPhysicalState: []runtime.Object{backwardRetainPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardRetainedVPv},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardRetainPPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				backwardRetainPPv := backwardRetainPPv.DeepCopy()
				backwardRetainInitialVPv := backwardRetainInitialVPv.DeepCopy()
				_, err := syncer.Sync(syncContext, backwardRetainPPv, backwardRetainInitialVPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete PV when reclaim policy is Delete",
			InitialVirtualState:  []runtime.Object{backwardDeleteVPv},
			InitialPhysicalState: []runtime.Object{backwardDeletePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardDeletePPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				backwardDeletePPv := backwardDeletePPv.DeepCopy()
				backwardDeleteVPv := backwardDeleteVPv.DeepCopy()
				_, err := syncer.Sync(syncContext, backwardDeletePPv, backwardDeleteVPv)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Wait for the VPV to be deleted, when reclaim policy is Delete",
			InitialVirtualState:  []runtime.Object{backwardDeleteVPvwithDelTS},
			InitialPhysicalState: []runtime.Object{backwardDeletePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardDeleteVPvwithDelTS},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardDeletePPv},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				backwardDeletePPv := backwardDeletePPv.DeepCopy()
				backwardDeleteVPvwithDelTS := backwardDeleteVPvwithDelTS.DeepCopy()
				_, err := syncer.Sync(syncContext, backwardDeletePPv, backwardDeleteVPvwithDelTS)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete PPV without an associated PVC",
			InitialVirtualState:  []runtime.Object{vPVforDeletePVWithoutClaim},
			InitialPhysicalState: []runtime.Object{pPVforDeletePVWithoutClaim},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {vPVforDeletePVWithoutClaim},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, ctx)
				vPVforDeletePVWithoutClaim := vPVforDeletePVWithoutClaim.DeepCopy()
				pPVforDeletePVWithoutClaim := pPVforDeletePVWithoutClaim.DeepCopy()
				_, err := syncer.Sync(syncContext, pPVforDeletePVWithoutClaim, vPVforDeletePVWithoutClaim)
				assert.NilError(t, err)
			},
		},
	})
}
