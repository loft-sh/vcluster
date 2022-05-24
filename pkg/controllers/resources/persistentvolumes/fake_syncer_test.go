package persistentvolumes

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func newFakeFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *fakePersistentVolumeSyncer) {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.PersistentVolumeClaim)
		return []string{pod.Spec.VolumeName}
	})
	assert.NilError(t, err)

	syncContext, object := generictesting.FakeStartSyncer(t, ctx, NewFakeSyncer)
	return syncContext, object.(*fakePersistentVolumeSyncer)
}

func TestFakeSync(t *testing.T) {
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testpvc",
			Namespace:       "testns",
			ResourceVersion: generictesting.FakeClientResourceVersion,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName:       "mypv",
			StorageClassName: stringPointer("mystorageclass"),
		},
	}
	basePvName := types.NamespacedName{
		Name:      "mypv",
		Namespace: "testns",
	}
	basePv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: basePvName.Name,
			Labels: map[string]string{
				"vcluster.loft.sh/fake-pv": "true",
			},
			Annotations: map[string]string{
				"kubernetes.io/createdby":              "fake-pv-provisioner",
				"pv.kubernetes.io/bound-by-controller": "true",
				"pv.kubernetes.io/provisioned-by":      "fake-pv-provisioner",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				FlexVolume: &corev1.FlexPersistentVolumeSource{
					Driver: "fake",
				},
			},
			Capacity:    basePvc.Spec.Resources.Requests,
			AccessModes: basePvc.Spec.AccessModes,
			ClaimRef: &corev1.ObjectReference{
				Kind:            "PersistentVolumeClaim",
				Namespace:       basePvc.Namespace,
				Name:            basePvc.Name,
				UID:             basePvc.UID,
				APIVersion:      corev1.SchemeGroupVersion.Version,
				ResourceVersion: basePvc.ResourceVersion,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              *basePvc.Spec.StorageClassName,
			VolumeMode:                    basePvc.Spec.VolumeMode,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	pvWithFinalizers := basePv.DeepCopy()
	pvWithFinalizers.Finalizers = []string{"myfinalizer"}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create",
			InitialVirtualState: []runtime.Object{basePvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {basePv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeFakeSyncer(t, ctx)
				_, err := syncer.FakeSyncUp(syncContext, basePvName)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create not needed",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeFakeSyncer(t, ctx)
				_, err := syncer.FakeSyncUp(syncContext, basePvName)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Delete",
			InitialVirtualState: []runtime.Object{pvWithFinalizers},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeFakeSyncer(t, ctx)
				_, err := syncer.FakeSync(syncContext, pvWithFinalizers)
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Delete not existent pv",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := newFakeFakeSyncer(t, ctx)
				_, err := syncer.FakeSync(syncContext, basePv)
				assert.NilError(t, err)
			},
		},
	})
}

func stringPointer(str string) *string {
	return &str
}
