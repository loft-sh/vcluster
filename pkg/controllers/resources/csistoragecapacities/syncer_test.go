package csistoragecapacities

import (
	"testing"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
)

const kind = "CSIStorageCapacity"

func TestSync(t *testing.T) {

	pObjectMeta := metav1.ObjectMeta{
		Name: "test-csistoragecapacity",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name: "test-csistoragecapacity",
	}

	pObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(10000, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	vObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(10000, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	pObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "b",
		Capacity:          resource.NewQuantity(200000, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(2000, resource.BinarySI),
	}

	vObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "b",
		Capacity:          resource.NewQuantity(200000, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(2000, resource.BinarySI),
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csistoragecapacitySyncer).SyncUp(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csistoragecapacitySyncer).SyncDown(syncCtx, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csistoragecapacitySyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
	})
}
