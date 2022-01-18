package storageclasses

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	baseObjectMeta := metav1.ObjectMeta{
		Name:      "testsc",
		Namespace: "testns",
	}
	baseSc := &v1.StorageClass{
		ObjectMeta: baseObjectMeta,
	}
	updateSc := &v1.StorageClass{
		ObjectMeta:  baseObjectMeta,
		Provisioner: "someProvisioner",
	}
	updateSc.Labels = map[string]string{
		"a": "b",
	}
	updatedSc := &v1.StorageClass{
		ObjectMeta:  baseObjectMeta,
		Provisioner: "someProvisioner",
	}
	updatedSc.Labels = map[string]string{
		"a": "b",
	}
	noUpdateSc := &v1.StorageClass{
		ObjectMeta: baseObjectMeta,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create backward",
			InitialPhysicalState: []runtime.Object{baseSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*storageClassSyncer).SyncUp(syncCtx, baseSc)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward",
			InitialVirtualState:  []runtime.Object{baseSc},
			InitialPhysicalState: []runtime.Object{updateSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {updatedSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {updateSc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*storageClassSyncer).Sync(syncCtx, updateSc, baseSc)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "No Update backward",
			InitialVirtualState:  []runtime.Object{baseSc},
			InitialPhysicalState: []runtime.Object{noUpdateSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {noUpdateSc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*storageClassSyncer).Sync(syncCtx, noUpdateSc, baseSc)
				assert.NilError(t, err)
			},
		},
	})
}
