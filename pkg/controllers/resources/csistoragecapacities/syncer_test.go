package csistoragecapacities

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
)

const kind = "CSIStorageCapacity"

func TestSyncLegacyStorageClass(t *testing.T) {

	pObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity",
		Namespace: "test",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity-x-test",
		Namespace: "kube-system",
		Annotations: map[string]string{
			"vcluster.loft.sh/object-name":      "test-csistoragecapacity",
			"vcluster.loft.sh/object-namespace": "test",
		},
		Labels: map[string]string{
			"vcluster.loft.sh/namespace": "test",
		},
	}

	pObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	vObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
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
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
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
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
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
				ctx.Controllers.Insert("legacy-storageclasses")
				ctx.Controllers.Delete("storageclasses")
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).SyncUp(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Insert("legacy-storageclasses")
				ctx.Controllers.Delete("storageclasses")
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).SyncDown(syncCtx, vObj)
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
				ctx.Controllers.Insert("legacy-storageclasses")
				ctx.Controllers.Delete("storageclasses")
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
	})
}

func TestSyncStorageClass(t *testing.T) {

	pObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity",
		Namespace: "test",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity-x-test",
		Namespace: "kube-system",
		Annotations: map[string]string{
			"vcluster.loft.sh/object-name":      "test-csistoragecapacity",
			"vcluster.loft.sh/object-namespace": "test",
		},
		Labels: map[string]string{
			"vcluster.loft.sh/namespace": "test",
		},
	}

	pObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "vcluster-a-x-test-x-suffix",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	vSCa := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a",
		},
	}
	vSCb := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "b",
		},
	}
	vObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
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
		StorageClassName:  "vcluster-b-x-test-x-suffix",
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
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
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{vSCa, vSCb},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Delete("legacy-storageclasses")
				ctx.Controllers.Insert("storageclasses")
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncUp(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync Up, corresponding storageclass missing",
			InitialVirtualState:  []runtime.Object{vSCb}, // corresponding one is vSCa
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			ExpectNotFoundVirtual: map[schema.GroupVersionKind][]types.NamespacedName{
				storagev1.SchemeGroupVersion.WithKind(kind): {{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Delete("legacy-storageclasses")
				ctx.Controllers.Insert("storageclasses")
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncUp(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj, vSCa, vSCb},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Delete("legacy-storageclasses")
				ctx.Controllers.Insert("storageclasses")
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncDown(syncCtx, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj, vSCa, vSCb},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Delete("legacy-storageclasses")
				ctx.Controllers.Insert("storageclasses")
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync, corresponding storageclass missing",
			InitialVirtualState:  []runtime.Object{vObj, vSCb}, // corresponding one is vSCa
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			ExpectNotFoundVirtual: map[schema.GroupVersionKind][]types.NamespacedName{
				storagev1.SchemeGroupVersion.WithKind(kind): {{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Controllers.Delete("legacy-storageclasses")
				ctx.Controllers.Insert("storageclasses")
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncDown(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObj, vObj)
				assert.NilError(t, err)
			},
		},
	})
}
